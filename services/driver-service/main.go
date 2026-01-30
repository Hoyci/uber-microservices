package main

import (
	"context"
	"log"
	"net"
	"os"
	"os/signal"
	"ride-sharing/shared/env"
	"ride-sharing/shared/messaging"
	"syscall"

	grpcserver "google.golang.org/grpc"
)

var (
	GrpcAddr = env.GetString("GRPC_ADDR", ":9092")
	AMQPAddr = env.GetString("RABBITMQ_URI", "amqp://guest:guest@rabbitmq:5672")
)

func main() {
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

	// Starting the rabbitMQ connection
	rabbitmq, err := messaging.NewRabbitMQ(AMQPAddr)
	if err != nil {
		log.Fatal(err)
	}
	defer rabbitmq.Close()

	log.Println("starting rabbitmq connection")

	// Starting the gRPC server
	service := NewService()
	grpcServer := grpcserver.NewServer()
	NewGRPCHandler(grpcServer, service)

	// Starting the consumer messages
	consumer := NewTripConsumer(rabbitmq, service)
	go func() {
		if err := consumer.Listen(); err != nil {
			log.Fatalf("failed to listen to the messages: %v", err)
		}
	}()

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
