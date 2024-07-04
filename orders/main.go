package main

import (
	"context"
	"flag"
	"fmt"
	"net"
	"strconv"
	"time"

	common "github.com/rikughi/commons"
	"github.com/rikughi/commons/broker"
	"github.com/rikughi/commons/discovery"
	"github.com/rikughi/commons/discovery/consul"
	"github.com/rikughi/omsv2-orders/gateway"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
	"go.uber.org/zap"
	"google.golang.org/grpc"
)

var (
	serviceName = "orders"
	port        = 2000
	consulAddr  = common.EnvString("CONSUL_ADDR", "localhost:8500")
	amqpUser    = common.EnvString("RABBITMQ_USER", "guest")
	amqpPass    = common.EnvString("RABBITMQ_PASS", "guest")
	amqpHost    = common.EnvString("RABBITMQ_HOST", "localhost")
	amqpPort    = common.EnvString("RABBITMQ_PORT", "5672")
	mongoUser   = common.EnvString("MONGO_DB_USER", "root")
	mongoPass   = common.EnvString("MONGO_DB_PASS", "example")
	mongoAddr   = common.EnvString("MONGO_DB_HOST", "localhost:27017")
	jaegerAddr  = common.EnvString("JAEGER_ADDR", "localhost:4318")
)

func main() {
	flag.IntVar(&port, "port", port, "GRPC Port")
	flag.Parse()

	logger, _ := zap.NewProduction()
	defer logger.Sync()

	zap.ReplaceGlobals(logger)

	if err := common.SetGlobalTracer(context.TODO(), serviceName, jaegerAddr); err != nil {
		logger.Fatal("could set global tracer", zap.Error(err))
	}

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
				logger.Fatal("failed to health check", zap.Error(err))
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

	// mongo db conn
	uri := fmt.Sprintf("mongodb://%s:%s@%s", mongoUser, mongoPass, mongoAddr)
	mongoClient, err := connectToMongoDB(uri)
	if err != nil {
		logger.Fatal("failed to connect to mongo db", zap.Error(err))
	}

	grpcServer := grpc.NewServer()

	l, err := net.Listen("tcp", grpcAddr)
	if err != nil {
		logger.Fatal("failed to listen: %v", zap.Error(err))
	}
	defer l.Close()

	gateway := gateway.NewGateway(registry)

	store := NewStore(mongoClient)
	svc := NewService(store, gateway)
	svcWithTelemetry := NewTelemetryMiddleware(svc)
	svcWithLogging := NewLoggingMiddleware(svcWithTelemetry)
	NewGRPCHandler(grpcServer, svcWithLogging, ch)

	consumer := NewConsumer(svc)
	go consumer.Listen(ch)

	logger.Info("GRPC Server Started at ", zap.String("port", grpcAddr))

	if err := grpcServer.Serve(l); err != nil {
		logger.Fatal("Failed to serve", zap.Error(err))
	}
}

func connectToMongoDB(uri string) (*mongo.Client, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	client, err := mongo.Connect(ctx, options.Client().ApplyURI(uri))
	if err != nil {
		return nil, err
	}

	err = client.Ping(ctx, readpref.Primary())
	return client, err
}
