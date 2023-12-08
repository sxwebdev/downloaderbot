package proxy

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/proxy"
	recoverMiddleware "github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/sxwebdev/downloaderbot/internal/config"
	"github.com/tkcrm/mx/logger"
	"github.com/tkcrm/mx/service"
)

const serviceName = "proxy-service"

type Service struct {
	logger logger.Logger
	config *config.Config
	name   string

	fiber *fiber.App
}

func New(l logger.ExtendedLogger, cfg *config.Config) *Service {
	app := fiber.New(fiber.Config{
		DisableStartupMessage: true,
	})

	// add recover
	app.Use(recoverMiddleware.New(recoverMiddleware.Config{
		EnableStackTrace: true,
	}))

	// add cors config
	app.Use(cors.New(cors.Config{
		AllowOrigins: "*",
		AllowMethods: strings.Join([]string{
			fiber.MethodGet,
			fiber.MethodHead,
		}, ","),
	}))

	app.Get("/download/:filename", func(c *fiber.Ctx) error {
		redirectUrlEscaped := c.Query("redirectUrl")
		if redirectUrlEscaped == "" {
			return fmt.Errorf("empty redirectUrl")
		}

		redirectUrl, err := url.QueryUnescape(redirectUrlEscaped)
		if err != nil {
			return fmt.Errorf("unascape query error: %w", err)
		}

		l.Infof("received redirect url: %s", redirectUrl)

		if err := proxy.Do(c, redirectUrl); err != nil {
			return err
		}

		c.Response().Header.Set("Content-type", "application/octet-stream")

		return nil
	})

	return &Service{
		logger: logger.With(l, "service", serviceName),
		config: cfg,
		name:   serviceName,
		fiber:  app,
	}
}

func (s Service) Name() string { return s.name }

func (s *Service) Start(ctx context.Context) error {
	errChan := make(chan error, 1)

	// start http server
	go func() {
		errChan <- s.fiber.Listen(":9020")
	}()

	select {
	case err := <-errChan:
		return err
	case <-ctx.Done():
	}

	return nil
}

func (s Service) Stop(ctx context.Context) error {
	return s.fiber.ShutdownWithContext(ctx)
}

var _ service.IService = (*Service)(nil)
