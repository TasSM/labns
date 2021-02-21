package config

import (
	"encoding/json"
	"os"
)

const (
	config_path = "/etc/labdns/config.json"
)

type Configuration struct {
	Foo string
	Bar []string `json:"foo_bar"`
}

func LoadConfig() *Configuration {
	file, err := os.Open(config_path)

	if err != nil {
		panic(err)
	}

	defer file.Close()

	config := &Configuration{}
	decoder := json.NewDecoder(file)
	err = decoder.Decode(config)

	if err != nil {
		panic(err)
	}

	return config
}
