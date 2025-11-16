package config

import (
	"os"
	"gopkg.in/yaml.v3"
	"log"
)

type Config struct {
	Address string `yaml:"address"`
	Port string `yaml:"port"`
	SupportedLanguages []string `yaml:"supportedLanguages"`
	RabbitMQ struct {
		URL string `yaml:"url"`
		Queue string `yaml:"queue"`
	} `yaml:"rabbitmq"`
	MaxJobs int `yaml:"maxJobs"`
}

func Load(path string) (*Config) {
	config, err := os.ReadFile(path)
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
