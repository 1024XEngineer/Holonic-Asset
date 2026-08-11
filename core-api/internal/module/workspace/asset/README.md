# Asset Module and API Contract

An Asset stores identity, editable information, type-specific dimensions, a
pointer to separately stored content, and a version.

```json
{
  "assetId": 1001,
  "projectId": 101,
  "type": "character",
  "name": "Hero",
  "description": "Main playable character",
  "perspective": "Top-Down",
  "dimensions": {"width": 64, "height": 64},
  "tags": ["player", "hero"],
  "content": {},
  "version": 1
}
```

`name`, `description`, `perspective`, and `dimensions` are direct Asset fields.
There is no nested `attributes` field, and Content must not duplicate these
fields. Task state and errors belong to the Task domain, not Asset Content.

## Dimensions

Asset `type` determines the accepted Dimensions shape. Unknown fields and zero
dimensions are rejected.

| Asset type | Dimensions |
| --- | --- |
| `character` | One frame: `{width,height}` |
| `object` | One frame: `{width,height}` |
| `tileSet` | `{tileSize:{width,height},tileAmount:{columns,rows}}` |
| `uiset` | Complete canvas: `{width,height}` |
| `scenery` | Complete canvas: `{width,height}` |
| `audio` | `null` |

Dimensions are the final game-asset specifications used by the deterministic image
Processor. It is not the image provider's generation size. The provider may
choose its own native output size; generated images are then resized by the
Processor to the Asset Dimensions.

For TileSet, `tileAmount` is grid capacity rather than the number of generated
Items. Complete canvas dimensions are derived and are not stored:

```text
width  = tileSize.width  * tileAmount.columns
height = tileSize.height * tileAmount.rows
```

## Content

Character Content contains `directionCount`, `prototype`, and `animations`.
`directionCount` is derived from the Asset perspective: `Side-On` uses `2`,
`Top-Down` uses `4`, and `Isometric` uses `8`. Callers do not provide it.
Object Content contains `prototype` and `animations`. TileSet Content contains
`items`; tile dimensions belong to Dimensions. UI Set Content contains `components`,
including component size and position. Scenery Content contains `layers`,
including each layer's resource, position, transform, visibility, opacity, and
z-order.

Component size and layer transform scaling never modify the parent Asset Dimensions.

## API

```http
GET /api/v1/asset/:asset_id
GET /api/v1/projects/:project_id/assets
POST /api/v1/asset/update
POST /api/v1/asset/save
GET /api/v1/asset/:asset_id/records
POST /api/v1/asset/rollback
```

Asset Update accepts partial changes to `name`, `description`, `perspective`,
`dimensions`, and `tags`. It does not accept `projectId`, `type`, `attributes`, or
`content`. Updating Dimensions changes Asset information only; it does not run a
Generator or Processor, rewrite Content, create a Record, or increment the
version.

## Records

A Record snapshots `name`, `description`, `perspective`, `dimensions`, `content`,
and `version`. Content remains in `asset_contents`, and the Record references
its immutable content snapshot by `contentId`.

Save creates a new snapshot. Rollback restores the recorded fields, current
Content pointer, and version, then removes records after the selected version.
Tags are editable Asset information but are not part of the version snapshot.

Normal Content edits use copy-on-write. Task callers poll Task status and
re-fetch the Asset after successful generation.
