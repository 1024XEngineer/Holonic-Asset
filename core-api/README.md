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

Generator is a self-contained business module under `internal/module/generator`; it owns generation requests, run projections, task types, payloads, and task-handler skeletons. HTTP request and response contracts live in the independent `internal/dto` package. External image-provider capabilities remain under `internal/module/generator/imageclient`. Shared helpers such as logging and Viper configuration loading live under `internal/module`. `internal/module/task` exposes one `task.Manager` entry point for task contracts, execution, queries, and transactional outbox dispatch.

Assets are aggregate documents. Asset metadata lives in the asset row, while nested content is stored in `asset_contents` and referenced by the asset's current `content_id`. Asset records map a version number to an immutable content snapshot; content edits use copy-on-write, records create a new snapshot, and rollback switches the current content pointer while discarding records newer than the target. Asset resources are not modeled as a separate table.

The Task module treats `Type` as an opaque string and `Payload` as opaque JSON. Business modules receive the task manager from the composition root and register their own type strings and handlers; Task never defines or switches on business task types. See `internal/module/task/README.md` for the task module usage guide.

## OpenAPI Contract

The Project, Generation, Upload, and Asset routes are OpenAPI-backed. Their Go
DTOs are the source of truth for runtime request binding, the OpenAPI 3.1
document, and generated frontend types. Successful responses consistently use
the `{code,message,data}` envelope.

With the API running, the contract and interactive documentation are available
at:

- `/api/v1/openapi.json`
- `/api/v1/openapi.yaml`
- `/api/v1/docs`

After changing an OpenAPI-backed DTO or route, regenerate the checked-in
contract and frontend types from `frontend/`:

```shell
pnpm api:generate
```

Run `pnpm api:check` to regenerate and type-check the frontend API surface.
Files under `frontend/src/model/generated/` must not be edited by hand.

## Authentication

Configure `auth.jwtSecret` and `auth.tokenExpiry` before starting the API. The
JWT secret must be replaced with a deployment-specific value of at least 32
bytes. `HOLONIC_AUTH_JWT_SECRET` overrides `auth.jwtSecret` from the YAML config
and is the preferred way to supply the secret in deployed environments. The
application only provides login and token verification; it does not create
initial users during startup.

To add the development users explicitly, generate a deployment-specific bcrypt
hash, export it as `BOOTSTRAP_PASSWORD_HASH`, and apply the SQL seed from the
`core-api` directory:

```shell
psql "$DATABASE_URL" -v ON_ERROR_STOP=1 \
  -v password_hash="$BOOTSTRAP_PASSWORD_HASH" -f scripts/seed.sql
```

The seed is idempotent by username and does not replace existing passwords.

## Qiniu Uploads

Configure `qiniu.accessKey`, `qiniu.secretKey`, `qiniu.bucket`, and
`qiniu.domain` in the selected YAML config. `bucket` is the Kodo bucket/S3
bucket name, not a URL. `domain` may be a Kodo download domain or a virtual-host
S3 endpoint such as `https://bucket.s3.cn-east-1.qiniucs.com`.
`qiniu.uploadURL` defaults to `https://upload.qiniup.com`, the upload token
defaults to one hour, and private download URLs default to 30 minutes.

`POST /api/v1/uploads` accepts `contentType` and `contentLength`. It returns a
server-generated object key, temporary private object URL, Qiniu upload
endpoint, and a short-lived upload token. Upload the file to `uploadURL` as
multipart form data using `uploadToken` as `token`, `objectKey` as `key`, and
the file as `file`. The token only permits that object key, MIME type, and exact
size. Project and asset data persist object keys; API responses resolve those
keys to temporary URLs without changing their JSON fields.
