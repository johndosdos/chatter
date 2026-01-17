package broker

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
	"github.com/johndosdos/chatter/internal/model"
	"github.com/nats-io/nats.go/jetstream"
)

func Publisher(ctx context.Context, js jetstream.JetStream, payload model.Message) error {
	if js == nil {
		return fmt.Errorf("jetstream interface is nil")
	}
	if ctx == nil {
		return fmt.Errorf("context is nil")
	}

	p, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("could not encode payload to JSON: %w", err)
	}

	_, err = js.Publish(ctx,
		SubjectGlobalRoom,
		p,
		jetstream.WithMsgID(uuid.NewString()), // Deduplication
	)
	if err != nil {
		return fmt.Errorf("failed to publish to stream [%s]: %v", SubjectGlobalRoom, err)
	}
	log.Printf("publish successful from [%s]\n", payload.Username)

	return nil
}

func Subscriber(ctx context.Context, stream jetstream.Stream, handler func(model.Message)) error {
	consumer, err := stream.CreateOrUpdateConsumer(ctx, jetstream.ConsumerConfig{
		Durable:    "chat_consumer", // Persistent consumer
		AckPolicy:  jetstream.AckExplicitPolicy,
		AckWait:    30 * time.Second, // Time to wait before redelivery
		MaxDeliver: 3,                // Max retry attempts
	})
	if err != nil {
		return fmt.Errorf("failed to create or update consumer: %w", err)
	}

	consumeHandler := func(msg jetstream.Msg) {
		var payload model.Message

		err := json.Unmarshal(msg.Data(), &payload)
		if err != nil {
			log.Printf("could not decode payload: %v", err)
			if err := msg.Term(); err != nil {
				log.Printf("could not terminate message: %v", err)
				return
			}
			return
		}

		handler(payload)

		if err := msg.Ack(); err != nil {
			log.Printf("failed to acknowledge message: %+v", err)
		}
	}

	optErrHandler := jetstream.ConsumeErrHandler(func(ctx jetstream.ConsumeContext, err error) {
		log.Printf("consumer error: %v", err)
		ctx.Drain()
	})

	consumeCtx, err := consumer.Consume(consumeHandler, optErrHandler)
	if err != nil {
		return fmt.Errorf("failed to start consuming messages: %w", err)
	}

	go func(ctx context.Context, consumeCtx jetstream.ConsumeContext) {
		<-ctx.Done()
		consumeCtx.Drain()
	}(ctx, consumeCtx)

	return nil
}
