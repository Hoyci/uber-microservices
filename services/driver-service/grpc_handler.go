package main

import (
	"context"
	pb "ride-sharing/shared/proto/driver"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type gRPCHandler struct {
	pb.UnimplementedDriverServiceServer
	Service *Service
}

func NewGRPCHandler(server *grpc.Server, service *Service) {
	handler := &gRPCHandler{
		Service: service,
	}

	pb.RegisterDriverServiceServer(server, handler)
}

func (h *gRPCHandler) RegisterDriver(ctx context.Context, req *pb.RegisterDriverRequest) (*pb.RegisterDriverResponse, error) {
	driverID := req.GetDriverID()
	packageSlug := req.GetPackageSlug()

	driver, err := h.Service.RegisterDriver(driverID, packageSlug)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to register driver")
	}

	return &pb.RegisterDriverResponse{
		Driver: driver,
	}, nil
}
func (h *gRPCHandler) UnregisterDriver(ctx context.Context, req *pb.RegisterDriverRequest) (*pb.RegisterDriverResponse, error) {
	driverID := req.GetDriverID()

	h.Service.UnregisterDriver(driverID)

	return &pb.RegisterDriverResponse{
		Driver: &pb.Driver{
			Id: driverID,
		},
	}, nil
}
