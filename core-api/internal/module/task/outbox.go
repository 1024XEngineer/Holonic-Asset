package task

import (
	"context"
	"encoding/json"
	"log"
)

type queuePublisher interface {
	publish(ctx context.Context, task *Task) error
}

type dispatcher struct {
	store    OutboxStore
	producer queuePublisher
}

func newDispatcher(store OutboxStore, producer queuePublisher) *dispatcher {
	return &dispatcher{store: store, producer: producer}
}

func (d *dispatcher) run(ctx context.Context, batchSize int) (int, error) {
	records, err := d.store.FetchPendingOutbox(ctx, batchSize)
	if err != nil {
		return 0, err
	}

	published := 0
	for _, record := range records {
		var message Task
		if err := json.Unmarshal(record.Payload, &message); err != nil {
			log.Printf("task dispatcher: decode outbox %d: %v", record.ID, err)
			continue
		}

		if err := d.producer.publish(ctx, &message); err != nil {
			log.Printf("task dispatcher: publish outbox %d (%s): %v", record.ID, message.Type, err)
			continue
		}

		if err := d.store.MarkOutboxPublished(ctx, record.ID, 0); err != nil {
			log.Printf("task dispatcher: mark published outbox %d: %v", record.ID, err)
			continue
		}
		published++
	}

	return published, nil
}
