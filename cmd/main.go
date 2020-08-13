package main

import (
	"context"
	"flag"
	"gitlab.com/faemproject/backend/faem/pkg/jobqueue"
	"gitlab.com/faemproject/backend/faem/services/msgbot/models"
	"log"
	"strings"
	"time"

	"github.com/go-pg/pg"
	"github.com/labstack/echo/v4"
	"github.com/sirupsen/logrus"
	"gitlab.com/faemproject/backend/faem/pkg/prometheus"

	"gitlab.com/faemproject/backend/faem/pkg/logs"
	"gitlab.com/faemproject/backend/faem/pkg/os"
	"gitlab.com/faemproject/backend/faem/pkg/rabbit"
	"gitlab.com/faemproject/backend/faem/pkg/store"
	"gitlab.com/faemproject/backend/faem/pkg/web"
	"gitlab.com/faemproject/backend/faem/pkg/web/middleware"
	"gitlab.com/faemproject/backend/faem/services/msgbot/broker/publisher"
	"gitlab.com/faemproject/backend/faem/services/msgbot/broker/subscriber"
	"gitlab.com/faemproject/backend/faem/services/msgbot/config"
	"gitlab.com/faemproject/backend/faem/services/msgbot/dialogflow"
	"gitlab.com/faemproject/backend/faem/services/msgbot/handler"
	"gitlab.com/faemproject/backend/faem/services/msgbot/repository"
	"gitlab.com/faemproject/backend/faem/services/msgbot/server"
	"gitlab.com/faemproject/backend/faem/services/msgbot/telegrambot"
)

const (
	defaultConfigPath     = "config/msgbot.toml"
	maxRequestsAllowed    = 1000
	serverShutdownTimeout = 30 * time.Second
	brokerShutdownTimeout = 30 * time.Second
)

func main() {
	// Parse flags
	configPath := flag.String("config", defaultConfigPath, "configuration file path")
	flag.Parse()

	cfg, err := config.Parse(*configPath)
	if err != nil {
		log.Fatalf("failed to parse the config file: %v", err)
	}

	cfg.Print() // just for debugging

	if err := logs.SetLogLevel(cfg.Application.LogLevel); err != nil {
		log.Fatalf("Failed to set log level: %v", err)
	}
	if err := logs.SetLogFormat(cfg.Application.LogFormat); err != nil {
		log.Fatalf("Failed to set log format: %v", err)
	}
	logger := logs.Eloger

	// Connect to the db and remember to close it
	db, err := store.Connect(&pg.Options{
		Addr:     store.Addr(cfg.Database.Host, cfg.Database.Port),
		User:     cfg.Database.User,
		Password: cfg.Database.Password,
		Database: cfg.Database.Db,
	})
	if err != nil {
		logger.Fatalf("failed to create a db instance: %v", err)
	}
	defer db.Close()

	// Connect to the broker and remember to close it
	rmq := &rabbit.Rabbit{
		Credits: rabbit.ConnCredits{
			URL:  cfg.Broker.UserURL,
			User: cfg.Broker.UserCredits,
		},
	}
	if err = rmq.Init(cfg.Broker.ExchangePrefix, cfg.Broker.ExchangePostfix); err != nil {
		logger.Fatalf("failed to connect to RabbitMQ: %v", err)
	}
	defer rmq.CloseRabbit()

	// Create a publisher
	pub := publisher.Publisher{
		Rabbit:  rmq,
		Encoder: &rabbit.JsonEncoder{},
	}
	if err = pub.Init(); err != nil {
		logger.Fatalf("failed to init the publisher: %v", err)
	}
	defer pub.Wait(brokerShutdownTimeout)

	//Dialog Flow Agent
	dfAgent, err := dialogflow.InitAgent(dialogflow.DFConfig{
		ProjectID:    cfg.DialogFlow.ProjectID,
		JSONFilePath: cfg.DialogFlow.JSONFilePath,
		Lang:         cfg.DialogFlow.Lang,
		Timezone:     cfg.DialogFlow.Timezone,
	})
	if err != nil {
		logger.Fatalf("failed to init dialog flow: %v", err)
	}

	tlgBot := telegrambot.BotClient{
		Token: cfg.Application.TelegramToken,
	}
	if err = tlgBot.Init(); err != nil {
		logger.Fatalf("failed to init the telegram: %v", err)
	}

	// Throwing chatapi data to handler
	cfg.Settings.ChatApi = cfg.ChatApi

	// Create a service object
	hdlr := handler.Handler{
		DB:       &repository.Pg{Db: db},
		Pub:      &pub,
		Telegram: &tlgBot,
		DF:       &dfAgent,
		Config:   cfg.Settings,
		Buffers: handler.Buffers{
			WIPOrders:     make(map[string]string),
			CRMOrders:     make(map[string]string),
			DriverFounded: make(map[string]string),
			WIPOrdersFull: make(map[string]models.LocalOrders),
		},
		Jobs: jobqueue.NewJobQueues(),
	}

	if err = hdlr.InitBuffer(context.Background()); err != nil {
		logger.Fatalf("error initing buffer data: %v", err)
	}

	//Telegram Subscriber
	tlgSub := telegrambot.Subscriber{
		Bot:     tlgBot.Bot,
		Handler: &hdlr,
	}
	if err = tlgSub.Init(); err != nil {
		logger.Fatalf("failed to handling telegram messages: %v", err)
	}

	// Create a subscriber
	sub := subscriber.Subscriber{
		Rabbit:  rmq,
		Encoder: &rabbit.JsonEncoder{},
		Handler: &hdlr,
	}
	if err = sub.Init(); err != nil {
		logger.Fatalf("failed to start the subscriber: %v", err)
	}
	defer sub.Wait(brokerShutdownTimeout)

	// Create a rest gateway and handle http requests
	router := web.NewRouter(
		loggerOption(logger),
		prometheusmetric,
		throttler,
	)
	rest := server.Rest{
		Router:  router,
		Handler: &hdlr,
	}
	rest.Route()

	// Start an http server and remember to shut it down
	go web.Start(router, cfg.Application.Port)
	defer web.Stop(router, serverShutdownTimeout)

	// Wait for program exit
	<-os.NotifyAboutExit()
}

func loggerOption(logger *logrus.Logger) web.Option {
	return func(e *echo.Echo) {
		e.Logger = &middleware.Logger{Logger: logger} // replace the original echo.Logger with the logrus one
		// Log the requests
		e.Use(middleware.LoggerWithSkipper(
			func(c echo.Context) bool {
				return strings.Contains(c.Request().RequestURI, "/api/v2/locations")
			}),
		)
	}
}

func throttler(e *echo.Echo) {
	e.Use(middleware.Throttle(maxRequestsAllowed))
}

func prometheusmetric(e *echo.Echo) {
	p := prometheus.NewPrometheus("echo", nil)
	p.Use(e)
}
