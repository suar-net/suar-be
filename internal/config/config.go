package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

type Config struct {
	Server ServerConfig
	DB     DBConfig
}

type ServerConfig struct {
	Port         string
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
	IdleTimeout  time.Duration
}

type DBConfig struct {
	Host    string
	Port    int
	User    string
	Pass    string
	Name    string
	SSLMode string
	DSN     string
}

func LoadConfig() (*Config, error) {
	dbPort, err := strconv.Atoi(os.Getenv("DB_PORT"))
	if err != nil {
		return nil, fmt.Errorf("invalid DB_PORT: %v", err)
	}

	dBConfig := DBConfig{
		Host:    os.Getenv("DB_HOST"),
		Port:    dbPort,
		User:    os.Getenv("DB_USER"),
		Pass:    os.Getenv("DB_PASS"),
		Name:    os.Getenv("DB_NAME"),
		SSLMode: os.Getenv("DB_SSLMODE"),
	}
	dBConfig.DSN = fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		dBConfig.Host, dBConfig.Port, dBConfig.User, dBConfig.Pass, dBConfig.Name, dBConfig.SSLMode,
	)

	serverConfig := ServerConfig{
		Port:         os.Getenv("SERVER_PORT"),
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	return &Config{
		Server: serverConfig,
		DB:     dBConfig,
	}, nil

}
