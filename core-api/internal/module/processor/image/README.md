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
- `allow_sampled_matte_fallback`: when true, a supplied matte that produces no usable transparent subject may be replaced by an edge-sampled matte; inspect `fallback_applied` and `matte_color_source` in the report.
- `material`: selects a threshold preset based on the material type.
- `threshold`, `softness`, `spill_suppression`: override the preset parameters.

Processing selects between two internal chroma paths without changing the
public API. Subjects with substantial key-coloured content use global Euclidean
distance alpha and matte decontamination, which also clears enclosed background
regions without applying key-dominance alpha to the subject. Other images use a
border-connected soft matte with key-dominance alpha, partial-alpha despill,
one-pixel edge contraction, and light alpha feathering. Both paths finish with
edge speckle cleanup and transparent-pixel RGB cleanup.

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
- When content cropping is enabled, the alpha-bounded subject is selected
  without trimming it to the target aspect ratio.
- The selected content is resized with contain semantics: its aspect ratio is
  preserved, it is centred, and any unused target area remains transparent.
- A transparent safety margin is automatically added.
- Downscaling uses alpha-weighted area sampling in straight-alpha colour space;
  enlargement uses alpha-weighted bilinear sampling. This prevents dark or
  contaminated RGB from bleeding through transparent edges.
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
registers a shared root anchor, and returns fixed-size frames with a common
coordinate system. For video-generated frames, preserve the source-cell scale
so the longest weapon pose cannot make the character body smaller.

