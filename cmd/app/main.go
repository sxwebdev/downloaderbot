package main

import (
	"fmt"

	"github.com/sxwebdev/downloaderbot/internal/api"
	"github.com/sxwebdev/downloaderbot/internal/config"
	"github.com/sxwebdev/downloaderbot/internal/daemons"
	"github.com/sxwebdev/downloaderbot/internal/proxy"
	"github.com/sxwebdev/downloaderbot/internal/services/files"
	"github.com/sxwebdev/downloaderbot/internal/services/parser"
	"github.com/sxwebdev/downloaderbot/internal/services/telegram"
	"github.com/tkcrm/modules/pkg/db/dragonfly"
	"github.com/tkcrm/modules/pkg/limiter"
	"github.com/tkcrm/modules/pkg/taskmanager"
	"github.com/tkcrm/mx/launcher"
	"github.com/tkcrm/mx/logger"
	"github.com/tkcrm/mx/service"
	"github.com/tkcrm/mx/service/pingpong"
	"github.com/tkcrm/mx/transport/grpc_transport"
)

var (
	appName    = "downloaderbot"
	version    = "local"
	commitHash = "unknown"
	buildDate  = "unknown"
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
	conf, err := config.Load[config.Config]([]string{"config.yaml"}, "")
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

	rd, err := dragonfly.New(ln.Context(), conf.Redis, l)
	if err != nil {
		return fmt.Errorf("failed to init redis connection: %w", err)
	}

	// init limiter
	lm, err := limiter.New(l, conf.Limiter, rd.Conn)
	if err != nil {
		return fmt.Errorf("failed to init limiter: %w", err)
	}

	if err := lm.RegisterServices(
		limiter.NewService(telegram.ServiceName, limiter.WithFormattedLimit("10-M")),
	); err != nil {
		return fmt.Errorf("failed to register limiter services: %w", err)
	}

	// services
	filesService, err := files.New(ln.Context(), l, conf)
	if err != nil {
		return fmt.Errorf("failed to init files service: %w", err)
	}

	proxyService := proxy.New(l, conf)
	parserService := parser.New(l, conf, filesService)
	telegramService := telegram.New(l, conf, parserService, lm)
	tm := taskmanager.New(l, taskmanager.Config{
		UniqueTasks: true,
		RedisConfig: taskmanager.RedisConfig{
			Addr:     conf.Redis.Addr,
			Username: conf.Redis.User,
			Password: conf.Redis.Password,
			DB:       conf.Redis.DbIndex,
		},
	})
	daemons := daemons.New(l, conf, tm, filesService)
	// grpc servers
	botGrpcServer := api.NewBotGrpcServer(parserService)

	// grpc instance
	grpcServer := grpc_transport.NewServer(
		grpc_transport.WithLogger(l),
		grpc_transport.WithConfig(conf.Grpc),
		grpc_transport.WithServices(botGrpcServer),
	)

	ln.ServicesRunner().Register(
		service.New(service.WithService(pingpong.New(l))),
		service.New(service.WithService(rd), service.WithName("redis")),
		service.New(service.WithService(tm)),
		service.New(service.WithService(grpcServer)),
		service.New(service.WithService(proxyService)),
		service.New(service.WithService(telegramService)),
		service.New(service.WithService(daemons)),
	)

	return ln.Run()
}
