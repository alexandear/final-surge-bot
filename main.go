package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"strconv"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/jackc/pgx/v5/pgxpool"
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
	config, err := NewConfig()
	if err != nil {
		return fmt.Errorf("failed to init config: %w", err)
	}

	dbPool, err := pgxpool.New(context.Background(), config.DatabaseURL)
	if err != nil {
		return fmt.Errorf("unable to connect to database %s: %w", config.DatabaseURL, err)
	}

	defer dbPool.Close()

	pg := &Postgres{
		dbPool: dbPool,
	}

	if errInit := pg.Init(context.Background()); errInit != nil {
		return fmt.Errorf("failed to init postgres: %w", errInit)
	}

	bot, err := tgbotapi.NewBotAPI(config.BotAPIKey)
	if err != nil {
		return fmt.Errorf("failed to init bot api: %w", err)
	}

	bot.Debug = config.Debug

	updates, err := updates(bot, config)
	if err != nil {
		return fmt.Errorf("failed to init bot api: %w", err)
	}

	go func() {
		var host string
		if config.Debug {
			host = "localhost"
		}

		addr := net.JoinHostPort(host, strconv.Itoa(config.Port))
		serve(config.Debug, addr)
	}()

	fs := &FinalSurgeAPI{
		client: &http.Client{
			Timeout: fsClientTimeout,
		},
	}

	clock := &RealClock{}

	b := NewBot(bot, pg, fs, clock)

	for update := range updates {
		if err := b.ProcessUpdate(context.Background(), update); err != nil {
			log.Printf("failed to process update: %v", err)
		}
	}

	return nil
}

func updates(bot *tgbotapi.BotAPI, config *Config) (tgbotapi.UpdatesChannel, error) {
	if config.Debug {
		log.Printf("bot authorized on account %s", bot.Self.UserName)
	}

	if config.RunOnHeroku {
		updatesCh, err := updatesHeroku(bot, config.PublicURL)
		if err != nil {
			return nil, fmt.Errorf("failed to get updates on heroku: %w", err)
		}

		return updatesCh, nil
	}

	return updatesLocal(bot), nil
}

func updatesHeroku(bot *tgbotapi.BotAPI, publicURL string) (tgbotapi.UpdatesChannel, error) {
	webhookURL := publicURL + bot.Token
	webhook, err := tgbotapi.NewWebhook(webhookURL)
	if err != nil {
		return nil, fmt.Errorf("failed to create webhook to %s: %w", webhookURL, err)
	}

	if _, err = bot.Request(webhook); err != nil {
		return nil, fmt.Errorf("failed to request webhook: %w", err)
	}

	info, err := bot.GetWebhookInfo()
	if err != nil {
		return nil, fmt.Errorf("failed to get webhook info: %w", err)
	}

	if info.LastErrorDate != 0 {
		return nil, fmt.Errorf("telegram callback failed: %s", info.LastErrorMessage)
	}

	return bot.ListenForWebhook("/" + bot.Token), nil
}

func updatesLocal(bot *tgbotapi.BotAPI) tgbotapi.UpdatesChannel {
	const updateTimeout = 60 * time.Second

	u := tgbotapi.NewUpdate(0)
	u.Timeout = int(updateTimeout.Seconds())

	return bot.GetUpdatesChan(u)
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
		log.Fatalf("failed to start listen and serve: %v", err)
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
