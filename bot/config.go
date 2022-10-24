package bot

import (
	"fmt"

	"github.com/kelseyhightower/envconfig"
)

type Config struct {
	Debug       bool   `envconfig:"DEBUG"`
	PublicURL   string `envconfig:"PUBLIC_URL" required:"true"`
	BotAPIKey   string `envconfig:"BOT_API_KEY" required:"true"`
	Port        int    `envconfig:"PORT" required:"true"`
	DatabaseURL string `envconfig:"DATABASE_URL" required:"true"`
	RunOnHeroku bool   `envconfig:"RUN_ON_HEROKU"`
}

func NewConfig() (*Config, error) {
	c := &Config{}
	if err := envconfig.Process("", c); err != nil {
		return nil, fmt.Errorf("process config: %w", err)
	}

	return c, nil
}
