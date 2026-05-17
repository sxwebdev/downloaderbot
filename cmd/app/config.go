package main

import (
	"bytes"
	"context"
	"fmt"
	"html/template"
	"os"

	"github.com/goccy/go-yaml"
	"github.com/sxwebdev/downloaderbot/internal/config"
	"github.com/sxwebdev/downloaderbot/templates"
	"github.com/sxwebdev/xconfig"
	"github.com/sxwebdev/xconfig/plugins/env"
	"github.com/urfave/cli/v3"
)

func cfgPathsFlag() *cli.StringSliceFlag {
	return &cli.StringSliceFlag{
		Name:    "configs",
		Aliases: []string{"c"},
		Usage:   "allows you to use your own paths to configuration files, separated by commas (config.yaml,config.prod.yml)",
		Value:   cli.NewStringSlice("config.yaml").Value(),
	}
}

func configCMD() *cli.Command {
	return &cli.Command{
		Name:  "config",
		Usage: "validate, gen envs and flags for config",
		Commands: []*cli.Command{
			{
				Name:  "genenvs",
				Usage: "generate markdown for all envs and config yaml template",
				Action: func(_ context.Context, cl *cli.Command) error {
					conf := new(config.Config)

					_, err := xconfig.Load(conf, xconfig.WithEnvPrefix(envPrefix))
					if err != nil {
						return fmt.Errorf("failed to generate markdown: %w", err)
					}

					buf := bytes.NewBuffer(nil)
					enc := yaml.NewEncoder(buf, yaml.Indent(2))
					if err := enc.Encode(conf); err != nil {
						return fmt.Errorf("failed to encode yaml: %w", err)
					}
					if err := enc.Close(); err != nil {
						return fmt.Errorf("failed to close encoder: %w", err)
					}

					if err := os.WriteFile("config.template.yaml", buf.Bytes(), 0o600); err != nil {
						return fmt.Errorf("failed to write file: %w", err)
					}

					// generate ENVS.md
					envMarkdown, err := xconfig.GenerateMarkdown(
						conf,
						xconfig.WithEnvPrefix(envPrefix),
						xconfig.WithPlugins(
							env.New(envPrefix),
						),
					)
					if err != nil {
						return fmt.Errorf("failed to generate markdown: %w", err)
					}

					output := new(bytes.Buffer)
					cl.Root().Writer = output
					if err := cli.ShowAppHelp(cl.Root()); err != nil {
						return err
					}

					tmpl, err := template.ParseFS(templates.EnvsFS, "ENVS.go.tmpl")
					if err != nil {
						return err
					}

					data := struct {
						VaultEnvironments string
						AppEnvironments   string
					}{
						AppEnvironments: envMarkdown,
					}

					buf = bytes.NewBuffer(nil)
					if err := tmpl.ExecuteTemplate(buf, "envs", data); err != nil {
						return err
					}

					if err := os.WriteFile("ENVS.md", buf.Bytes(), 0o600); err != nil {
						return err
					}

					return nil
				},
			},
		},
	}
}
