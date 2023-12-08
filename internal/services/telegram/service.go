package telegram

import (
	"context"
	"fmt"
	"time"

	"github.com/sxwebdev/downloaderbot/internal/config"
	"github.com/sxwebdev/downloaderbot/internal/services/parser"
	"github.com/tkcrm/modules/pkg/limiter"
	"github.com/tkcrm/mx/logger"
	"github.com/tkcrm/mx/service"
	"gopkg.in/telebot.v3"
)

const ServiceName = "telegram-service"

type Service struct {
	logger logger.Logger
	config *config.Config
	name   string

	lim           limiter.ILimiter
	parserService *parser.Service

	bot *telebot.Bot
}

func New(l logger.Logger, cfg *config.Config, parserService *parser.Service, lim limiter.ILimiter) *Service {
	return &Service{
		logger:        logger.With(l, "service", ServiceName),
		config:        cfg,
		name:          ServiceName,
		parserService: parserService,
		lim:           lim,
	}
}

func (s Service) Name() string { return s.name }

func (s *Service) Start(ctx context.Context) error {
	bot, err := telebot.NewBot(telebot.Settings{
		Token:  s.config.TelegramBotApiToken,
		Poller: &telebot.LongPoller{Timeout: 10 * time.Second},
		//Verbose: s.config.EnvCI == "local",
	})
	if err != nil {
		return fmt.Errorf("failed to create telegram bot instance: %w", err)
	}

	s.bot = bot

	// init handler
	handler := newHandler(s.logger, s.config, s.parserService, s.lim, s.bot)

	// set command for bot
	s.bot.Handle("/start", handler.Start)
	s.bot.Handle(telebot.OnText, handler.OnText)
	s.bot.Handle(telebot.OnQuery, handler.OnQuery)

	// start bot instance
	go func() {
		s.bot.Start()
	}()

	<-ctx.Done()

	return nil
}

func (s Service) Stop(ctx context.Context) error {
	s.bot.Stop()
	return nil
}

var _ service.IService = (*Service)(nil)
