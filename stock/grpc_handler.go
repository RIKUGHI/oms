package main

import (
	"context"

	pb "github.com/rikughi/commons/api"
	"google.golang.org/grpc"
)

type StockGrpcHandler struct {
	pb.UnimplementedStockServiceServer

	service StockService
}

func NewGRPCHandler(
	server *grpc.Server,
	stockService StockService,
) {
	handler := &StockGrpcHandler{
		service: stockService,
	}

	pb.RegisterStockServiceServer(server, handler)
}

func (s *StockGrpcHandler) CheckIfItemIsInStock(ctx context.Context, p *pb.CheckIfItemIsInStockRequest) (*pb.CheckIfItemIsInStockResponse, error) {
	inStock, items, err := s.service.CheckIfItemAreInStock(ctx, p.Items)
	if err != nil {
		return nil, err
	}

	return &pb.CheckIfItemIsInStockResponse{
		InStock: inStock,
		Items:   items,
	}, nil
}
