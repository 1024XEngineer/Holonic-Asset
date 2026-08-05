# Image Processing Module

This directory provides only local, deterministic image processing capabilities — no image generation models, prompts, providers, or generation tasks:

1. Background removal: `RemoveBackground`
2. Resize: `Resize`
3. Quality verification: `Verify`
4. Generic image slicing: `SplitImage`
5. Stable animation-sheet processing: `SplitImage` with animation mode

## Data Conventions

- The input image field is uniformly `image_base64`.
- Input accepts plain Base64 as well as `data:image/...;base64,...` Data URLs.
- Background removal and resizing uniformly return plain Base64-encoded PNG, with `mime_type` fixed to `image/png`.
- Quality verification does not modify the image; it returns only a verification report without echoing back duplicate image data.
- The API does not read from or write to file paths — callers are responsible for object storage, HTTP uploads, or file I/O themselves.

## Interface

```go
type Processor interface {
    RemoveBackground(context.Context, *RemoveBackgroundRequest) (*RemoveBackgroundResult, error)
    Resize(context.Context, *ResizeRequest) (*ResizeResult, error)
    Verify(context.Context, *VerifyRequest) (*VerificationReport, error)
    SplitImage(context.Context, *SplitImageRequest) (*SplitImageResult, error)
}
```

Creating a processor:

```go
processor := image.NewProcessor()
```

The processor is stateless and can be safely injected as a dependency into the service layer.

## Background Removal

```go
result, err := processor.RemoveBackground(ctx, &image.RemoveBackgroundRequest{
    ImageBase64: sourceBase64,
    MatteColor:  "#ff00ff", // can also be "auto"
    Material:    image.MaterialFlatIcon,
})
if err != nil {
    return err
}
transparentPNGBase64 := result.ImageBase64
```

Optional parameters:

- `matte_color`: a named color, `#RRGGBB`, or `auto`; defaults to `#00ff00` when empty.
- `material`: selects a threshold preset based on the material type.
- `threshold`, `softness`, `spill_suppression`: override the preset parameters.

Processing includes color-key alpha extraction, matte decontamination, spill suppression, edge speckle cleanup, and transparent-pixel RGB cleanup.

## Resize

```go
options := image.DefaultResizeOptions(64, 64)
result, err := processor.Resize(ctx, &image.ResizeRequest{
    ImageBase64: transparentPNGBase64,
    Options:     options,
})
if err != nil {
    return err
}
image64Base64 := result.ImageBase64
```

Defaults are suitable for ordinary 2D game assets:

- Output dimensions are strictly equal to the specified width and height.
- The subject is cropped by alpha and its aspect ratio is preserved.
- A transparent safety margin is automatically added.
- Downscaling uses alpha-aware area sampling.
- Full-color and smooth semi-transparent edges are preserved.

Only set the following when you explicitly need traditional pixel art:

```go
options.Mode = image.RasterModePixel
options.PaletteSize = 24
options.HardAlpha = true
```

## Quality Verification

```go
report, err := processor.Verify(ctx, &image.VerifyRequest{
    ImageBase64:        image64Base64,
    Profile:            image.ProfileIcon,
    ExpectedMatteColor: "#ff00ff",
})
if err != nil {
    return err
}
if !report.Passed {
    // report.FailureReasons / report.Warnings
}
```

Verification checks include:

- PNG and alpha channel presence.
- Actual transparent pixel ratio.
- Subject bounding box, transparent margins, and edge-touching conditions.
- Matte residue, halos, and alpha noise.
- Whether transparent-pixel RGB has been cleaned.
- Checkerboard pseudo-transparent backgrounds.
- Profile-specific quality gates and scoring.

## Image Slicing

`SplitImage` is the single public entry point for both final animation frames
and generic static-region extraction. It does not write files or call a
generation provider; callers persist the returned Base64 values themselves.

### Animation sheets

Use `ImageSplitModeAnimation` for any frames that will be played as one
animation. This mode does more than cut the grid: it removes a flat background,
registers a shared root anchor, computes one shared crop, applies one global
scale, and returns fixed-size frames with a common coordinate system.

```go
result, err := processor.SplitImage(ctx, &image.SplitImageRequest{
    ImageBase64: sourceSheetBase64,
    Mode:        image.ImageSplitModeAnimation,
    Columns:     4,
    Rows:        2,
    FrameCount:  8,
    FrameWidth:  256,
    FrameHeight: 256,
    Anchor:      image.AnimationAnchorFeet,
})
if err != nil {
    return err
}
normalizedSheetBase64 := result.ImageBase64
normalizedFrames := result.Regions
report := result.AnimationReport
```

Opaque animation inputs automatically use edge-based matte detection when
`Background` is omitted. Set an explicit green-screen colour when known:

```go
Background: &image.AnimationBackgroundOptions{MatteColor: "#00ff00"},
```

The animation pipeline:

1. Splits the known grid with fixed proportional source cells by default.
2. Removes a configured or automatically detected flat background.
3. Estimates one robust root anchor per frame and translates frames to one
   common integer target. Set `PreserveHorizontalMotion` or
   `PreserveVerticalMotion` when motion on that axis is intentional.
4. Computes one union bounding box after registration.
5. Applies the same crop, one global uniform scale, and one destination to all
   frames.
6. Returns the normalized spritesheet in `ImageBase64`, same-size PNG frames in
   `Regions`, and the full measurements in `AnimationReport`.

`CropToContent` is rejected in animation mode because independent tight crops
are exactly what create playback displacement. `DetectGridBounds` remains
opt-in because pose silhouettes are not reliable cell separators.

The normalization engine is private to the processor. There is no second
public animation endpoint: callers always use
`SplitImage(ImageSplitModeAnimation)`.

### Static images and structural extraction

The other modes intentionally do not change placement inside a source cell:

- `ImageSplitModeGrid`: fixed grid cells for independent tiles, icons, cards,
  or other static assets.
- `ImageSplitModeComponents`: one tight image per 8-connected visible region.
- `ImageSplitModeProjection`: starts with alpha-connected components, expands
  their bounds by `ProjectionMergeGap`, and uses union-find to merge nearby
  body, weapon, shadow, or effect pieces. A zero gap selects a size-based
  default; set it explicitly when generated object groups are densely packed.

These modes are not animation stabilizers. `CropToContent` is valid for
independent static assets, where each output does not need to share a playback
coordinate system. If `mode` is omitted, the processor uses components mode
unless `columns` or `rows` is provided, in which case it defaults to animation
mode. Static grid callers must explicitly set `Mode: ImageSplitModeGrid`.