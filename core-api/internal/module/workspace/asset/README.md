# Asset Module and API Contract

This document defines the data structures returned by the Asset detail endpoint to the frontend.

The recommended frontend endpoint:

```http
GET /api/v1/asset/:asset_id
```

Asset list supports fuzzy search by name/description, tag and asset type filtering:

```http
GET /api/v1/projects/:project_id/assets?query=hero&tags=hero&tags=player&types=character
```

`query` performs case-sensitive matching against the asset's `name` or `description`. Multiple `tags` mean the asset must contain all of them (AND); multiple `types` mean any match is accepted (OR). When `query`, `tags`, and `types` are all absent, all assets in the project are returned.

`POST /api/v1/asset/save` copies the current asset's content into a new snapshot version and points the asset's content pointer to the new snapshot. `POST /api/v1/asset/rollback` switches the current pointer back to the specified version's snapshot and deletes all records and their exclusive content after the target version.

```http
GET /api/v1/asset/:asset_id/records
```

This endpoint returns all currently retained records for the asset in ascending version order. Each record includes `recordId`, `assetId`, `version`, `contentId`, `createdAt`, and the snapshot `content`. Versions deleted after a rollback no longer appear in the history list.

At the database level, an asset's basic information is stored in `assets`, current content in `asset_contents` (with `assets.content_id` pointing to the current content), and version records in `asset_records` (with `version`, `content_id`, and `createdAt`). Normal content modifications use copy-on-write and never overwrite content referenced by existing history records.

Updating basic asset parameters still uses the existing route, but the request body is no longer limited to tags:

```http
POST /api/v1/asset/update
Content-Type: application/json
```

```json
{
  "assetId": 1001,
  "name": "Hero",
  "projectId": 101,
  "type": "character",
  "description": "Main character",
  "tags": ["player", "hero"],
  "attributes": {
    "rarity": "legendary"
  },
  "version": 2
}
```

The update endpoint supports partial updates; omitted fields retain their original values. `content` is not accepted for updates, to avoid overwriting prototype, animation, or tileset content when only modifying basic asset metadata. The response includes all updated basic parameters but no `content`.

The asset's full business content is in `data.content`. `data.type` determines the concrete structure of `content`.

## Common Response

```json
{
  "code": 200,
  "message": "success",
  "data": {
    "assetId": 1001,
    "name": "Hero",
    "projectId": 101,
    "type": "character",
    "description": "Main character",
    "tags": ["player", "hero"],
    "attributes": {},
    "content": {},
    "version": 1
  }
}
```

| Field | Type | Description |
| --- | --- | --- |
| `assetId` | `number` | Asset ID |
| `name` | `string` | Asset name |
| `projectId` | `number` | Owning project ID |
| `type` | `string` | Asset type |
| `attributes` | `object` | Form data submitted at asset creation |
| `content` | `object` | Full business content of the asset |
| `version` | `number` | Asset business version |

Current asset types: `character`, `object`, `tileSet`, `audio`, `ui`, `scenery`.

The asset's own metadata goes in `data.attributes`. Extended metadata within the business content goes in `data.content.metadata`. Task IDs, execution status, and error information belong to Tasks and are not nested inside Asset Content. The frontend should query the corresponding Task to determine whether the generation task is complete, then read the written resources from the Asset.

## Character

Character contains `prototype` and `animations`. Direction resources are represented as simple arrays for now; direction names are not modeled in Asset content.

```json
{
  "perspective": "Top-Down",
  "directionCount": 4,
  "prototype": [
    {
      "id": 2101,
      "url": "https://cdn.example.com/hero/prototype-01.png"
    },
    {
      "id": 2102,
      "url": "https://cdn.example.com/hero/prototype-02.png"
    },
    {
      "id": 2103,
      "url": "https://cdn.example.com/hero/prototype-03.png"
    },
    {
      "id": 2104,
      "url": "https://cdn.example.com/hero/prototype-04.png"
    }
  ],
  "animations": [
    {
      "id": 3001,
      "name": "walk",
      "frames": [
        {
          "id": 2201,
          "url": "https://cdn.example.com/hero/walk/001.png",
          "duration": 100
        },
        {
          "id": 2202,
          "url": "https://cdn.example.com/hero/walk/002.png",
          "duration": 100
        }
      ]
    }
  ]
}
```

`directionCount` retains the expected number of directional resources. The arrays do not currently expose names such as `up`, `down`, `left`, or `right`.

For example, a side-on character uses two side-facing prototypes:

```json
{
  "perspective": "Side-On",
  "directionCount": 2,
  "prototype": [
    {
      "id": 2101,
      "url": "https://cdn.example.com/hero/prototype-01.png"
    },
    {
      "id": 2102,
      "url": "https://cdn.example.com/hero/prototype-02.png"
    }
  ]
}
```

Character conventions:

- `perspective` is exactly one of `Top-Down`, `Side-On`, or `Isometric`.
- `directionCount` is derived from character perspective: `Side-On` uses `2`, `Top-Down` uses `4`, and `Isometric` uses `8`.
- `prototype` is a plain array of image resources.
- `animations[].frames` is a plain array of animation frames.
- Direction names and direction-to-array-index mappings are intentionally not defined at this stage.
- Array elements may be absent while generation is incomplete.
- Whether a resource has been generated is determined by the corresponding Task status; Asset content itself does not carry generation state.

## Object

Object uses the same content structure as Character. It may have only a prototype, or it may also include animations.

```json
{
  "perspective": "Side-On",
  "prototype": [],
  "animations": [
    {
      "id": 3101,
      "name": "open",
      "frames": []
    }
  ]
}
```

Object animation names may be business actions such as `open`, `close`, `destroyed`, `activated`, etc.

## TileSet

TileSet does not contain prototype or animations. Instead, it consists of an item list, with each item containing a tile list.

```json
{
  "tileSize": {
    "width": 32,
    "height": 32
  },
  "items": [
    {
      "name": "grass",
      "tiles": [
        {
          "url": "https://cdn.example.com/tileset/grass/center.png",
          "position": {
            "x": 0,
            "y": 1
          }
        },
        {
          "url": "https://cdn.example.com/tileset/grass/top-left.png",
          "position": {
            "x": 1,
            "y": 1
          }
        }
      ]
    },
    { "name": "water", "tiles": [] }
  ]
}
```

TileSet conventions:

- `tileSize` defines the fixed tile dimensions for the entire TileSet. All tiles share the same width and height.
- `items` is the top-level business content of the TileSet.
- `items[].tiles` is the list of tiles generated for that item.
- `items[].name` identifies the tile group; individual tiles do not repeat a name.
- `tiles[].position` is the tile's position on the grid. `x` is the column, `y` is the row, both zero-indexed. For example, `{ "x": 0, "y": 1 }` means column 0, row 1.
- Whether an item has been generated is determined by the corresponding Task status; Asset content itself does not carry state.

## Task Polling

Asset content does not provide a `status` field. The frontend should save the Task ID returned by create or generate operations and query the Task endpoint directly:

1. While the Task is incomplete, keep polling the Task.
2. After the Task completes, re-fetch the Asset detail.
3. For Character/Object, read `prototype[*]` or `animations[*].frames`.
4. For TileSet, read `items[*].tiles`.
5. If the Task fails or is cancelled, stop polling and display the failure reason based on the error information returned by the Task.
