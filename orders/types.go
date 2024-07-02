package main

import (
	"context"

	pb "github.com/rikughi/commons/api"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type OrdersService interface {
	CreateOrder(context.Context, *pb.CreateOrderRequest, []*pb.Item) (*pb.Order, error)
	ValidateOrder(context.Context, *pb.CreateOrderRequest) ([]*pb.Item, error)
}

type OrdersStore interface {
	Create(context.Context, Order) (primitive.ObjectID, error)
}

type Order struct {
	ID          primitive.ObjectID `bson:"_id,omitempty"`
	CustomerID  string             `bson:"customerID,omitempty"`
	Status      string             `bson:"status,omitempty"`
	PaymentLink string             `bson:"paymentLink,omitempty"`
	Items       []*pb.Item         `bson:"items,omitempty"`
}
