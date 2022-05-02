package main

import (
	"os"

	"github.com/XMonetae-DeFi/apollo/db"
	"gopkg.in/yaml.v2"
)

type Config struct {
	Rpc        map[string]string `yaml:"rpc"`
	DbSettings db.DbSettings     `yaml:"postgres"`
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
