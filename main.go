package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"strconv"
	"time"

	"github.com/alexandear/final-surge-bot/bot"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
	"github.com/jackc/pgx/v4/pgxpool"
)

const (
	fsClientTimeout = 2 * time.Second

	serverReadTimeout  = 2 * time.Second
	serverWriteTimeout = 4 * time.Second
	serverIdleTimeout  = 120 * time.Second
)

func main() {
	if err := run(); err != nil {
		log.Panic(err)
	}
}

func run() error {
	config, err := bot.NewConfig()
	if err != nil {
		return fmt.Errorf("init config: %w", err)
	}

	dbPool, err := pgxpool.Connect(context.Background(), config.DatabaseURL)
	if err != nil {
		return fmt.Errorf("unable to connect to database %s: %w", config.DatabaseURL, err)
	}

	defer dbPool.Close()

	pg := bot.NewPostgres(dbPool)

	if errInit := pg.Init(context.Background()); errInit != nil {
		return fmt.Errorf("init postgres: %w", errInit)
	}

	tgbot, err := tgbotapi.NewBotAPI(config.BotAPIKey)
	if err != nil {
		return fmt.Errorf("init bot api: %w", err)
	}

	tgbot.Debug = config.Debug

	updates, err := updates(tgbot, config)
	if err != nil {
		return fmt.Errorf("init bot api: %w", err)
	}

	go func() {
		var host string
		if config.Debug {
			host = "localhost"
		}

		addr := net.JoinHostPort(host, strconv.Itoa(config.Port))
		serve(config.Debug, addr)
	}()

	fs := bot.NewFinalSurgeAPI(&http.Client{
		Timeout: fsClientTimeout,
	})

	clock := bot.NewClock()

	b := bot.NewBot(tgbot, pg, fs, clock)

	for update := range updates {
		if err := b.ProcessUpdate(context.Background(), update); err != nil {
			log.Printf("process update: %v", err)
		}
	}

	return nil
}

func updates(bot *tgbotapi.BotAPI, config *bot.Config) (tgbotapi.UpdatesChannel, error) {
	if config.Debug {
		log.Printf("bot authorized on account %s", bot.Self.UserName)
	}

	if config.RunOnCloud {
		updates, err := updatesCloud(bot, config.PublicURL)
		if err != nil {
			return nil, fmt.Errorf("get updates on cloud: %w", err)
		}

		return updates, nil
	}

	updates, err := updatesLocal(bot)
	if err != nil {
		return nil, fmt.Errorf("get updates local: %w", err)
	}

	return updates, nil
}

func updatesCloud(bot *tgbotapi.BotAPI, publicURL string) (updates tgbotapi.UpdatesChannel, err error) {
	webhookURL := publicURL + bot.Token
	if _, err = bot.SetWebhook(tgbotapi.NewWebhook(webhookURL)); err != nil {
		return nil, fmt.Errorf("set webhook to %s: %w", webhookURL, err)
	}

	info, err := bot.GetWebhookInfo()
	if err != nil {
		return nil, fmt.Errorf("get webhook info: %w", err)
	}

	if info.LastErrorDate != 0 {
		log.Printf("telegram callback failed: %s", info.LastErrorMessage)
	}

	updates = bot.ListenForWebhook("/" + bot.Token)

	return updates, nil
}

func updatesLocal(bot *tgbotapi.BotAPI) (updates tgbotapi.UpdatesChannel, err error) {
	const updateTimeout = 60 * time.Second

	u := tgbotapi.NewUpdate(0)
	u.Timeout = int(updateTimeout.Seconds())

	updates, err = bot.GetUpdatesChan(u)
	if err != nil {
		return nil, fmt.Errorf("get updates chan: %w", err)
	}

	return updates, nil
}

func serve(debug bool, addr string) {
	if debug {
		log.Printf("start listening on %s", addr)
	}

	mux := http.NewServeMux()
	mux.Handle("/", http.FileServer(http.Dir("./web")))
	mux.Handle("/check", checkHandler(debug))

	srv := &http.Server{
		Addr:         addr,
		Handler:      mux,
		ReadTimeout:  serverReadTimeout,
		WriteTimeout: serverWriteTimeout,
		IdleTimeout:  serverIdleTimeout,
	}
	if err := srv.ListenAndServe(); err != nil {
		log.Fatalf("start listen and serve: %v", err)
	}
}

func checkHandler(debug bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if debug {
			log.Println("check requested")
		}

		w.WriteHeader(http.StatusOK)
	}
}
