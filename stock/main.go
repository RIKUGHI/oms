package main

import (
	"context"
	"flag"
	"log"
	"net"
	"strconv"
	"time"

	common "github.com/rikughi/commons"
	"github.com/rikughi/commons/discovery"
	"github.com/rikughi/commons/discovery/consul"
	"google.golang.org/grpc"
)

var (
	serviceName = "stock"
	port        = 2011
	consulAddr  = common.EnvString("CONSUL_ADDR", "localhost:8500")
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

	grpcServer := grpc.NewServer()

	l, err := net.Listen("tcp", grpcAddr)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	defer l.Close()

	store := NewStore()
	svc := NewService(store)
	NewGRPCHandler(grpcServer, svc)

	log.Println("GRPC Server Started at ", grpcAddr)

	if err := grpcServer.Serve(l); err != nil {
		log.Fatal(err.Error())
	}
}