```go
result, err := processor.SplitImage(ctx, &image.SplitImageRequest{
    ImageBase64: sourceSheetBase64,
    Mode:        image.ImageSplitModeAnimation,
    Columns:     4,
    Rows:        2,
    FrameCount:  8,
    FrameWidth:  256,
    FrameHeight: 256,
    Margin:      image.AnimationFrameMargin(256, 256),
    Anchor:      image.AnimationAnchorFeet,
    PreserveSourceCellScale: true,
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

`PreserveSourceCellScale` is the recommended setting for video-to-spritesheet
processing. It renders the complete source grid cell into the final frame with
one fixed cell-to-frame scale. It deliberately does **not** fit the union of all
visible poses. Consequently, a sword, staff, or tool extending farther in one
pose does not shrink the character in every returned frame.

The canonical prototype and the video reference must use the same padded-frame
contract. Prepare both with `AnimationFrameResizeOptions`; for example, a 64x64
prototype uses a 16-pixel safety margin and its 1024x1024 video reference uses a
256-pixel safety margin:

```go
prototypeOptions := image.AnimationFrameResizeOptions(64, 64)
referenceOptions := image.AnimationFrameResizeOptions(1024, 1024)
```

This space must be reserved before video generation. `SplitImage` can preserve
the scale and register the root anchor, but it cannot recover weapon pixels that
the video provider already rendered outside the source frame.

For a static multi-direction character or object sheet, an image model may draw
the same subject at different apparent sizes in different cells. Opt in to
content-scale normalization for that input only:

```go
NormalizeContentScale: true,
```

This rescales each visible cell to the median source content height before
anchor registration, then returns the requested fixed-size canvases. It is not
intended for action frames, where silhouette changes can be part of the motion.
`NormalizeContentScale`, `NormalizeContentArea`, and `PreserveSourceCellScale`
are mutually exclusive. Use `NormalizeContentScale` for characters and
`NormalizeContentArea` for static objects whose directional views can have
different aspect ratios but should occupy the same visual footprint. For a
static object sheet, also set `CenterContent: true`: anchor registration alone
can leave a direction's silhouette bbox off-centre when the generated views have
different internal geometry. `CenterContent` only translates each final frame;
it does not crop, rescale, recolour, or remove pixels, and must not be used for
frames whose intentional action displacement needs to be preserved.

The animation pipeline:

1. Splits the known grid with fixed proportional source cells by default.
2. Removes a configured or automatically detected flat background.
3. Optionally normalizes static multi-direction subjects to the median visible
   height when `NormalizeContentScale` is enabled.
4. Estimates one robust root anchor per frame and translates frames to one
   common integer target. Set `PreserveHorizontalMotion` or
   `PreserveVerticalMotion` when motion on that axis is intentional.
5. Computes one union bounding box after registration for diagnostics and for
   the legacy shared-union fitting mode.
6. With `PreserveSourceCellScale`, renders the full source cell using its fixed
   cell-to-frame scale. Otherwise, applies the legacy shared union crop and one
   global fit scale.
7. Returns the normalized spritesheet in `ImageBase64`, same-size PNG frames in
   `Regions`, and the full measurements in `AnimationReport`.

`CropToContent` is rejected in animation mode because independent tight crops
are exactly what create playback displacement. `DetectGridBounds` remains
opt-in because pose silhouettes are not reliable cell separators.

The normalization engine is private to the processor. There is no second
public animation endpoint: callers always use
`SplitImage(ImageSplitModeAnimation)`.

For generated prototypes, `PrototypePixelResizeOptions` and
`CharacterPrototypePixelResizeOptions` use the same target-size palette budget
and enable the dedicated `SpritePixelPipeline`. This profile intentionally does
not run generic object contour repair, round-shape regularization, isolated
component deletion, or colour-island consolidation. Those heuristics can make a
basketball oval, erase a thin blade joint, or turn a valid internal line into a
random colour block. Instead it follows the safer ordering used by dedicated
pixel-art converters:

1. hard-threshold the alpha channel at the converter's 128 cutoff;
2. quantize the source colours with the same weighted median-cut and eight-pass
   centroid refinement structure used by the browser converter;
3. for standalone content conversion, crop to visible content and fit it on a
   4x intermediate canvas; for a pre-padded prototype frame, preserve the
   complete canvas geometry instead of refitting its alpha bounds;
4. centre the intermediate result and reduce it with floor-based nearest
   sampling; and
5. scrub transparent RGB without inventing geometry.

The generator uses two geometry contracts. Character frames already contain a
shared pose canvas with action/safety space, so their final pixel pass sets
`PreserveCanvasGeometry` and does not refit the alpha bounds. Object frames also
need animation safety space, but their generated direction cells can contain
large, inconsistent transparent borders. They therefore crop to the visible
alpha bounds, fill the inner canvas (`target - 2*margin`) with the Sprite-like
content-fit pass, and finally place that result on the complete 32/64px canvas
with a transparent outer safety margin. Neither contract permits the subject to
touch the final canvas edge.

Direction frames receive a final conservative colour canonicalization pass that
merges only near-duplicate colours and never moves pixels or changes silhouette
geometry.

### Prototype Sprite-AI-compatible fitting

The prototype profile now uses a dedicated `SpritePixelPipeline` rather than
trying to repair the final image with object-specific contour heuristics. Its
geometry stage is intentionally compatible with the public browser converter
used as the reference implementation:

- source alpha is made binary before palette reduction;
- each direction is quantized independently, so every frame receives the full
  palette budget and a thin direction-specific seam cannot be displaced by
  colours from another frame;
- visible content is fitted into a 4x intermediate canvas before final reduction;
- nearest sampling uses the source-cell floor rule instead of the processor's
  centre-sampling nearest rule; and
- no isolated-component deletion, round-shape forcing, colour-island merging, or
  contour regularization runs afterward.

`RecoverPixelGrid` remains available for generic pixel callers and tests, but it
is not the active geometry path when `SpritePixelPipeline` is enabled. This is
deliberate: the browser converter does not infer a hidden logical grid from
edge energy; it quantizes, fits, and nearest-resamples the visible image.

This pass is intentionally limited to prototype pixel options. General image
resizing and non-integral inputs retain the prior area behaviour. The sprite
profile stops after quantization, grid/nearest sampling, and hard-alpha cleanup;
it does not run shape-repair heuristics after sampling.
