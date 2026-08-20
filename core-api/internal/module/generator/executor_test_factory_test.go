package generator

import (
	"net/http"

	"github.com/1024XEngineer/Holonic-Asset/internal/module/generator/imageclient"
	imageprocessor "github.com/1024XEngineer/Holonic-Asset/internal/module/processor/image"
)

// NewExecutorForTest constructs an executor without depending on the public
// constructor API, which may differ between the feature branch and its CI merge base.
func NewExecutorForTest(
	images imageclient.ImageGenerationService,
	processor imageprocessor.Processor,
	assets AssetWriter,
	references ReferenceStore,
) Executor {
	return &executor{
		images:     images,
		processor:  processor,
		assets:     assets,
		references: references,
	}
}

// NewExecutorWithPrototypeReferenceHTTPClientForTest injects transport behavior
// without exposing the production SSRF boundary as a runtime dependency.
func NewExecutorWithPrototypeReferenceHTTPClientForTest(
	images imageclient.ImageGenerationService,
	processor imageprocessor.Processor,
	assets AssetWriter,
	references ReferenceStore,
	client *http.Client,
) Executor {
	return &executor{
		images:              images,
		processor:           processor,
		assets:              assets,
		references:          references,
		referenceHTTPClient: client,
	}
}
