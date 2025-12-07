package config

import "os"

type Config struct {
	Port string
}

func Load() *Config {
	return &Config{
		Port: getEnv("PORT", "8080"),
	}
}

func getEnv(k, d string) string {
	if val, ok := os.LookupEnv(k); ok {
		return val
	}
	return d
}
