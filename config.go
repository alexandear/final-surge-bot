package main

import (
	"errors"
	"os"
)

type Config struct {
	PublicURL string
	BotAPIKey string
	Port      string
}

func InitConfig() (Config, error) {
	publicURL := os.Getenv("PUBLIC_URL")
	if publicURL == "" {
		return Config{}, errors.New("PUBLIC_URL env is missing")
	}

	apiKey := os.Getenv("BOT_API_KEY")
	if apiKey == "" {
		return Config{}, errors.New("BOT_API_KEY env is missing")
	}

	port := os.Getenv("PORT")
	if port == "" {
		return Config{}, errors.New("PORT env is missing")
	}

	return Config{
		PublicURL: publicURL,
		BotAPIKey: apiKey,
		Port:      port,
	}, nil
}
