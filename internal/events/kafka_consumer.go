package events

import (
	"context"

	kafkaLib "github.com/Kilat-Pet-Delivery/lib-common/kafka"
	"github.com/Kilat-Pet-Delivery/lib-proto/events"
	"github.com/Kilat-Pet-Delivery/service-tracking/internal/application"
	kafkaGo "github.com/segmentio/kafka-go"
	"go.uber.org/zap"
)

// BookingEventConsumer consumes booking events and dispatches them to the tracking service.
type BookingEventConsumer struct {
	consumer *kafkaLib.Consumer
	service  *application.TrackingService
	logger   *zap.Logger
}

// NewBookingEventConsumer creates a new consumer for booking events.
func NewBookingEventConsumer(
	brokers []string,
	groupID string,
	service *application.TrackingService,
	logger *zap.Logger,
) *BookingEventConsumer {
	consumer := kafkaLib.NewConsumer(brokers, groupID, events.TopicBookingEvents, logger)
	return &BookingEventConsumer{
		consumer: consumer,
		service:  service,
		logger:   logger,
	}
}

// Start begins consuming booking events. Blocks until the context is cancelled.
func (c *BookingEventConsumer) Start(ctx context.Context) error {
	return c.consumer.Consume(ctx, c.handleMessage)
}

// handleMessage processes a single booking event message.
func (c *BookingEventConsumer) handleMessage(ctx context.Context, msg kafkaGo.Message) error {
	cloudEvent, err := kafkaLib.ParseCloudEvent(msg.Value)
	if err != nil {
		c.logger.Error("failed to parse cloud event from booking topic",
			zap.Error(err),
			zap.Int64("offset", msg.Offset),
		)
		return err
	}

	c.logger.Debug("received booking event",
		zap.String("type", cloudEvent.Type),
		zap.String("id", cloudEvent.ID),
	)

	switch cloudEvent.Type {
	case events.BookingAccepted:
		var evt events.BookingAcceptedEvent
		if err := cloudEvent.ParseData(&evt); err != nil {
			c.logger.Error("failed to parse booking accepted event data", zap.Error(err))
			return err
		}
		return c.service.HandleBookingAccepted(ctx, evt)

	case events.BookingDeliveryConfirmed:
		var evt events.DeliveryConfirmedEvent
		if err := cloudEvent.ParseData(&evt); err != nil {
			c.logger.Error("failed to parse delivery confirmed event data", zap.Error(err))
			return err
		}
		return c.service.HandleDeliveryConfirmed(ctx, evt)

	default:
		c.logger.Debug("ignoring unhandled booking event type",
			zap.String("type", cloudEvent.Type),
		)
		return nil
	}
}

// Close shuts down the booking event consumer.
func (c *BookingEventConsumer) Close() error {
	return c.consumer.Close()
}

// RunnerEventConsumer consumes runner events and dispatches them to the tracking service.
type RunnerEventConsumer struct {
	consumer *kafkaLib.Consumer
	service  *application.TrackingService
	logger   *zap.Logger
}

// NewRunnerEventConsumer creates a new consumer for runner events.
func NewRunnerEventConsumer(
	brokers []string,
	groupID string,
	service *application.TrackingService,
	logger *zap.Logger,
) *RunnerEventConsumer {
	consumer := kafkaLib.NewConsumer(brokers, groupID, events.TopicRunnerEvents, logger)
	return &RunnerEventConsumer{
		consumer: consumer,
		service:  service,
		logger:   logger,
	}
}

// Start begins consuming runner events. Blocks until the context is cancelled.
func (c *RunnerEventConsumer) Start(ctx context.Context) error {
	return c.consumer.Consume(ctx, c.handleMessage)
}

// handleMessage processes a single runner event message.
func (c *RunnerEventConsumer) handleMessage(ctx context.Context, msg kafkaGo.Message) error {
	cloudEvent, err := kafkaLib.ParseCloudEvent(msg.Value)
	if err != nil {
		c.logger.Error("failed to parse cloud event from runner topic",
			zap.Error(err),
			zap.Int64("offset", msg.Offset),
		)
		return err
	}

	c.logger.Debug("received runner event",
		zap.String("type", cloudEvent.Type),
		zap.String("id", cloudEvent.ID),
	)

	switch cloudEvent.Type {
	case events.RunnerLocationUpdate:
		var evt events.RunnerLocationUpdateEvent
		if err := cloudEvent.ParseData(&evt); err != nil {
			c.logger.Error("failed to parse runner location update event data", zap.Error(err))
			return err
		}
		return c.service.HandleRunnerLocationUpdate(ctx, evt)

	default:
		c.logger.Debug("ignoring unhandled runner event type",
			zap.String("type", cloudEvent.Type),
		)
		return nil
	}
}

// Close shuts down the runner event consumer.
func (c *RunnerEventConsumer) Close() error {
	return c.consumer.Close()
}
