package config

import (
	"github.com/tkcrm/modules/pkg/db/dragonfly"
	"github.com/tkcrm/mx/logger"
	"github.com/tkcrm/mx/ops"
	"github.com/tkcrm/mx/transport/grpc_transport"
)

// Config ...
type Config struct {
	EnvCI       string `validate:"required" env:"ENV_CI" example:"dev"`
	ServiceName string `default:"downloaderbot" validate:"required"`
	Log         logger.Config
	Ops         ops.Config
	Redis       dragonfly.Config
	Grpc        grpc_transport.Config
}
