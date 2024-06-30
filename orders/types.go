package main

import (
	"context"

	pb "github.com/rikughi/commons/api"
)

type OrdersService interface {
	CreateOrder(context.Context) error
	ValidateOrder(context.Context, *pb.CreaetOrderRequest) error
}

type OrdersStore interface {
	Create(context.Context) error
}