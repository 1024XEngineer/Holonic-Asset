package outbox

import (
	"context"
	"encoding/json"
	"log"

	"github.com/1024XEngineer/Holonic-Asset/internal/module/task"
	"github.com/1024XEngineer/Holonic-Asset/internal/repository"
)

// Dispatcher publishes generic task envelopes stored in the transactional outbox.
type Dispatcher struct {
	repo     repository.TaskRepository
	producer task.Producer
}

func NewDispatcher(repo repository.TaskRepository, producer task.Producer) *Dispatcher {
	return &Dispatcher{repo: repo, producer: producer}
}

func (d *Dispatcher) Run(ctx context.Context, batchSize int) (int, error) {
	records, err := d.repo.FetchPendingOutbox(ctx, batchSize)
	if err != nil {
		return 0, err
	}

	published := 0
	for _, record := range records {
		var message task.Task
		if err := json.Unmarshal(record.Payload, &message); err != nil {
			log.Printf("task dispatcher: decode outbox %d: %v", record.ID, err)
			continue
		}

		if err := d.producer.Publish(ctx, &message); err != nil {
			log.Printf("task dispatcher: publish outbox %d (%s): %v", record.ID, message.Type, err)
			continue
		}

		if err := d.repo.MarkOutboxPublished(ctx, record.ID, 0); err != nil {
			log.Printf("task dispatcher: mark published outbox %d: %v", record.ID, err)
			continue
		}
		published++
	}

	return published, nil
}
