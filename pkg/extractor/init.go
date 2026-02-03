package extractor

import (
	extInstagram "github.com/sxwebdev/downloaderbot/pkg/extractor/instagram"
	extLux "github.com/sxwebdev/downloaderbot/pkg/extractor/lux"
	extYoutube "github.com/sxwebdev/downloaderbot/pkg/extractor/youtube"
)

// DefaultRegistry is the global extractor registry
var DefaultRegistry = NewRegistry()

func init() {
	// Register Instagram extractor (custom implementation)
	_ = DefaultRegistry.Register(extInstagram.New())

	// Register YouTube extractor (custom implementation)
	_ = DefaultRegistry.Register(extYoutube.New())

	// Register all lux-based extractors
	for _, ext := range extLux.GetAllExtractors() {
		_ = DefaultRegistry.Register(ext)
	}
}

// GetRegistry returns the default extractor registry
func GetRegistry() *Registry {
	return DefaultRegistry
}
