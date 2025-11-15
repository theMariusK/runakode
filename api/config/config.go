package config

import (
	"os"
	"gopkg.in/yaml.v3"
	"log"
)

type Config struct {
	SupportedLanguages []string `yaml:"supportedLanguages"`
}

func Load(path string) (*Config) {
	config, err := os.ReadFile(path)
	defer config.Close()
	if err != nil {
		log.Fatalf("Cannot find the configuration in %s!", path)
	}

	var conf Config
	err = yaml.Unmarshal(config, &conf)
	if err != nil {
		log.Fatalf("Cannot read the configuration!")
	}

	return &conf
}
