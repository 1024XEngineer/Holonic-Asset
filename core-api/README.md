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
    generation/
    project/
    task/
  module/
    processor/
    task/
  outbox/
  repository/
    dao/
  router/
  service/
ioc/
middleware/
pkg/
```

Feature business models live directly under `internal/model/<feature>` without generic `domain` packages. HTTP request and response contracts live in the independent `internal/dto` package; handlers only map DTOs to use-case contracts. Shared use-case and external-provider ports are owned by `internal/service`. `internal/module/task` contains task-queue abstractions, while the currently empty `internal/module/processor` is reserved for future image operations such as crop, resize, and compositing.

The Task module treats `Type` as an opaque string and `Payload` as opaque JSON. Business modules register their own type strings and handlers; Task never defines or switches on business task types.
