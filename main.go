package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
	bolt "go.etcd.io/bbolt"
)

const (
	fsClientTimeout = 2 * time.Second

	databaseFile       = "final-surge-bot.db"
	databaseModeCreate = 0o600

	boltTimeout = time.Second
)

func main() {
	config, err := InitConfig()
	if err != nil {
		log.Panic(fmt.Errorf("failed to init config: %w", err))
	}

	db, err := bolt.Open(databaseFile, databaseModeCreate, &bolt.Options{
		Timeout: boltTimeout,
	})
	if err != nil {
		log.Panic(fmt.Errorf("failed to open database: %w", err))
	}

	defer func() {
		if cerr := db.Close(); cerr != nil {
			log.Println(fmt.Errorf("failed to close db: %w", cerr))
		}
	}()

	bot, updates, err := initBotAPI(config.BotAPIKey, config.PublicURL)
	if err != nil {
		log.Panic("failed to init bot api: %w", err)
	}

	go listen(":" + config.Port)

	bolts, err := NewBolt(db)
	if err != nil {
		log.Panic(fmt.Errorf("failed to init bolt: %w", err))
	}

	fs := NewFinalSurgeAPI(&http.Client{
		Timeout: fsClientTimeout,
	})

	b := NewBot(bot, bolts, fs)

	for update := range updates {
		if err := b.Process(context.Background(), update); err != nil {
			log.Println(err)
		}
	}
}

func initBotAPI(apiKey, publicURL string) (bot *tgbotapi.BotAPI, updates tgbotapi.UpdatesChannel, err error) {
	bot, err = tgbotapi.NewBotAPI(apiKey)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to init bot api: %w", err)
	}

	bot.Debug = true

	log.Printf("bot authorized on account %s", bot.Self.UserName)

	webhookURL := publicURL + bot.Token
	if _, werr := bot.SetWebhook(tgbotapi.NewWebhook(webhookURL)); werr != nil {
		return nil, nil, fmt.Errorf("failed to set webhook to %s: %w", webhookURL, werr)
	}

	info, err := bot.GetWebhookInfo()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get webhook info: %w", err)
	}

	if info.LastErrorDate != 0 {
		log.Println(fmt.Errorf("telegram callback failed: %s", info.LastErrorMessage))
	}

	updates = bot.ListenForWebhook("/" + bot.Token)

	return bot, updates, nil
}

func listen(addr string) {
	log.Printf("start listening on %s", addr)

	http.DefaultServeMux.Handle("/check", checkHandler())

	if lerr := http.ListenAndServe(addr, nil); lerr != nil {
		log.Println(fmt.Errorf("failed to listen and serve: %w", lerr))
	}
}

func checkHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log.Println("check requested")

		w.WriteHeader(http.StatusOK)
	}
}
