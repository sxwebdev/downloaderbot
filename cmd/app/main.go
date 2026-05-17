package main

import (
	"fmt"

	"github.com/sxwebdev/downloaderbot/internal/api"
	"github.com/sxwebdev/downloaderbot/internal/config"
	"github.com/sxwebdev/downloaderbot/internal/daemons"
	"github.com/sxwebdev/downloaderbot/internal/limiter"
	"github.com/sxwebdev/downloaderbot/internal/proxy"
	"github.com/sxwebdev/downloaderbot/internal/services/files"
	"github.com/sxwebdev/downloaderbot/internal/services/parser"
	"github.com/sxwebdev/downloaderbot/internal/services/telegram"
	"github.com/tkcrm/mx/launcher"
	"github.com/tkcrm/mx/launcher/services/pingpong"
	"github.com/tkcrm/mx/logger"
	"github.com/tkcrm/mx/transport/grpc_transport"
)

var (
	appName    = "downloaderbot"
	version    = "local"
	commitHash = "unknown"
	// buildDate  = "unknown"
)

func getVersion() string { return version + "-" + commitHash }

func defaultLoggerOpts() []logger.Option {
	return []logger.Option{
		logger.WithAppName(appName),
		logger.WithAppVersion(getVersion()),
	}
}

func main() {
	l := logger.NewExtended(defaultLoggerOpts()...)
	if err := run(l); err != nil {
		l.Fatalf("application stopped with error: %s", err)
	}
}

func run(l logger.ExtendedLogger) error {
	conf, err := config.Load[config.Config]([]string{"config.yaml", ".env"}, "")
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
	filesService, err := files.New(ln.Context(), l, conf)
	if err != nil {
		return fmt.Errorf("failed to init files service: %w", err)
	}

	proxyService := proxy.New(l, conf)
	parserService := parser.New(l, conf, filesService)
	telegramService := telegram.New(l, conf, parserService, lm)
	daemons := daemons.New(l, conf, filesService)
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
		launcher.NewService(launcher.WithService(proxyService)),
		launcher.NewService(launcher.WithService(telegramService)),
		launcher.NewService(launcher.WithService(daemons)),
	)

	return ln.Run()
}
