# Holonic-Asset Backend

## Architecture

```text
config/
internal/
  dto/
  handler/
  model/
    ai/
  module/
    generator/
    processor/
    task/
    workspace/
      asset/
      project/
  repository/
    dao/
  router/
  service/
ioc/
middleware/
pkg/
```

Workspace is the business module for project and asset lifecycle operations. `internal/module/workspace/project` and `internal/module/workspace/asset` own their domain models, persistence ports, and managers; the root `workspace.Workspace` groups both capabilities. Repository implementations and DAOs remain infrastructure adapters under `internal/repository`.

Generator is a self-contained business module under `internal/module/generator`; it owns generation requests, run projections, task types, payloads, and task-handler skeletons. HTTP request and response contracts live in the independent `internal/dto` package. Shared external-provider capabilities remain under `internal/service`. `internal/module/task` owns task contracts, task management, queue execution, and transactional outbox dispatch, while the currently empty `internal/module/processor` is reserved for future image operations such as crop, resize, and compositing.

Assets are aggregate documents. Asset metadata lives in the asset row, while nested content is stored in `asset_contents` and referenced by the asset's current `content_id`. Asset records map a version number to an immutable content snapshot; content edits use copy-on-write, records create a new snapshot, and rollback switches the current content pointer while discarding records newer than the target. Asset resources are not modeled as a separate table.

The Task module treats `Type` as an opaque string and `Payload` as opaque JSON. Business modules receive the queue from the composition root and register their own type strings and handlers; Task never defines or switches on business task types. See `internal/module/task/README.md` for the task module usage guide.
