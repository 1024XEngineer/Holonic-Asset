# Animation Generation Module

This document describes the request fields, direction selection, image references, and processing boundaries for `generate_animation` in `internal/module/generator`.

## 1. Core Conventions

Animation belongs to an existing character or object asset; it does not create a new top-level asset.

- `assetId`: The existing asset ID, used to look up the prototype and write the animation back to that asset.
- `animation_name`: The animation clip name, e.g., `idle`, `walk`, `sword_slash`, or `open`.
- `direction`: English direction name for both character and object assets.
- `prompt`: Natural language description of the action, which becomes `creative_brief` in the task payload.

Direction semantics are guaranteed by the prototype array ordering. The API never exposes a numeric prototype index. It accepts at most eight English direction names and maps them according to the asset's `direction_count`:

| `direction_count` | Accepted direction order |
| --- | --- |
| 2 | `left`, `right` |
| 4 | `front`, `right`, `back`, `left` |
| 8 | `front`, `front_right`, `right`, `back_right`, `back`, `back_left`, `left`, `front_left` |

Character and object animation both require an explicit `direction` and support only assets with `direction_count` equal to `2`, `4`, or `8`. The generator resolves the name to the same prototype-array ordering for both asset types and never sends a complete multi-direction sheet to the video model.

Animation requests do not use `parent_id`, `asset_name`, or frontend-uploaded `reference_image`. The animation name field is `animation_name`, and the target asset field is the outer `assetId`.

## 2. HTTP Request

Endpoint:

```text
POST /api/v1/projects/{project_id}/generation-runs
```

Example:

```json
{
  "assetId": 123,
  "kind": "generate_animation",
  "prompt": "Raise the longsword and slash forward, follow through with the swing, then return to the initial idle stance",
  "parameters": {
    "animation_name": "sword_slash",
    "direction": "front",
    "style": "clean 2D pixel-art game character",
    "frame_count": 16,
    "columns": 4,
    "frame_width": 64,
    "frame_height": 64,
    "fps": 10,
    "resolution": "720p",
    "duration": 5,
    "aspect_ratio": "1:1"
  }
}
```

For an object animation such as opening a treasure chest, select the object view with the same `direction` field:

```json
{
  "assetId": 456,
  "kind": "generate_animation",
  "prompt": "Slowly open the chest lid, hold briefly, then close it and return to the initial closed pose",
  "parameters": {
    "animation_name": "open",
    "direction": "front",
    "frame_count": 8,
    "columns": 4,
    "frame_width": 64,
    "frame_height": 64,
    "fps": 10,
    "resolution": "720p",
    "duration": 5,
    "aspect_ratio": "1:1"
  }
}
```

The task payload is normalized to:

```json
{
  "animation_name": "sword_slash",
  "project_id": 42,
  "asset_id": 123,
  "direction": "front",
  "creative_brief": "Raise the longsword and slash forward, follow through with the swing, then return to the initial idle stance",
  "style": "clean 2D pixel-art game character",
  "frame_count": 16,
  "columns": 4,
  "frame_width": 64,
  "frame_height": 64,
  "fps": 10,
  "resolution": "720p",
  "duration": 5,
  "aspect_ratio": "1:1"
}
```

The outer `assetId` maps to the internal `asset_id`, and the outer `prompt` maps to `creative_brief`. The frontend should not duplicate `asset_id` inside `parameters`.

## 3. Parameter Reference

| Parameter | Purpose | Default / Constraints |
| --- | --- | --- |
| `assetId` | Existing character or object asset ID | Required, greater than 0 |
| `animation_name` | Animation clip name | Required |
| `direction` | English direction name for character or object assets | Required; resolved against the asset's 2/4/8-direction prototype order |
| `prompt` | Full action semantics | Falls back to `idle` internally when empty, but callers should provide an explicit value |
| `style` | Art style for video generation | Defaults to a production-grade 2D game asset style |
| `frame_count` | Final keyframe count | Default 16, range 1–32 |
| `columns` | Spritesheet column count | Default 4, range 1–8, total rows must not exceed 8 |
| `frame_width` | Final single-frame width | Default 256, range 32–1024 |
| `frame_height` | Final single-frame height | Default 256, range 32–1024 |
| `fps` | Frame playback metadata | Default 10, range 1–60 |
| `resolution` | Source video resolution for the video service | Default `720p` |
| `duration` | Source video duration in seconds for the video service | Default 5, range 4–15 |
| `aspect_ratio` | Video service aspect ratio | Default `1:1` |

`fps` does not affect the video service's original generation frame rate, nor does it change the final image count. The final image count is determined by `frame_count`; `fps` is primarily used to compute `frame.Duration ≈ 1000 / fps`.

## 4. Single-Subject High-Resolution Reference Image

Runtime-facing prototypes are typically already background-removed and compressed to 32×32 or 64×64. They are suitable for game display but not as high-quality identity references for a video model. Both character and object animation select one direction from the asset prototype array, such as the front view of a closed treasure chest.

The formal animation pipeline retrieves the reference image according to the following rules:

1. `AssetWriter.GetDetail(asset_id)` fetches the asset.
2. Decode `AssetContent`.
3. Validate the English `direction` against the character or object asset's 2/4/8-direction layout.
4. Use only the selected `content.Prototype[index]`; do not read other directions or a full multi-direction sheet.
5. Read the URL for that prototype.
6. Insert `-unprocessed` before the file extension to indicate the "uncompressed, background-removed" original image for the same prototype in the image hosting service.

Example:

