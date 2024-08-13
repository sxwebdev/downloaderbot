package main

import (
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
	"github.com/tkcrm/mx/cfg"
	"github.com/tkcrm/mx/launcher"
	"github.com/tkcrm/mx/logger"
	"github.com/tkcrm/mx/service"
	"github.com/tkcrm/mx/service/pingpong"
	"github.com/tkcrm/mx/transport/grpc_transport"
)

var (
	appName = "downloaderbot"
	version = "local"
)

func main() {
	logger := logger.NewExtended(
		logger.WithAppVersion(version),
		logger.WithAppName(appName),
	)

	conf := new(config.Config)
	if err := cfg.Load(conf); err != nil {
		logger.Fatalf("failed to load configuration: %s", err)
	}

	ln := launcher.New(
		launcher.WithName(appName),
		launcher.WithLogger(logger),
		launcher.WithVersion(version),
		launcher.WithRunnerServicesSequence(launcher.RunnerServicesSequenceFifo),
		launcher.WithOpsConfig(conf.Ops),
		launcher.WithAppStartStopLog(true),
	)

	rd, err := dragonfly.New(ln.Context(), conf.Redis, logger)
	if err != nil {
		logger.Fatalf("failed to init redis connection: %s", err)
	}

	// init limiter
	lm, err := limiter.New(logger, conf.Limiter, rd.Conn)
	if err != nil {
		logger.Fatalf("failed to init limiter: %s", err)
	}

	if err := lm.RegisterServices(
		limiter.NewService(telegram.ServiceName, limiter.WithFormattedLimit("10-M")),
	); err != nil {
		logger.Fatalf("failed to register limiter services: %s", err)
	}

	// services
	filesService, err := files.New(ln.Context(), logger, conf)
	if err != nil {
		logger.Fatalf("failed to init files service: %s", err)
	}

	proxyService := proxy.New(logger, conf)
	parserService := parser.New(logger, conf, filesService)
	telegramService := telegram.New(logger, conf, parserService, lm)
	tm := taskmanager.New(logger, taskmanager.Config{
		UniqueTasks: true,
		RedisConfig: taskmanager.RedisConfig{
			Addr:     conf.Redis.Addr,
			Username: conf.Redis.User,
			Password: conf.Redis.Password,
			DB:       conf.Redis.DbIndex,
		},
	})
	daemons := daemons.New(logger, conf, tm, filesService)

	// grpc servers
	botGrpcServer := api.NewBotGrpcServer(parserService)

	// grpc instance
	grpcServer := grpc_transport.NewServer(
		grpc_transport.WithLogger(logger),
		grpc_transport.WithConfig(conf.Grpc),
		grpc_transport.WithServices(botGrpcServer),
	)

	ln.ServicesRunner().Register(
		service.New(service.WithService(rd), service.WithName("redis")),
		service.New(service.WithService(tm)),
		service.New(service.WithService(grpcServer)),
		service.New(service.WithService(proxyService)),
		service.New(service.WithService(telegramService)),
		service.New(service.WithService(daemons)),
		service.New(service.WithService(pingpong.New(logger))),
	)

	if err := ln.Run(); err != nil {
		logger.Fatal(err)
	}
}
