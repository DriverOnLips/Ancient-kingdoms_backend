package config

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/golang-jwt/jwt"
	"github.com/joho/godotenv"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

type Config struct {
	ServiceHost string
	ServicePort int

	JWT   JWTConfig
	Redis RedisConfig
}

type RedisConfig struct {
	Host        string
	Password    string
	Port        int
	User        string
	DialTimeout time.Duration
	ReadTimeout time.Duration
}

type JWTConfig struct {
	ExpiresIn     time.Duration
	SigningMethod jwt.SigningMethod
	Token         string
}

const (
	envRedisHost = "REDIS_HOST"
	envRedisPort = "REDIS_PORT"
	envRedisUser = "REDIS_USER"
	envRedisPass = "REDIS_PASSWORD"
)

func NewConfig(ctx context.Context) (*Config, error) {
	var err error

	configName := "config.toml"
	_ = godotenv.Load()

	if os.Getenv("CONFIG_NAME") != "" {
		configName = os.Getenv("CONFIG_NAME")
	}

	viper.SetConfigName(configName)
	viper.SetConfigType("toml")
	viper.AddConfigPath("../../internal/config")
	viper.WatchConfig()

	err = viper.ReadInConfig()
	if err != nil {
		return nil, err
	}

	cfg := &Config{}
	err = viper.Unmarshal(cfg)
	if err != nil {
		return nil, err
	}

	cfg.Redis.Host = os.Getenv("REDIS_HOST")

	cfg.Redis.Port, err = strconv.Atoi(os.Getenv(envRedisPort))
	if err != nil {
		return nil, fmt.Errorf("redis port must be int value: %w", err)
	}

	cfg.Redis.Password = os.Getenv(envRedisPass)
	cfg.Redis.User = os.Getenv(envRedisUser)

	cfg.JWT.ExpiresIn = time.Duration(int64(^uint64(0) >> 1))
	cfg.JWT.SigningMethod = jwt.SigningMethodHS256
	cfg.JWT.Token = "test"

	log.Info("config parsed")

	return cfg, nil

}
