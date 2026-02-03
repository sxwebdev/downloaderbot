package extractor

import (
	"fmt"
	"net/url"
	"strings"
	"sync"

	"github.com/sxwebdev/downloaderbot/internal/models"
)

// Registry manages all registered extractors
type Registry struct {
	mu         sync.RWMutex
	extractors map[string]Extractor   // name -> extractor
	hostMap    map[string]Extractor   // host -> extractor
	sources    []models.MediaSource   // all registered sources
}

// NewRegistry creates a new extractor registry
func NewRegistry() *Registry {
	return &Registry{
		extractors: make(map[string]Extractor),
		hostMap:    make(map[string]Extractor),
		sources:    make([]models.MediaSource, 0),
	}
}

// Register adds an extractor to the registry
func (r *Registry) Register(ext Extractor) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	name := ext.Name()
	if _, exists := r.extractors[name]; exists {
		return fmt.Errorf("extractor %s already registered", name)
	}

	r.extractors[name] = ext
	r.sources = append(r.sources, models.MediaSource(name))

	for _, host := range ext.Hosts() {
		// Normalize host (remove www. prefix for matching)
		normalizedHost := normalizeHost(host)
		if _, exists := r.hostMap[normalizedHost]; exists {
			return fmt.Errorf("host %s already registered", host)
		}
		r.hostMap[normalizedHost] = ext
	}

	return nil
}

// GetByName returns an extractor by its name
func (r *Registry) GetByName(name string) (Extractor, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	ext, ok := r.extractors[name]
	return ext, ok
}

// GetByHost returns an extractor that handles the given host
func (r *Registry) GetByHost(host string) (Extractor, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	normalizedHost := normalizeHost(host)
	ext, ok := r.hostMap[normalizedHost]
	return ext, ok
}

// GetByURL parses the URL and returns the appropriate extractor
func (r *Registry) GetByURL(rawURL string) (Extractor, error) {
	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse URL: %w", err)
	}

	ext, ok := r.GetByHost(parsedURL.Host)
	if !ok {
		return nil, fmt.Errorf("no extractor found for host: %s", parsedURL.Host)
	}

	return ext, nil
}

// GetSupportedSources returns all registered media sources
func (r *Registry) GetSupportedSources() []models.MediaSource {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]models.MediaSource, len(r.sources))
	copy(result, r.sources)
	return result
}

// GetAllExtractors returns all registered extractors
func (r *Registry) GetAllExtractors() []Extractor {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]Extractor, 0, len(r.extractors))
	for _, ext := range r.extractors {
		result = append(result, ext)
	}
	return result
}

// GetSupportedHosts returns all supported hosts
func (r *Registry) GetSupportedHosts() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	hosts := make([]string, 0, len(r.hostMap))
	for host := range r.hostMap {
		hosts = append(hosts, host)
	}
	return hosts
}

// normalizeHost removes common prefixes from host for matching
func normalizeHost(host string) string {
	host = strings.ToLower(host)
	host = strings.TrimPrefix(host, "www.")
	host = strings.TrimPrefix(host, "m.")
	host = strings.TrimPrefix(host, "mobile.")
	return host
}
