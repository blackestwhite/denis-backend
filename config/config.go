package config

import "os"

var (
	KEY       string
	REDIS_URL string
)

func Load() {
	KEY = os.Getenv("OPEN_AI_KEY")
	REDIS_URL = os.Getenv("REDIS_URL")
}
