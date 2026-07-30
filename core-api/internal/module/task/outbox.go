package task

import (
	"context"
	"encoding/json"
	"log"
)

// Dispatcher publishes task messages stored in the transactional outbox.
type Dispatcher struct {
	store    OutboxStore
	producer Producer
}

func NewDispatcher(store OutboxStore, producer Producer) *Dispatcher {
	return &Dispatcher{store: store, producer: producer}
}

func (d *Dispatcher) Run(ctx context.Context, batchSize int) (int, error) {
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

		if err := d.producer.Publish(ctx, &message); err != nil {
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
