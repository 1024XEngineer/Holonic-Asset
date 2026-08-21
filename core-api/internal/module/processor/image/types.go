package image

const (
	DefaultMatteColor               = "#00ff00"
	DefaultReferenceMaxEdge         = 768
	DefaultChromaThreshold          = 28.0
	DefaultChromaSoftness           = 34.0
	DefaultSpillSuppression         = 0.85
	TransparentAlphaMax       uint8 = 5
	NontransparentAlphaMin    uint8 = 20
	MinTransparentRatio             = 0.005
	StrictMinTransparentRatio       = 0.05
	MinOpaqueAlpha            uint8 = 250
)

type MatteColor [3]uint8

type Method string

const (
	MethodAuto   Method = "auto"
	MethodChroma Method = "chroma"
	MethodDual   Method = "dual"
)

type Profile string

const (
	ProfileGeneric Profile = "generic"
	// ProfileOpaqueBackground accepts a non-empty PNG that covers the full
	// canvas. Transparent pixels indicate invalid letterboxing or holes.
	ProfileOpaqueBackground Profile = "opaque-background"
	ProfileIcon             Profile = "icon"
	ProfileProduct          Profile = "product"
	ProfileSticker          Profile = "sticker"
	ProfileSeal             Profile = "seal"
	ProfileTranslucent      Profile = "translucent"
	ProfileGlow             Profile = "glow"
	ProfileShadow           Profile = "shadow"
	ProfileEffect           Profile = "effect"
)

type Material string

const (
	MaterialStandard Material = "standard"
	MaterialSoft3D   Material = "soft-3d"
	MaterialFlatIcon Material = "flat-icon"
	MaterialSticker  Material = "sticker"
	MaterialGlow     Material = "glow"
)

type ChromaSettings struct {
	Threshold        float64  `json:"threshold"`
	Softness         float64  `json:"softness"`
	SpillSuppression float64  `json:"spill_suppression"`
	Material         Material `json:"material,omitempty"`
}

func DefaultChromaSettings() ChromaSettings {
	return ChromaSettings{
		Threshold:        DefaultChromaThreshold,
		Softness:         DefaultChromaSoftness,
		SpillSuppression: DefaultSpillSuppression,
	}
}

func ChromaSettingsForMaterial(material Material) ChromaSettings {
	settings := DefaultChromaSettings()
	settings.Material = material
	switch material {
	case MaterialSoft3D:
		settings.Threshold = 60
		settings.Softness = 40
		settings.SpillSuppression = 0.20
	case MaterialFlatIcon:
		settings.Threshold = 32
		settings.Softness = 28
		settings.SpillSuppression = 0.75
	case MaterialSticker:
		settings.Threshold = 45
		settings.Softness = 38
		settings.SpillSuppression = 0.45
	case MaterialGlow:
		settings.Threshold = 18
		settings.Softness = 62
		settings.SpillSuppression = 0.15
	case MaterialStandard, "":
		settings.Material = ""
	}
	return settings
}

type VerificationOptions struct {
	Profile            Profile     `json:"profile"`
	ExpectedMatteColor *MatteColor `json:"expected_matte_color,omitempty"`
}

type AlphaBoundingBox struct {
	X      int `json:"x"`
	Y      int `json:"y"`
	Width  int `json:"width"`
	Height int `json:"height"`
}

type DualAlignmentReport struct {
	Score              float64 `json:"score"`
	Passed             bool    `json:"passed"`
	NegativeDeltaRatio float64 `json:"negative_delta_ratio"`
	DeltaChannelNoise  float64 `json:"delta_channel_noise"`
	ColorSpace         string  `json:"color_space"`
}

type ExtractionReport struct {
	Method                      Method               `json:"method"`
	MatteColor                  string               `json:"matte_color,omitempty"`
	MatteColorSource            string               `json:"matte_color_source,omitempty"`
	Threshold                   float64              `json:"threshold,omitempty"`
	Softness                    float64              `json:"softness,omitempty"`
	SpillSuppression            float64              `json:"spill_suppression,omitempty"`
	Material                    Material             `json:"material,omitempty"`
	MatteDecontaminationApplied bool                 `json:"matte_decontamination_applied"`
	RGBScrubbed                 bool                 `json:"rgb_scrubbed"`
	EdgeNoisePixelsRemoved      uint64               `json:"edge_noise_pixels_removed,omitempty"`
	FallbackApplied             bool                 `json:"fallback_applied,omitempty"`
	DualAlignment               *DualAlignmentReport `json:"dual_alignment,omitempty"`
}

