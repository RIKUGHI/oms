package gateway

import (
	"context"

	pb "github.com/rikughi/commons/api"
)

type KitchenGateway interface {
	UpdateOrder(context.Context, *pb.Order) error
}
