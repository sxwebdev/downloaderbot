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
	if err := DefaultRegistry.Register(extInstagram.New()); err != nil {
		panic("register instagram extractor: " + err.Error())
	}

	// Register YouTube extractor (custom implementation)
	if err := DefaultRegistry.Register(extYoutube.New()); err != nil {
		panic("register youtube extractor: " + err.Error())
	}

	// Register all lux-based extractors
	for _, ext := range extLux.GetAllExtractors() {
		if err := DefaultRegistry.Register(ext); err != nil {
			panic("register lux extractor " + ext.Name() + ": " + err.Error())
		}
	}
}

// GetRegistry returns the default extractor registry
func GetRegistry() *Registry {
	return DefaultRegistry
}
