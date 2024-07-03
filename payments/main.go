package main

import (
	"context"
	"flag"
	"log"
	"net"
	"net/http"
	"strconv"
	"time"

	common "github.com/rikughi/commons"
	"github.com/rikughi/commons/broker"
	"github.com/rikughi/commons/discovery"
	"github.com/rikughi/commons/discovery/consul"
	"github.com/rikughi/omsv2-payments/gateway"
	stripeProcessor "github.com/rikughi/omsv2-payments/processor/stripe"
	"github.com/stripe/stripe-go/v78"
	"google.golang.org/grpc"
)

var (
	serviceName          = "payment"
	port                 = 2021
	amqpUser             = common.EnvString("RABBITMQ_USER", "guest")
	amqpPass             = common.EnvString("RABBITMQ_PASS", "guest")
	amqpHost             = common.EnvString("RABBITMQ_HOST", "localhost")
	amqpPort             = common.EnvString("RABBITMQ_PORT", "5672")
	consulAddr           = common.EnvString("CONSUL_ADDR", "localhost:8500")
	stripeKey            = common.EnvString("STRIPE_KEY", "sk_test_51PXBh3CurvtCrm275UJEbk4QavAII53E5QMr6jY8vJJLTTmd80vOeKYIr6DDfDyCP4BSWhA2eztQtIjNdz29SC9u0001FHYET0")
	endpointStripeSecret = common.EnvString("STRIPE_ENDPOINT_SECRET", "whsec_72f8640ff44b07e2184c409369339b0c6688f0873997a26c1122c786797276e1")
	httpAddr             = common.EnvString("HTTP_ADDR", "localhost:8081")
)

func main() {
	flag.IntVar(&port, "port", port, "GRPC Port")
	flag.Parse()

	registry, err := consul.NewRegistery(consulAddr)
	if err != nil {
		panic(err)
	}

	ctx := context.Background()
	instanceID := discovery.GenerateInstanceID(serviceName)

	grpcAddr := "localhost:" + strconv.Itoa(port)
	if err := registry.Register(ctx, instanceID, serviceName, grpcAddr); err != nil {
		panic(err)
	}

	go func() {
		for {
			if err := registry.HealthCheck(instanceID, serviceName); err != nil {
				log.Fatal("failed to health check")
			}
			time.Sleep(time.Second * 1)
		}
	}()

	defer registry.Deregister(ctx, instanceID, serviceName)

	ch, close := broker.Connect(amqpUser, amqpPass, amqpHost, amqpPort)
	defer func() {
		close()
		ch.Close()
	}()

	stripe.Key = stripeKey
	stripeProcessor := stripeProcessor.NewProcessor()
	gateway := gateway.NewGateway(registry)
	svc := NewService(stripeProcessor, gateway)

	amqpConsumer := NewConsumer(svc)
	go amqpConsumer.Listen(ch)

	mux := http.NewServeMux()

	httpServer := NewPaymentHTTPHandler(ch)
	httpServer.registerRoutes(mux)

	go func() {
		log.Printf("Starting HTTP server at %s", httpAddr)
		if err := http.ListenAndServe(httpAddr, mux); err != nil {
			log.Fatal("failed to start http server")
		}
	}()

	grpcServer := grpc.NewServer()

	l, err := net.Listen("tcp", grpcAddr)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	defer l.Close()

	log.Println("GRPC Server Started at ", grpcAddr)

	if err := grpcServer.Serve(l); err != nil {
		log.Fatal(err.Error())
	}
}
