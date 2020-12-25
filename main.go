package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
	bolt "go.etcd.io/bbolt"
)

const (
	fsClientTimeout = 2 * time.Second

	boltTimeout = time.Second
)

func main() {
	publicURL := os.Getenv("PUBLIC_URL")
	if publicURL == "" {
		log.Fatal("PUBLIC_URL env is missing")
	}

	apiKey := os.Getenv("BOT_API_KEY")
	if apiKey == "" {
		log.Fatal("BOT_API_KEY env is missing")
	}

	port := os.Getenv("PORT")
	if port == "" {
		log.Fatal("PORT env is missing")
	}

	bot, err := tgbotapi.NewBotAPI(apiKey)
	if err != nil {
		log.Panic(fmt.Errorf("failed to init bot api: %w", err))
	}

	bot.Debug = true

	db, err := bolt.Open(DatabaseFile, 0o600, &bolt.Options{
		Timeout: boltTimeout,
	})
	if err != nil {
		log.Fatal(fmt.Errorf("failed to open database: %w", err))
	}
	defer func() {
		if err := db.Close(); err != nil {
			log.Println(fmt.Errorf("failed to close db: %w", err))
		}
	}()

	log.Printf("bot authorized on account %s", bot.Self.UserName)

	u := tgbotapi.NewUpdate(0)

	u.Timeout = 60

	webhookURL := publicURL + bot.Token
	if _, err := bot.SetWebhook(tgbotapi.NewWebhook(webhookURL)); err != nil {
		log.Fatal(fmt.Errorf("failed to set webhook to %s: %w", webhookURL, err))
	}

	info, err := bot.GetWebhookInfo()
	if err != nil {
		log.Fatal(fmt.Errorf("failed to get webhook info: %w", err))
	}

	if info.LastErrorDate != 0 {
		log.Printf("telegram callback failed: %s", info.LastErrorMessage)
	}

	updates := bot.ListenForWebhook("/" + bot.Token)

	go func() {
		addr := ":" + port

		log.Printf("start listening on %s", addr)

		http.DefaultServeMux.Handle("/check", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			log.Println("check requested")

			w.WriteHeader(http.StatusOK)
		}))

		if err := http.ListenAndServe(addr, nil); err != nil {
			log.Println(fmt.Errorf("failed to listen and serve: %w", err))
		}
	}()

	bolts, err := NewBolt(db)
	if err != nil {
		log.Fatal(fmt.Errorf("failed to init bolt: %w", err))
	}

	fs := NewFinalSurgeAPI(&http.Client{
		Timeout: fsClientTimeout,
	})

	b := NewBot(bot, bolts, fs)

	for update := range updates {
		if err := b.update(update); err != nil {
			log.Println(err)
		}
	}
}
