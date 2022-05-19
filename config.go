package main

import (
	"os"
	"path"

	"github.com/XMonetae-DeFi/apollo/db"
	"github.com/XMonetae-DeFi/apollo/types"

	"gopkg.in/yaml.v2"
)

type Config struct {
	Rpc        map[types.Chain]string `yaml:"rpc"`
	DbSettings db.DbSettings          `yaml:"postgres"`
}

func NewConfig(path string) (*Config, error) {
	var c Config

	f, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	if err = yaml.Unmarshal(f, &c); err != nil {
		return nil, err
	}

	return &c, nil
}

func ConfigPath() (string, error) {
	confDir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}

	return path.Join(confDir, "apollo", "config.yml"), nil
}

func ConfigDir() (string, error) {
	confDir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}

	return path.Join(confDir, "apollo"), nil
}

func SchemaPath() (string, error) {
	confDir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}

	return path.Join(confDir, "apollo", "schema.yml"), nil
}
