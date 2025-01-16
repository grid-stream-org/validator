package config

import (
	"path/filepath"

	"github.com/grid-stream-org/batcher/pkg/logger"
	"github.com/knadh/koanf/parsers/json"
	"github.com/knadh/koanf/providers/file"
	"github.com/knadh/koanf/v2"
	"github.com/pkg/errors"
)

type Config struct {
	Log *logger.Config `koanf:"log"`
	Server struct {
		Address string `koanf:"address"` // Add server address configuration
	} `koanf:"server"`
}

func Load() (*Config, error){
	k:= koanf.New(".")
	path := filepath.Join("configs", "config.json")
	if err := k.Load(file.Provider(path), json.Parser()); err != nil{
		return nil, errors.WithStack(err)
	}

	var cfg Config
	if err := k.Unmarshal("", &cfg); err != nil{
		return nil, errors.WithStack(err)
	}

	return &cfg, nil
}