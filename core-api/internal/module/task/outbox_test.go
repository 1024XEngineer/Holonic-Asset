package task

import (
	"context"
	"encoding/json"
	"testing"
)

type outboxStoreStub struct {
	records   []OutboxRecord
	published []uint
}

func (s *outboxStoreStub) FetchPendingOutbox(context.Context, int) ([]OutboxRecord, error) {
	return s.records, nil
}

func (s *outboxStoreStub) MarkOutboxPublished(_ context.Context, outboxID uint, _ int64) error {
	s.published = append(s.published, outboxID)
	return nil
}

type producerStub struct {
	messages []*Task
}

func (p *producerStub) Publish(_ context.Context, message *Task) error {
	p.messages = append(p.messages, message)
	return nil
}

func TestDispatcherPublishesAndMarksOutboxRecords(t *testing.T) {
	payload, err := json.Marshal(Task{ID: 7, Type: "example.v1"})
	if err != nil {
		t.Fatalf("marshal task: %v", err)
	}

	store := &outboxStoreStub{records: []OutboxRecord{{ID: 11, Payload: payload}}}
	producer := &producerStub{}
	dispatcher := NewDispatcher(store, producer)

	published, err := dispatcher.Run(context.Background(), 10)
	if err != nil {
		t.Fatalf("run dispatcher: %v", err)
	}
	if published != 1 || len(producer.messages) != 1 {
		t.Fatalf("unexpected dispatch result: published=%d messages=%d", published, len(producer.messages))
	}
	if producer.messages[0].ID != 7 || len(store.published) != 1 || store.published[0] != 11 {
		t.Fatalf("unexpected dispatch state: messages=%+v published=%v", producer.messages, store.published)
	}
}
