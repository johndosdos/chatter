package broker

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/johndosdos/chatter/internal/model"
	"github.com/nats-io/nats.go/jetstream"
)

func Publisher(ctx context.Context, js jetstream.JetStream, payload model.Message) error {
	if js == nil {
		return fmt.Errorf("internal/broker: jetstream interface is nil")
	}
	if ctx == nil {
		return fmt.Errorf("internal/broker: context is nil")
	}

	p, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("internal/broker: could not encode payload to JSON: %w", err)
	}

	_, err = js.Publish(ctx,
		SubjectGlobalRoom,
		p,
	)
	if err != nil {
		return fmt.Errorf("internal/broker: failed to publish to stream [%s]: %v", SubjectGlobalRoom, err)
	}
	log.Println("internal/broker: publish successful")

	return nil
}

func Subscriber(ctx context.Context, stream jetstream.Stream, handler func(model.Message)) error {
	consumer, err := stream.CreateOrUpdateConsumer(ctx, jetstream.ConsumerConfig{})
	if err != nil {
		return fmt.Errorf("internal/broker: failed to create or update consumer: %w", err)
	}

	consumeHandler := func(msg jetstream.Msg) {
		var payload model.Message

		err := json.Unmarshal(msg.Data(), &payload)
		if err != nil {
			msg.Term()
			log.Printf("internal/broker: could not decode payload: %v", err)
		}

		handler(payload)

		msg.Ack()
	}

	optErrHandler := jetstream.ConsumeErrHandler(func(ctx jetstream.ConsumeContext, err error) {
		log.Printf("consumer error: %v", err)
		ctx.Drain()
	})

	/* 	fetchAmount := 10
	   	messageBatch, err := consumer.Fetch(fetchAmount)
	   	if err != nil {
	   		return fmt.Errorf("internal/broker: failed to fetch [%d] messages", fetchAmount)
	   	}
	*/
	consumeCtx, err := consumer.Consume(consumeHandler, optErrHandler)
	if err != nil {
		return fmt.Errorf("internal/broker: failed to start consuming messages: %w", err)
	}

	go func(ctx context.Context, consumeCtx jetstream.ConsumeContext) {
		<-ctx.Done()
		consumeCtx.Drain()
	}(ctx, consumeCtx)

	return nil
}
