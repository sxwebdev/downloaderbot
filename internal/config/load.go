package config

import (
	"fmt"

	"github.com/go-playground/validator/v10"
	"github.com/sxwebdev/xconfig"
	"github.com/sxwebdev/xconfig/decoders/xconfigyaml"
	"github.com/sxwebdev/xconfig/plugins/loader"
	"github.com/sxwebdev/xconfig/plugins/validate"
)

func Load[T any](configPaths []string, envPrefix string) (*T, error) {
	conf := new(T)

	loader, err := loader.NewLoader(map[string]loader.Unmarshal{
		"yaml": xconfigyaml.New().Unmarshal,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create config loader: %w", err)
	}

	for _, path := range configPaths {
		if err := loader.AddFile(path, true); err != nil {
			return nil, fmt.Errorf("failed to add config file %q: %w", path, err)
		}
	}

	_, err = xconfig.Load(conf,
		xconfig.WithDisallowUnknownFields(),
		xconfig.WithEnvPrefix(envPrefix),
		xconfig.WithLoader(loader),
		xconfig.WithPlugins(
			validate.New(func(a any) error {
				return validator.New().Struct(a)
			}),
		),
	)
	if err != nil {
		return nil, err
	}

	return conf, nil
}
