# Task Module

The Task module owns task contracts, task management, queue execution, and transactional outbox dispatch. It exposes one `task.Queue` entry point for handler registration, publishing, and consumption. Business code can still depend on the narrower `task.Producer` or `task.Consumer` interfaces when needed.

Before starting the queue, configure PostgreSQL and ensure the task queue database schema has been migrated.

## 1. Define the Business Payload and Handler

Task types, payloads, and handlers belong to the business module that owns them.

```go
const exampleTaskType = "example.v1"

type examplePayload struct {
	Message string `json:"message"`
}

func registerExampleHandler(queue task.Queue) {
	queue.Register(exampleTaskType, task.HandlerFunc(
		func(_ context.Context, message *task.Task) (any, error) {
			var payload examplePayload
			if err := json.Unmarshal(message.Payload, &payload); err != nil {
				return fmt.Errorf("decode example payload: %w", err)
			}

			fmt.Printf("handled task %d: %s\n", message.ID, payload.Message)
			return payload, nil
		},
	))
}
```

The business handler does not read from the queue directly and does not perform task-type routing. `task.TaskQueue` owns handler lookup and invocation.

## 2. Create and Start the Task Queue

Application code can depend on the generic `task.Consumer` interface:

```go
func startConsumer(ctx context.Context, consumer task.Consumer) error {
	return consumer.Start(ctx)
}
```

Create the queue through the task module. No queue vendor configuration or type is exposed to callers:

```go
queue, err := task.NewQueue(ctx, config.QueueConfig{
	DatabaseURL: databaseURL,
	MaxWorkers:  4,
	JobTimeout:  5 * time.Minute,
}, taskRepo)
if err != nil {
	return err
}

registerExampleHandler(queue)

defer queue.Stop()
if err := startConsumer(ctx, queue); err != nil {
	return err
}
```

`Start` launches the queue's fetching and worker loops in the background. `MaxWorkers` controls how many task handlers can execute concurrently in one application instance.

## 3. Publish Through the Producer Interface

Producers should also depend on the generic interface:

```go
func publishExample(ctx context.Context, producer task.Producer) error {
	payload, err := json.Marshal(examplePayload{Message: "hello from the task queue"})
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

Pass the same task queue as the implementation:

```go
if err := publishExample(ctx, queue); err != nil {
	return err
}
```

`Publish` waits for the task to be inserted into the queue, but handler execution is asynchronous. Production workflows that must atomically persist business state and enqueue a task should publish through the transactional Outbox flow instead of calling the queue directly.

## Responsibility Boundaries

- Business modules define task type strings, payloads, and handlers.
- `task.Queue` combines handler registration, publishing, and consumption.
- `task.TaskQueue` is the queue implementation and owns the handler registry.
- `task.Consumer` controls receiving messages and handler execution.
- `task.Producer` provides the queue-neutral publishing contract.
- `task.TaskManager` owns task lifecycle operations backed by `task.TaskStore`.
- `task.Dispatcher` publishes pending records from `task.OutboxStore`.
- The composition root creates the queue from `config.QueueConfig` and injects it into business modules, which register their handlers during construction.

## Task Management and Outbox

Task lifecycle operations and outbox dispatch belong to this module as well:

```go
manager := task.NewTaskManager(taskStore)
pending, err := manager.ListByStatus(ctx, task.StatusPending)
if err != nil {
	return err
}
_ = pending

queue, err := task.NewQueue(ctx, cfg.Queue, taskResultStore)
if err != nil {
	return err
}
dispatcher := task.NewDispatcher(taskStore, queue)
```

The repository package implements `task.TaskStore`, `task.TaskResultStore`, and `task.OutboxStore`; the queue only depends on the narrow `task.TaskResultStore` port needed for automatic completion updates.
