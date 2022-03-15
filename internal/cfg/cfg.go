// Package cfg provides parsing system variables and command line flags for fetch service parameters.
// Command line flags replaces system variables if set
package cfg

import (
	"flag"
	"fmt"
	"github.com/antonevtu/go_shortener_adv/internal/pool"
	"github.com/caarlos0/env/v6"
	"strconv"
)

type Config struct {
	ServerAddress   string `env:"SERVER_ADDRESS" envDefault:":8080"`
	BaseURL         string `env:"BASE_URL" envDefault:"http://localhost:8080"`
	FileStoragePath string `env:"FILE_STORAGE_PATH" envDefault:"./storage.txt"`
	DatabaseDSN     string `env:"DATABASE_DSN"`
	CtxTimeout      int64  `env:"CTX_TIMEOUT" envDefault:"500"`
	DeleterChan     chan pool.ToDeleteItem
}

func New() (Config, error) {
	var cfg Config

	// Заполнение cfg значениями из переменных окружения, в том числе дефолтными значениями
	err := env.Parse(&cfg)
	if err != nil {
		return cfg, err
	}

	// Если заданы аргументы командной строки - перетираем значения переменных окружения
	flag.Func("a", "server address for shorten", func(flagValue string) error {
		cfg.ServerAddress = flagValue
		return nil
	})
	flag.Func("b", "base url for expand", func(flagValue string) error {
		cfg.BaseURL = flagValue
		return nil
	})
	flag.Func("f", "path to storage file", func(flagValue string) error {
		cfg.FileStoragePath = flagValue
		return nil
	})
	flag.Func("d", "postgres url", func(flagValue string) error {
		cfg.DatabaseDSN = flagValue
		return nil
	})
	flag.Func("t", "context timeout", func(flagValue string) error {
		t, err := strconv.Atoi(flagValue)
		if err != nil {
			return fmt.Errorf("can't parse context timeout -t: %w", err)
		}
		cfg.CtxTimeout = int64(t)
		return nil
	})

	flag.Parse()

	return cfg, err
}
