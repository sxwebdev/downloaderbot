package config

import (
	"github.com/sxwebdev/downloaderbot/pkg/s3"
	"github.com/tkcrm/modules/pkg/db/dragonfly"
	"github.com/tkcrm/modules/pkg/limiter"
	"github.com/tkcrm/mx/logger"
	"github.com/tkcrm/mx/ops"
	"github.com/tkcrm/mx/transport/grpc_transport"
)

// Config ...
type Config struct {
	EnvCI               string `validate:"required" env:"ENV_CI" example:"dev"`
	ServiceName         string `default:"downloaderbot" validate:"required"`
	Log                 logger.Config
	Ops                 ops.Config
	Redis               dragonfly.Config
	Grpc                grpc_transport.Config
	S3                  s3.Config `env:"S3"`
	Limiter             limiter.Config
	TelegramBotApiToken string `validate:"required" secret:"true" usage:"use token for your telegram bot"`
	S3BaseUrl           string `env:"S3_BASE_URL" validate:"required" default:"http://localhost:9050"`
}
