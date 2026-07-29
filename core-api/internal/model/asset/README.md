# Asset API Response Contract

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

Character contains `prototype` and `animations`. Directions under an animation use dynamic keys; the number of directions is not fixed.

```json
{
  "viewMode": "top_down",
  "directionCount": 4,
  "prototype": {
    "directions": {
      "up": {
        "image": {
          "url": "https://cdn.example.com/hero/prototype-up.png"
        }
      },
      "down": {
        "image": {
          "url": "https://cdn.example.com/hero/prototype-down.png"
        }
      },
      "left": {
        "image": {
          "url": "https://cdn.example.com/hero/prototype-left.png"
        }
      },
      "right": {
        "image": {
          "url": "https://cdn.example.com/hero/prototype-right.png"
        }
      }
    }
  },
  "animations": [
    {
      "id": 3001,
      "name": "walk",
      "directions": {
        "left": {
          "frames": [
            {
              "url": "https://cdn.example.com/hero/walk/left/001.png",
              "duration": 100
            }
          ]
        }
      }
    }
  ]
}
```

The four directions above are only an example for `top_down`. The direction set is specified by `directionCount`; the actual direction names are determined by the keys of `prototype.directions` and `animations[].directions`.

For example, `side_on` may only contain left and right directions:

```json
{
  "viewMode": "side_on",
  "directionCount": 2,
  "prototype": {
    "directions": {
      "left": {
        "image": {
          "id": 2101,
          "url": "https://cdn.example.com/hero/prototype-left.png"
        }
      },
      "right": {
        "image": {
          "id": 2102,
          "url": "https://cdn.example.com/hero/prototype-right.png"
        }
      }
    }
  }
}
```

Character conventions:

- `viewMode` represents the asset's perspective mode. Currently includes `side_on` and `top_down`.
- `directionCount` is the number of directions to generate for the current asset. Only `1`, `2`, `4`, or `8` are used.
- `directionCount: 1` corresponds to `front`.
- `directionCount: 2` corresponds to `left`, `right`.
- `directionCount: 4` corresponds to `up`, `down`, `left`, `right`.
- `directionCount: 8` corresponds to the four cardinal directions plus `up_left`, `up_right`, `down_left`, `down_right`.
- Keys of `prototype.directions` must use the direction names listed above and match `directionCount`.
- Each prototype direction must contain exactly one `image` when complete.
- `side_on` may have `left` and `right` (two directions), corresponding to two images.
- `top_down` may have `up`, `down`, `left`, `right` (four directions), corresponding to four images.
- The keys in `directions` indicate which directions are actually selected for generation this time.
- `frames` may be absent when the direction is not yet complete.
- Whether a resource has been generated is determined by the corresponding Task status; Asset content itself does not carry generation state.

## Object

Object uses the same content structure as Character. It may have only a prototype, or it may also include animations.

```json
{
  "viewMode": "side_on",
  "directionCount": 2,
  "prototype": {
    "directions": {}
  },
  "animations": [
    {
      "id": 3101,
      "name": "open",
      "directions": {}
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
          "name": "grass-center",
          "url": "https://cdn.example.com/tileset/grass/center.png",
          "position": {
            "x": 0,
            "y": 1
          },
          "metadata": {
            "tileType": "center"
          }
        },
        {
          "name": "grass-top-left",
          "url": "https://cdn.example.com/tileset/grass/top-left.png",
          "position": {
            "x": 1,
            "y": 1
          },
          "metadata": {
            "tileType": "top-left"
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
- `tiles[].position` is the tile's position on the grid. `x` is the column, `y` is the row, both zero-indexed. For example, `{ "x": 0, "y": 1 }` means column 0, row 1.
- Whether an item has been generated is determined by the corresponding Task status; Asset content itself does not carry state.
- Extended tile properties go in `metadata`, e.g. tile type. Fixed dimensions and grid position use `tileSize` and `position` respectively.

## Task Polling

Asset content does not provide a `status` field. The frontend should save the Task ID returned by create or generate operations and query the Task endpoint directly:

1. While the Task is incomplete, keep polling the Task.
2. After the Task completes, re-fetch the Asset detail.
3. For Character/Object, read `prototype.directions[*].image` or `animations[*].directions[*].frames`.
4. For TileSet, read `items[*].tiles`.
5. If the Task fails or is cancelled, stop polling and display the failure reason based on the error information returned by the Task.
