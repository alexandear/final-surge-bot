package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

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
	config, err := InitConfig()
	if err != nil {
		return fmt.Errorf("failed to init config: %w", err)
	}

	dbPool, err := pgxpool.Connect(context.Background(), config.DatabaseURL)
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

	go serve(config.Debug, ":"+config.Port)

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

func updates(bot *tgbotapi.BotAPI, config Config) (tgbotapi.UpdatesChannel, error) {
	if config.Debug {
		log.Printf("bot authorized on account %s", bot.Self.UserName)
	}

	if config.RunOnHeroku {
		updates, err := updatesHeroku(bot, config)
		if err != nil {
			return nil, fmt.Errorf("failed to get updates on heroku: %w", err)
		}

		return updates, nil
	}

	updates, err := updatesLocal(bot)
	if err != nil {
		return nil, fmt.Errorf("failed to get updates local: %w", err)
	}

	return updates, nil
}

func updatesHeroku(bot *tgbotapi.BotAPI, config Config) (updates tgbotapi.UpdatesChannel, err error) {
	webhookURL := config.PublicURL + bot.Token
	if _, err = bot.SetWebhook(tgbotapi.NewWebhook(webhookURL)); err != nil {
		return nil, fmt.Errorf("failed to set webhook to %s: %w", webhookURL, err)
	}

	info, err := bot.GetWebhookInfo()
	if err != nil {
		return nil, fmt.Errorf("failed to get webhook info: %w", err)
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
		return nil, fmt.Errorf("failed to get updates chan: %w", err)
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
