# Task Module

The Task module exposes one application-facing concept: `task.Manager`. It owns task creation, handler registration, queue consumption, transactional outbox dispatch, and task queries. Queue and outbox implementations remain internal details.

Before starting the manager, configure PostgreSQL and ensure the task and outbox tables have been migrated.

## Define a Business Handler

Task types, payloads, and handlers belong to the business module that owns them.

```go
const exampleTaskType = "example.v1"

type examplePayload struct {
	Message string `json:"message"`
}

func registerExampleHandler(manager task.Manager) {
	manager.Register(exampleTaskType, task.HandlerFunc(
		func(_ context.Context, message *task.Task) (any, error) {
			var payload examplePayload
			if err := json.Unmarshal(message.Payload, &payload); err != nil {
				return nil, fmt.Errorf("decode example payload: %w", err)
			}

			fmt.Printf("handled task %d: %s\n", message.ID, payload.Message)
			return payload, nil
		},
	))
}
```

## Create and Start the Manager

```go
manager, err := task.NewManager(ctx, config.QueueConfig{
	DatabaseURL: databaseURL,
	MaxWorkers:  4,
	JobTimeout:  5 * time.Minute,
}, taskStore)
if err != nil {
	return err
}

registerExampleHandler(manager)

if err := manager.Start(ctx); err != nil {
	return err
}
defer manager.Stop()
```

`Start` starts both the queue consumer and the transactional outbox poller. The outbox poller continuously forwards persisted task records to the queue, so callers do not need to construct or run a dispatcher separately.

## Publish and Query Tasks

`Publish` creates the task and its transactional outbox record, returning the task ID. Handler execution remains asynchronous.

```go
payload, err := json.Marshal(examplePayload{Message: "hello from the task manager"})
if err != nil {
	return err
}

taskID, err := manager.Publish(ctx, &task.Task{
	Type:    exampleTaskType,
	Status:  task.StatusPending,
	Payload: payload,
})
if err != nil {
	return err
}

detail, err := manager.GetDetail(ctx, taskID)
if err != nil {
	return err
}

pending, err := manager.ListByStatus(ctx, task.StatusPending)
if err != nil {
	return err
}

_ = detail
_ = pending
```

Production workflows that must atomically persist business state and enqueue a task should use the repository's `TaskStore` transaction boundary. The manager's `Publish` method already uses `CreateWithOutbox`, so callers never need to handle outbox records directly.

## Responsibility Boundaries

- Business modules define task type strings, payloads, and handlers.
- `task.Manager` is the only task lifecycle and execution entry point exposed to application code.
- `task.Task` and `task.Handler` define the queue-neutral task contract.
- `task.TaskStore` is the persistence port implemented by the repository adapter.
