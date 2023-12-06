package main

import (
	"github.com/sxwebdev/downloaderbot/internal/api"
	"github.com/sxwebdev/downloaderbot/internal/config"
	"github.com/sxwebdev/downloaderbot/internal/services/instagram"
	"github.com/tkcrm/mx/cfg"
	"github.com/tkcrm/mx/launcher"
	"github.com/tkcrm/mx/logger"
	"github.com/tkcrm/mx/service"
	"github.com/tkcrm/mx/service/pingpong"
	"github.com/tkcrm/mx/transport/grpc_transport"
)

var (
	appName = "micro-example"
	version = "local"
)

func main() {
	logger := logger.NewExtended(
		logger.WithAppVersion(version),
		logger.WithAppName(appName),
	)

	conf := new(config.Config)
	if err := cfg.Load(conf, cfg.WithVersion(version)); err != nil {
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

	// services
	instagramService := instagram.New(logger)

	// grpc servers
	botGrpcServer := api.NewBotGrpcServer(instagramService)

	// grpc instance
	grpcServer := grpc_transport.NewServer(
		grpc_transport.WithLogger(logger),
		grpc_transport.WithConfig(conf.Grpc),
		grpc_transport.WithServices(botGrpcServer),
	)

	ln.ServicesRunner().Register(
		service.New(service.WithService(grpcServer)),
		service.New(service.WithService(instagramService)),
		service.New(service.WithService(pingpong.New(logger))),
	)

	if err := ln.Run(); err != nil {
		logger.Fatal(err)
	}
}
