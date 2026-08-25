package task

import (
	"context"
	"encoding/json"
	"errors"
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

func (p *producerStub) publish(_ context.Context, message *Task) error {
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
	dispatcher := newDispatcher(store, producer)

	published, err := dispatcher.run(context.Background(), 10)
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

type outboxDispatchStore struct {
	records     []OutboxRecord
	fetchErr    error
	markErr     error
	fetchLimit  int
	marked      []uint
	fetchSignal chan struct{}
}

func (s *outboxDispatchStore) FetchPendingOutbox(_ context.Context, limit int) ([]OutboxRecord, error) {
	s.fetchLimit = limit
	if s.fetchSignal != nil {
		select {
		case s.fetchSignal <- struct{}{}:
		default:
		}
	}
	return s.records, s.fetchErr
}

func (s *outboxDispatchStore) MarkOutboxPublished(_ context.Context, id uint, _ int64) error {
	if s.markErr != nil {
		return s.markErr
	}
	s.marked = append(s.marked, id)
	return nil
}

type queuePublisherStub struct {
	err      error
	messages []*Task
}

func (p *queuePublisherStub) publish(_ context.Context, task *Task) error {
	if p.err != nil {
		return p.err
	}
	p.messages = append(p.messages, task)
	return nil
}

func TestDispatcherHandlesFetchDecodePublishAndMarkFailures(t *testing.T) {
	t.Run("fetch failure", func(t *testing.T) {
		wantErr := errors.New("outbox unavailable")
		d := newDispatcher(&outboxDispatchStore{fetchErr: wantErr}, &queuePublisherStub{})
		if count, err := d.run(context.Background(), 5); !errors.Is(err, wantErr) || count != 0 {
			t.Fatalf("fetch failure result: count=%d err=%v", count, err)
		}
	})

	t.Run("decode and publish failures", func(t *testing.T) {
		payload, err := json.Marshal(Task{ID: 31, Type: "publish-error"})
		if err != nil {
			t.Fatalf("marshal task: %v", err)
		}
		store := &outboxDispatchStore{records: []OutboxRecord{
			{ID: 1, Payload: []byte("invalid")},
			{ID: 2, Payload: payload},
		}}
		producer := &queuePublisherStub{err: errors.New("queue unavailable")}
		if count, err := newDispatcher(store, producer).run(context.Background(), 5); err != nil || count != 0 || len(store.marked) != 0 {
			t.Fatalf("publish failure result: count=%d err=%v marked=%v", count, err, store.marked)
		}
	})

	t.Run("mark failure", func(t *testing.T) {
		payload, err := json.Marshal(Task{ID: 32, Type: "mark-error"})
		if err != nil {
			t.Fatalf("marshal task: %v", err)
		}
		store := &outboxDispatchStore{records: []OutboxRecord{{ID: 3, Payload: payload}}, markErr: errors.New("mark unavailable")}
		producer := &queuePublisherStub{}
		count, err := newDispatcher(store, producer).run(context.Background(), 5)
		if err != nil || count != 0 || len(producer.messages) != 1 {
			t.Fatalf("mark failure result: count=%d err=%v messages=%d", count, err, len(producer.messages))
		}
	})
}
