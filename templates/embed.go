package templates

import "embed"

//go:embed README.go.tmpl
var ReadmeFS embed.FS

//go:embed ENVS.go.tmpl
var EnvsFS embed.FS
