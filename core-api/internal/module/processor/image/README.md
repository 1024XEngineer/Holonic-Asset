# Image Processing Module

This directory provides only local, deterministic image processing capabilities — no image generation models, prompts, providers, or generation tasks:

1. Background removal: `RemoveBackground`
2. Resize: `Resize`
3. Quality verification: `Verify`

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
sprite64Base64 := result.ImageBase64
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
    ImageBase64:        sprite64Base64,
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