```text
https://cdn.example.com/hero/direction_00.png
→ https://cdn.example.com/hero/direction_00-unprocessed.png
```

When query parameters are present, only the URL path is modified:

```text
https://cdn.example.com/hero/direction_00.png?version=7
→ https://cdn.example.com/hero/direction_00-unprocessed.png?version=7
```

This way the video model sees exactly one canonical subject direction, whether the asset is a character or an object. It never receives the complete multi-direction sheet as the video reference.

### Image hosting integration

The formal generator receives the selected prototype URL, appends the
`-unprocessed` suffix, and resolves it through the injected read-only reference resolver.
The configured upload storage (currently Qiniu) signs the object URL; the
 generator then downloads the original image, validates it as a raster image,
normalizes it to PNG Base64, and continues through the existing deterministic
animation preprocessing path.

The storage boundary remains outside `workspace/asset`: the generator only
requires `AnimationReferenceResolver`:

```go
animations := generator.NewAnimationGenerationService(videos, processor, uploadStore)
```

The full multi-direction sheet is never passed to the video model. A missing
`-unprocessed` object, non-2xx response, oversized response, or invalid image fails the
animation request instead of silently falling back to a compressed prototype or
the full direction sheet.

## 5. Image Preprocessing Semantics

The `-unprocessed` image is assumed to be:

- Single-direction;
- High-resolution;
- Background already removed;
- Not yet compressed to final game frame dimensions.

After loading as base64, it should use:

```go
ReferenceImagePrepared: false
```

The `false` here does not mean "redraw." The animation service will not call imageclient to repaint the subject; instead, it reuses the processor to perform deterministic processing:

```text
Detect transparent background
→ Resize to 1024×1024 with AnimationFrameResizeOptions (256px safe margin)
→ Composite onto a solid green background
→ Pass to image-to-video
```

This preserves the original prototype's identity while providing the single-subject safe canvas that the video model requires.

The final character or object prototype must use the same proportional canvas contract.
For example, a 64×64 prototype uses a 16-pixel margin while its 1024×1024
video reference uses a 256-pixel margin. Both therefore keep the canonical
canonical subject in the centre half of the canvas. The unused area is motion
budget for arms, weapons, and tools; it is not removed during animation frame
normalization.

## 6. Complete Processing Pipeline

```text
HTTP Request(assetId, direction, prompt, parameters)
→ Engine.buildTaskPayload
→ CreateAnimationPayload(asset_id, English direction, animation_name, creative_brief, ...)
→ Task manager publishes generate_animation
→ Executor validates asset_id and animation_name
→ AssetWriter.GetDetail(asset_id)
→ character/object: content.Prototype[direction]
→ Append -unprocessed suffix to prototype URL
→ TODO: Image hosting loads uncompressed, background-removed single-direction image
→ processor.Resize to 1024×1024 and composite green-screen safe canvas
→ prompts.BuildAnimationVideo
→ video_client.Generate / Download
→ ffmpeg extracts candidate frames
→ Subject and safe boundary quality checks
→ Search for complete and loopable action intervals
→ Sample frame_count keyframes
→ processor.SplitImage(animation) removes the background, aligns the foot anchor, and preserves the full source-cell scale
→ Output transparent frames and spritesheet
→ AssetWriter.CreateAnimation(asset_id, animation_name)
→ AssetWriter.UpdateAnimationFrames(asset_id, animation_id, frames)
```

The animation record is only created after video generation, download, frame extraction, and image processing all succeed, avoiding orphaned animation records when the provider fails.

## 7. Module Boundaries

### `executor.go`

- Fetch the asset by `asset_id`;
- Select the character or object prototype by `direction`;
- Construct the `-unprocessed` URL;
- Build the `AnimationGenerationRequest`;
- Create the animation and write frames on success.

### `animation.go` / `animation_video.go`

- Parameter defaults and validation;
- Single-direction reference image safe-canvas preprocessing;
- Video invocation, download, and frame extraction;
- Quality checks, loopable interval search, and keyframe sampling;
- Call processor to output transparent frames and spritesheet.

### `prompts/animation.go`

Responsible only for video prompts; does not use action keyword regex for business classification.

### `video_client`

Responsible only for provider communication; does not handle direction selection, image downloading, frame extraction, background removal, or asset writing.

### `processor/image`

Responsible only for deterministic image processing: background removal,
canonical padded resizing, animation source-cell scale preservation, anchor
normalization, and spritesheet packing. The animation generator enables
`PreserveSourceCellScale`; it must not refit the union of all poses because a
a long weapon or moving object part would shrink the subject in every final frame.

### `workspace/asset`

This module's data structures and business logic are not modified in this change. The generator only reads the existing `AssetContent.Prototype` array.

## 8. Compatibility Behavior

- Character and object animation assets must have `direction_count` equal to `2`, `4`, or `8`.
- Both asset types require an English `direction`.
- The generator maps the direction name to the corresponding `Prototype` array index and never uses a full multi-direction `metadata.animation_reference`.
- Assets that are missing a target prototype, have an empty URL, or provide an unavailable direction fail before video generation and asset writing.

## 9. Verification

The local example can slice out a specified direction directly from a multi-direction spritesheet without depending on the not-yet-implemented image hosting:

```bash
go run ./examples/generate_animation \
  -input /absolute/path/to/character_8_directions_green.png \
  -directions 8 \
  -direction front \
  -frame-size 64 \
  -prepare-only
```

Routine code verification:

```bash
go test ./internal/module/generator/...
go test ./examples/generate_animation
go vet ./...
```
