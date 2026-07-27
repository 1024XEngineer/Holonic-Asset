# Task Module

The Task module defines queue-neutral task contracts. Business code depends on `task.Consumer` and `task.Producer`; the composition root supplies `taskriver.Queue` as the concrete River implementation.

Before starting the queue, configure PostgreSQL and ensure the River database schema has been migrated.

## 1. Define the Business Payload and Handler

Task types, payloads, and handlers belong to the business module that owns them.

```go
const exampleTaskType = "example.v1"

type examplePayload struct {
	Message string `json:"message"`
}

func registerExampleHandler(registry *task.Registry) {
	registry.Register(exampleTaskType, task.HandlerFunc(
		func(_ context.Context, message *task.Task) error {
			var payload examplePayload
			if err := json.Unmarshal(message.Payload, &payload); err != nil {
				return fmt.Errorf("decode example payload: %w", err)
			}

			fmt.Printf("handled task %d: %s\n", message.ID, payload.Message)
			return nil
		},
	))
}
```

The business handler does not read from River directly and does not perform task-type routing. `Registry.Dispatch` owns the lookup and invocation logic.

## 2. Start a Consumer Through the Task Interface

Application code accepts the generic `task.Consumer` interface:

```go
func startConsumer(
	ctx context.Context,
	consumer task.Consumer,
	registry *task.Registry,
) error {
	return consumer.Start(ctx, registry.Dispatch)
}
```

Create the River implementation only in the composition root:

```go
dbPool, err := pgxpool.New(ctx, databaseURL)
if err != nil {
	return err
}
defer dbPool.Close()

riverQueue, err := taskriver.New(dbPool, &river.Config{
	Queues: map[string]river.QueueConfig{
		river.QueueDefault: {
			MaxWorkers: 4,
		},
	},
})
if err != nil {
	return err
}
defer riverQueue.Stop()

registry := task.NewRegistry()
registerExampleHandler(registry)

if err := startConsumer(ctx, riverQueue, registry); err != nil {
	return err
}
```

`Start` launches River's fetching and worker loops in the background. `MaxWorkers` controls how many task handlers the default queue can execute concurrently in one application instance.

## 3. Publish Through the Producer Interface

Producers should also depend on the generic interface:

```go
func publishExample(ctx context.Context, producer task.Producer) error {
	payload, err := json.Marshal(examplePayload{Message: "hello from River"})
	if err != nil {
		return fmt.Errorf("encode example payload: %w", err)
	}

	return producer.Publish(ctx, &task.Task{
		Type:    exampleTaskType,
		Status:  task.StatusPending,
		Payload: payload,
	})
}
```

Pass the same River queue as the implementation:

```go
if err := publishExample(ctx, riverQueue); err != nil {
	return err
}
```

`Publish` waits for the task to be inserted into River, but handler execution is asynchronous. Production workflows that must atomically persist business state and enqueue a task should publish through the transactional Outbox flow instead of calling the queue directly.

## Responsibility Boundaries

- Business modules define task type strings, payloads, and handlers.
- `task.Registry` registers handlers and dispatches tasks by their opaque type.
- `task.Consumer` controls receiving messages and handler execution.
- `task.Producer` provides the queue-neutral publishing contract.
- `taskriver.Queue` implements both interfaces with River.
- The composition root selects River and configures worker concurrency.
