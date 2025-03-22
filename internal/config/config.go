package config

import (
	"os"
	"path/filepath"

	"github.com/grid-stream-org/api/pkg/firebase"
	"github.com/grid-stream-org/batcher/pkg/logger"
	"github.com/knadh/koanf/parsers/json"
	"github.com/knadh/koanf/providers/file"
	"github.com/knadh/koanf/v2"
	"github.com/pkg/errors"
)

type Config struct {
	Log       *logger.Config           `koanf:"log"`
	Server    *ServerConfig            `koanf:"server"`
	SendGrid  *SendGridConfig          `koanf:"sendgrid_api"`
	Firebase  *firebase.FirebaseConfig `koanf:"firebase"`
	WebAPIKey string                   `koanf:"web_api_key"`
}

type ServerConfig struct {
	Address string `koanf:"address"`
}

type SendGridConfig struct {
	Api    string `koanf:"key"`
	Sender string `koanf:"sender"`
}

func Load() (*Config, error) {
	k := koanf.New(".")

	path := os.Getenv("CONFIG_PATH")
	if path == "" {
		path = filepath.Join("configs", "config.json")
		logger.Default().Info("CONFIG_PATH not set, using default", "path", path)
	}
	if err := k.Load(file.Provider(path), json.Parser()); err != nil {
		return nil, errors.WithStack(err)
	}

	var cfg Config
	if err := k.Unmarshal("", &cfg); err != nil {
		return nil, errors.WithStack(err)
	}

	return &cfg, nil
}
