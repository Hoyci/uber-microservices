package events

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"ride-sharing/services/trip-service/internal/domain"
	"ride-sharing/shared/contracts"
	"ride-sharing/shared/messaging"
	pbd "ride-sharing/shared/proto/driver"

	amqp "github.com/rabbitmq/amqp091-go"
)

type driverConsumer struct {
	rabbitmq *messaging.RabbitMQ
	service  domain.TripService
}

func NewDriverConsumer(rabbitmq *messaging.RabbitMQ, service domain.TripService) *driverConsumer {
	return &driverConsumer{
		rabbitmq: rabbitmq,
		service:  service,
	}
}

func (c *driverConsumer) Listen() error {
	c.rabbitmq.ConsumeMessages(messaging.DriverCmdTripResponseQueue, func(ctx context.Context, msg amqp.Delivery) error {
		var message contracts.AmqpMessage
		if err := json.Unmarshal(msg.Body, &message); err != nil {
			log.Printf("failed to unmarshal message event: %v", err)
			return err
		}

		var payload messaging.DriverTripResponseData
		if err := json.Unmarshal(message.Data, &payload); err != nil {
			log.Printf("failed to unmarshal message event: %v", err)
			return err
		}

		log.Printf("driver response received message: %+v", payload)

		switch msg.RoutingKey {
		case contracts.DriverCmdTripAccept:
			if err := c.handleTripAccept(ctx, payload.TripID, payload.Driver); err != nil {
				log.Printf("failed to handle the trip accept: %v", err)
				return err
			}
		case contracts.DriverCmdTripDecline:
			if err := c.handleTripDecline(ctx, payload.TripID, payload.RiderID); err != nil {
				log.Printf("failed to handle the trip accept: %v", err)
				return err
			}
			return nil
		}

		log.Printf("unknown trip event: %+v", payload)

		return nil
	})

	return nil
}

func (c *driverConsumer) handleTripAccept(ctx context.Context, tripID string, driver *pbd.Driver) error {
	fmt.Println("tripID", tripID)
	trip, err := c.service.GetTripByID(ctx, tripID)
	if err != nil {
		return err
	}

	fmt.Println("tripando", trip)

	if trip == nil {
		return fmt.Errorf("Trip was not found %s", tripID)
	}

	// It should return the updated data, but I will fetch again for simplicity
	if err := c.service.UpdateTrip(ctx, tripID, "accepted", driver); err != nil {
		log.Printf("failed to update the trip: %v", err)
		return err
	}

	trip, err = c.service.GetTripByID(ctx, tripID)
	if err != nil {
		return err
	}

	tripMarshalled, err := json.Marshal(trip)
	if err != nil {
		return err
	}

	// Notify the rider that a driver has been assigned
	if err := c.rabbitmq.PublishMessage(ctx, contracts.TripEventDriverAssigned, contracts.AmqpMessage{
		OwnerID: trip.UserID,
		Data:    tripMarshalled,
	}); err != nil {
		return err
	}

	// TODO: Notify the payment service to start a payment link

	return nil
}

func (c *driverConsumer) handleTripDecline(ctx context.Context, tripID, riderID string) error {
	trip, err := c.service.GetTripByID(ctx, tripID)
	if err != nil {
		return err
	}

	newPayload := messaging.TripEventData{
		Trip: trip.ToProto(),
	}

	marshalledPayloaad, err := json.Marshal(newPayload)
	if err != nil {
		return err
	}

	if err := c.rabbitmq.PublishMessage(ctx, contracts.TripEventDriverNotInterested, contracts.AmqpMessage{
		OwnerID: riderID,
		Data:    marshalledPayloaad,
	}); err != nil {
		return err
	}

	return nil
}
