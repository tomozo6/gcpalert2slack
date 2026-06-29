package config

import (
	"os"
)

type Config struct {
	Port           string
	SlackBotToken  string
	SlackChannelID string
}

// Load は環境変数から Config を組み立てる。
func Load() Config {
	return Config{
		Port:           valueOrDefault("PORT", "8080"),
		SlackBotToken:  os.Getenv("SLACK_BOT_TOKEN"),
		SlackChannelID: os.Getenv("SLACK_CHANNEL_ID"),
	}
}

// valueOrDefault は環境変数に値があればそれを返し、
// 空なら defaultValue を返す。
func valueOrDefault(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}

	return value
}
