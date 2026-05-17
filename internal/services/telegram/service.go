package telegram

import (
	"context"
	"fmt"
	"time"

	"github.com/sxwebdev/downloaderbot/internal/config"
	"github.com/sxwebdev/downloaderbot/internal/limiter"
	"github.com/sxwebdev/downloaderbot/internal/services/parser"
	"github.com/tkcrm/mx/logger"
	"gopkg.in/telebot.v3"
)

const ServiceName = "telegram-service"

type Service struct {
	logger logger.Logger
	config *config.Config
	name   string

	lim           *limiter.Limiter
	parserService *parser.Service

	bot  *telebot.Bot
	done chan struct{}
}

func New(l logger.Logger, cfg *config.Config, parserService *parser.Service, lim *limiter.Limiter) *Service {
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
		// Verbose: s.config.EnvCI == "local",
	})
	if err != nil {
		return fmt.Errorf("failed to create telegram bot instance: %w", err)
	}

	s.bot = bot

	// init handler
	handler := newHandler(s.logger, s.config, s.parserService, s.lim, s.bot)

	// set command for bot
	s.bot.Handle("/start", handler.recover("start", handler.Start))
	s.bot.Handle(telebot.OnText, handler.recover("on_text", handler.OnText))
	s.bot.Handle(telebot.OnQuery, handler.recover("on_query", handler.OnQuery))

	// start bot instance
	s.done = make(chan struct{})
	go func() {
		defer close(s.done)
		s.bot.Start()
	}()

	<-ctx.Done()

	return nil
}

func (s *Service) Stop(ctx context.Context) error {
	if s.bot == nil {
		return nil
	}
	s.bot.Stop()
	if s.done != nil {
		select {
		case <-s.done:
		case <-ctx.Done():
		}
	}
	return nil
}
