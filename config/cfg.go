package config

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"github.com/ilyakaznacheev/cleanenv"
)

const cfgFilePath = ".env"

type (
	Config struct {
		App   app
		Redis redis
	}

	app struct {
		Name     string `env:"APP_NAME"    env-required:"true"`
		Version  string `env:"APP_VERSION" env-required:"true"`
		Env      string `env:"APP_ENV"     env-required:"true"`
		LogLevel string `env:"LOG_LEVEL"   env-required:"true"`
		Location string `env:"APP_LOCATION" env-default:"Asia/Ho_Chi_Minh"`
	}

	redis struct {
		Host           string `env:"REDIS_HOST"             env-required:"true"`
		Port           string `env:"REDIS_PORT"             env-required:"true"`
		Username       string `env:"REDIS_USERNAME"         env-required:"true"`
		Password       string `env:"REDIS_PASSWORD"         env-required:"true"`
		ClientName     string `env:"REDIS_CLIENT_NAME"      env-required:"true"`
		MaxRetries     int    `env:"REDIS_MAX_RETRIES"      env-default:"3"`
		PoolSize       int    `env:"REDIS_POOL_SIZE"        env-default:"10"`
		MaxIdleConns   int    `env:"REDIS_MAX_IDLE_CONNS"   env-default:"5"`
		MaxActiveConns int    `env:"REIDS_MAX_ACTIVE_CONNS" env-default:"10"`
		MaxIdleTime    int    `env:"REIDS_MAX_IDLE_TIME"    env-default:"30"`
		MaxLifeTime    int    `env:"REDIS_MAX_LIFE_TIME"    env-default:"10"`
	}
)

func NewConfig() *Config {
	cfg := &Config{}
	root := projectRoot()
	configFilePath := root + cfgFilePath

	err := loadCfg(configFilePath, cfg)
	if err != nil {
		panic(err)
	}

	return cfg
}

func loadCfg(cfgFilePath string, cfg *Config) error {
	envFileExists := checkFileExists(cfgFilePath)
	if envFileExists {
		err := cleanenv.ReadConfig(cfgFilePath, cfg)
		if err != nil {
			return fmt.Errorf("config error: %w", err)
		}
	} else {
		err := cleanenv.ReadEnv(cfg)
		if err != nil {
			if _, statErr := os.Stat(cfgFilePath); statErr != nil {
				return fmt.Errorf("missing environment variable: %w", err)
			}
			return err
		}
	}
	return nil
}

func checkFileExists(fileName string) bool {
	exist := false
	if _, err := os.Stat(fileName); err == nil {
		exist = true
	}
	return exist
}

func projectRoot() string {
	_, b, _, _ := runtime.Caller(0)
	cwd := filepath.Dir(b)
	return cwd + "/../"
}
