package main

import (
	"errors"
	"os"
	"strings"
)

type Config struct {
	Debug bool

	PublicURL   string
	BotAPIKey   string
	Port        string
	DatabaseURL string
	RunOnHeroku bool
}

func InitConfig() (Config, error) {
	var debug bool
	if v := os.Getenv("DEBUG"); strings.EqualFold(v, "true") || v == "1" {
		debug = true
	}

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

	var runOnHeroku bool
	if v := os.Getenv("RUN_ON_HEROKU"); strings.EqualFold(v, "true") || v == "1" {
		runOnHeroku = true
	}

	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		return Config{}, errors.New("DATABASE_URL is missing")
	}

	return Config{
		Debug: debug,

		PublicURL:   publicURL,
		BotAPIKey:   apiKey,
		Port:        port,
		DatabaseURL: databaseURL,
		RunOnHeroku: runOnHeroku,
	}, nil
}
