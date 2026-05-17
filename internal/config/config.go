package config

import (
	"github.com/tkcrm/mx/launcher/ops"
	"github.com/tkcrm/mx/logger"
	"github.com/tkcrm/mx/transport/grpc_transport"
)

// Config ...
type Config struct {
	Log                 logger.Config
	Ops                 ops.Config
	Grpc                grpc_transport.Config
	TelegramBotApiToken string `yaml:"telegram_bot_api_token" validate:"required" secret:"true" usage:"use token for your telegram bot"`
}
