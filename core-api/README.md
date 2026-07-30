# Holonic-Asset Backend

## Architecture

```text
config/
internal/
  dto/
  handler/
  model/
    ai/
    asset/
    project/
  module/
    generator/
    processor/
    task/
  repository/
    dao/
  router/
  service/
ioc/
middleware/
pkg/
```

Feature business models generally live under `internal/model/<feature>`. Generator is a self-contained business module under `internal/module/generator`; it owns generation requests, run projections, task types, payloads, and task-handler skeletons. HTTP request and response contracts live in the independent `internal/dto` package. Shared use-case and external-provider ports remain under `internal/service`. `internal/module/task` owns task contracts, task management, queue execution, and transactional outbox dispatch, while the currently empty `internal/module/processor` is reserved for future image operations such as crop, resize, and compositing.

Assets are aggregate documents. Asset metadata lives in the asset row, while nested content is stored in `asset_contents` and referenced by the asset's current `content_id`. Asset records map a version number to an immutable content snapshot; content edits use copy-on-write, records create a new snapshot, and rollback switches the current content pointer while discarding records newer than the target. Asset resources are not modeled as a separate table.

The Task module treats `Type` as an opaque string and `Payload` as opaque JSON. Business modules receive the queue from the composition root and register their own type strings and handlers; Task never defines or switches on business task types. See `internal/module/task/README.md` for the task module usage guide.
