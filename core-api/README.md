# Holonic-Asset Backend

## Architecture

```text
database.go
log.go
task.go
internal/
  config/
  middleware/
  module/
    echox/
    generator/
      imageclient/
    logger/
    task/
    upload/
    viperx/
    workspace/
      asset/
      project/
  dto/
  handler/
  repository/
    dao/
  router/
```

Workspace is the business module for project and asset lifecycle operations. `internal/module/workspace/project` and `internal/module/workspace/asset` own their domain models, persistence ports, and managers; the root `workspace.Workspace` groups both capabilities. Repository implementations and DAOs remain infrastructure adapters under `internal/repository`.

Generator is a self-contained business module under `internal/module/generator`; it owns generation requests, run projections, task types, payloads, and task-handler skeletons. HTTP request and response contracts live in the independent `internal/dto` package. External image-provider capabilities remain under `internal/module/generator/imageclient`. Shared helpers such as Echo integration, logging, and Viper configuration loading live under `internal/module`. `internal/module/task` exposes one `task.Manager` entry point for task contracts, execution, queries, and transactional outbox dispatch.

Assets are aggregate documents. Asset metadata lives in the asset row, while nested content is stored in `asset_contents` and referenced by the asset's current `content_id`. Asset records map a version number to an immutable content snapshot; content edits use copy-on-write, records create a new snapshot, and rollback switches the current content pointer while discarding records newer than the target. Asset resources are not modeled as a separate table.

The Task module treats `Type` as an opaque string and `Payload` as opaque JSON. Business modules receive the task manager from the composition root and register their own type strings and handlers; Task never defines or switches on business task types. See `internal/module/task/README.md` for the task module usage guide.
