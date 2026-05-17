package main

import (
	"context"
	"fmt"

	"github.com/sxwebdev/downloaderbot/internal/api"
	"github.com/sxwebdev/downloaderbot/internal/config"
	"github.com/sxwebdev/downloaderbot/internal/limiter"
	"github.com/sxwebdev/downloaderbot/internal/services/parser"
	"github.com/sxwebdev/downloaderbot/internal/services/telegram"
	"github.com/tkcrm/mx/launcher"
	"github.com/tkcrm/mx/launcher/services/pingpong"
	"github.com/tkcrm/mx/logger"
	"github.com/tkcrm/mx/transport/grpc_transport"
	"github.com/urfave/cli/v3"
)

func startCMD(l logger.ExtendedLogger) *cli.Command {
	return &cli.Command{
		Name:  "start",
		Usage: "start the downloader bot",
		Flags: []cli.Flag{cfgPathsFlag()},
		Action: func(ctx context.Context, cl *cli.Command) error {
			conf, err := config.Load[config.Config]([]string{"config.yaml", ".env"}, envPrefix)
			if err != nil {
				return fmt.Errorf("failed to load configuration: %w", err)
			}

			loggerOpts := append(defaultLoggerOpts(), logger.WithConfig(conf.Log))

			l = logger.WithExtended(logger.NewExtended(loggerOpts...))
			defer func() {
				_ = l.Sync()
			}()

			ln := launcher.New(
				launcher.WithName(appName),
				launcher.WithLogger(l),
				launcher.WithVersion(version),
				launcher.WithRunnerServicesSequence(launcher.RunnerServicesSequenceFifo),
				launcher.WithOpsConfig(conf.Ops),
				launcher.WithAppStartStopLog(true),
			)

			// init limiter
			lm, err := limiter.New("10-M")
			if err != nil {
				return fmt.Errorf("failed to init limiter: %w", err)
			}

			// services
			parserService := parser.New(l, conf)
			telegramService := telegram.New(l, conf, parserService, lm)
			// grpc servers
			botGrpcServer := api.NewBotGrpcServer(parserService)

			// grpc instance
			grpcServer := grpc_transport.NewServer(
				grpc_transport.WithLogger(l),
				grpc_transport.WithConfig(conf.Grpc),
				grpc_transport.WithServices(botGrpcServer),
			)

			ln.ServicesRunner().Register(
				launcher.NewService(launcher.WithService(pingpong.New(l))),
				launcher.NewService(launcher.WithService(grpcServer)),
				launcher.NewService(launcher.WithService(telegramService)),
			)

			return ln.Run()
		},
	}
}