type VerificationReport struct {
	Profile                  Profile           `json:"profile"`
	Width                    int               `json:"width"`
	Height                   int               `json:"height"`
	IsPNG                    bool              `json:"is_png"`
	ColorType                string            `json:"color_type"`
	HasAlpha                 bool              `json:"has_alpha"`
	InputHasAlpha            bool              `json:"input_has_alpha"`
	AlphaMin                 uint8             `json:"alpha_min"`
	AlphaMax                 uint8             `json:"alpha_max"`
	TransparentPixels        uint64            `json:"transparent_pixels"`
	PartialPixels            uint64            `json:"partial_pixels"`
	OpaquePixels             uint64            `json:"opaque_pixels"`
	NontransparentPixels     uint64            `json:"nontransparent_pixels"`
	TransparentRatio         float64           `json:"transparent_ratio"`
	PartialRatio             float64           `json:"partial_ratio"`
	OpaqueRatio              float64           `json:"opaque_ratio"`
	EdgeNontransparentPixels uint64            `json:"edge_nontransparent_pixels"`
	EdgeNontransparentRatio  float64           `json:"edge_nontransparent_ratio"`
	TouchesEdge              bool              `json:"touches_edge"`
	EdgeMarginPx             *int              `json:"edge_margin_px,omitempty"`
	ComponentCount           uint64            `json:"component_count"`
	LargestComponentPixels   uint64            `json:"largest_component_pixels"`
	LargestComponentRatio    float64           `json:"largest_component_ratio"`
	StrayPixelCount          uint64            `json:"stray_pixel_count"`
	AlphaNoiseScore          float64           `json:"alpha_noise_score"`
	MatteResidueScore        *float64          `json:"matte_residue_score,omitempty"`
	MatteResidueChecked      bool              `json:"matte_residue_checked"`
	HaloScore                float64           `json:"halo_score"`
	TransparentRGBScrubbed   bool              `json:"transparent_rgb_scrubbed"`
	CheckerboardDetected     bool              `json:"checkerboard_detected"`
	AlphaHealthScore         float64           `json:"alpha_health_score"`
	ResidueScore             float64           `json:"residue_score"`
	QualityScore             float64           `json:"quality_score"`
	BBox                     *AlphaBoundingBox `json:"bbox,omitempty"`
	Passed                   bool              `json:"passed"`
	FailureReasons           []string          `json:"failure_reasons"`
	Warnings                 []string          `json:"warnings"`
}

// RemoveBackgroundRequest configures local chroma-key background removal.
// ImageBase64 accepts raw Base64 or a data URL. MatteColor accepts a named
// colour, #RRGGBB, or "auto" for edge sampling. AllowSampledMatteFallback lets
// callers explicitly recover when a supplied matte does not match the source.
type RemoveBackgroundRequest struct {
	ImageBase64               string   `json:"image_base64"`
	MatteColor                string   `json:"matte_color,omitempty"`
	AllowSampledMatteFallback bool     `json:"allow_sampled_matte_fallback,omitempty"`
	Material                  Material `json:"material,omitempty"`
	Threshold                 *float64 `json:"threshold,omitempty"`
	Softness                  *float64 `json:"softness,omitempty"`
	SpillSuppression          *float64 `json:"spill_suppression,omitempty"`
}

type RemoveBackgroundResult struct {
	ImageBase64 string           `json:"image_base64"`
	MIMEType    string           `json:"mime_type"`
	Report      ExtractionReport `json:"report"`
}

// NormalizeReferenceRequest prepares a small raster reference for a visual
// model without inventing pixels. MaxEdge limits the longest output edge and
// defaults to DefaultReferenceMaxEdge.
type NormalizeReferenceRequest struct {
	ImageBase64 string `json:"image_base64"`
	MaxEdge     int    `json:"max_edge,omitempty"`
}

type ReferenceNormalizationReport struct {
	InputWidth   int  `json:"input_width"`
	InputHeight  int  `json:"input_height"`
	OutputWidth  int  `json:"output_width"`
	OutputHeight int  `json:"output_height"`
	Scale        int  `json:"scale"`
	Upscaled     bool `json:"upscaled"`
}

type NormalizeReferenceResult struct {
	ImageBase64 string                       `json:"image_base64"`
	MIMEType    string                       `json:"mime_type"`
	Report      ReferenceNormalizationReport `json:"report"`
}

// ResizeRequest converts a Base64 image to the requested final canvas.
type ResizeRequest struct {
	ImageBase64 string        `json:"image_base64"`
	Options     ResizeOptions `json:"options"`
}

type ResizeResult struct {
	ImageBase64 string       `json:"image_base64"`
	MIMEType    string       `json:"mime_type"`
	Report      ResizeReport `json:"report"`
}

// FlipHorizontalRequest mirrors an image around its vertical centre line.
type FlipHorizontalRequest struct {
	ImageBase64 string `json:"image_base64"`
}

// FlipHorizontalResult contains the mirrored PNG image.
type FlipHorizontalResult struct {
	ImageBase64 string `json:"image_base64"`
	MIMEType    string `json:"mime_type"`
}

// VerifyRequest validates a Base64 image without modifying it.
type VerifyRequest struct {
	ImageBase64        string  `json:"image_base64"`
	Profile            Profile `json:"profile,omitempty"`
	ExpectedMatteColor string  `json:"expected_matte_color,omitempty"`
}
