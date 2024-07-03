package main

import (
	"context"
	"flag"
	"log"
	"net"
	"strconv"
	"time"

	common "github.com/rikughi/commons"
	"github.com/rikughi/commons/broker"
	"github.com/rikughi/commons/discovery"
	"github.com/rikughi/commons/discovery/consul"
	"github.com/rikughi/omsv2-kitchen/gateway"
	"google.golang.org/grpc"
)

var (
	serviceName = "kitchen"
	port        = 2031
	consulAddr  = common.EnvString("CONSUL_ADDR", "localhost:8500")
	amqpUser    = common.EnvString("RABBITMQ_USER", "guest")
	amqpPass    = common.EnvString("RABBITMQ_PASS", "guest")
	amqpHost    = common.EnvString("RABBITMQ_HOST", "localhost")
	amqpPort    = common.EnvString("RABBITMQ_PORT", "5672")
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

	grpcServer := grpc.NewServer()

	l, err := net.Listen("tcp", grpcAddr)
	if err != nil {
		log.Fatal("failed to listen", err)
	}
	defer l.Close()

	gateway := gateway.NewGateway(registry)

	consumer := NewConsumer(gateway)
	go consumer.Listen(ch)

	log.Println("GRPC Server Started at ", grpcAddr)

	if err := grpcServer.Serve(l); err != nil {
		log.Fatal(err.Error())
	}
}
