package main

import (
	"context"
	"log"
	"net"
	"os"
	"os/signal"
	"ride-sharing/shared/env"
	"syscall"

	grpcserver "google.golang.org/grpc"
)

var (
	GrpcAddr = env.GetString("GRPC_ADDR", ":9092")
)

func main() {
	// inmemRepo := repository.NewInmemRepository()
	// svc := service.NewService(inmemRepo)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		signCh := make(chan os.Signal, 1)
		signal.Notify(signCh, os.Interrupt, syscall.SIGTERM)
		<-signCh
		cancel()
	}()

	lis, err := net.Listen("tcp", GrpcAddr)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	// Starting the gRPC server
	grpcServer := grpcserver.NewServer()
	// grpc.NewGRPCHandler(grpcServer, svc)

	log.Printf("starting GRPC driver service on port %s", lis.Addr().String())

	go func() {
		if err := grpcServer.Serve(lis); err != nil {
			log.Printf("failed to serve: %v", err)
			cancel()
		}
	}()

	// wait for the shutdown signal
	<-ctx.Done()
	log.Println("shutting down the server...")
	grpcServer.GracefulStop()
}
