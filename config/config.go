package config

import (
	"fmt"
	"log"
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Address            string            `yaml:"address"`
	Port               string            `yaml:"port"`
	SupportedLanguages []string          `yaml:"supportedLanguages"`
	RabbitMQ           struct {
		URL   string `yaml:"url"`
		Queue string `yaml:"queue"`
	} `yaml:"rabbitmq"`
	MaxWorkers     int               `yaml:"maxWorkers"`
	ApiTimeout     int               `yaml:"apiTimeout"`
	JobCPU         string            `yaml:"jobCpu"`
	JobMemory      string            `yaml:"jobMemory"`
	JobTimeout     int               `yaml:"jobTimeout"`
	OutputMaxBytes int               `yaml:"outputMaxBytes"`
	SandboxImages  map[string]string `yaml:"sandboxImages"`
}

func (c *Config) applyDefaults() {
	if c.Address == "" {
		c.Address = "127.0.0.1"
	}
	if c.Port == "" {
		c.Port = "8080"
	}
	if c.RabbitMQ.URL == "" {
		c.RabbitMQ.URL = "amqp://guest:guest@localhost:5672/"
	}
	if c.RabbitMQ.Queue == "" {
		c.RabbitMQ.Queue = "runner_jobs"
	}
	if c.MaxWorkers <= 0 {
		c.MaxWorkers = 2
	}
	if c.ApiTimeout <= 0 {
		c.ApiTimeout = 15
	}
	if c.JobCPU == "" {
		c.JobCPU = "1"
	}
	if c.JobMemory == "" {
		c.JobMemory = "1024m"
	}
	if c.JobTimeout <= 0 {
		c.JobTimeout = 10
	}
	if c.OutputMaxBytes <= 0 {
		c.OutputMaxBytes = 1048576 // 1MB
	}
	if len(c.SupportedLanguages) == 0 {
		c.SupportedLanguages = []string{"python", "go"}
	}
	if c.SandboxImages == nil {
		c.SandboxImages = map[string]string{
			"python": "python-runner",
			"go":     "go-runner",
		}
	}
}

func (c *Config) Validate() error {
	if c.MaxWorkers <= 0 {
		return fmt.Errorf("maxWorkers must be > 0, got %d", c.MaxWorkers)
	}
	if c.JobTimeout <= 0 {
		return fmt.Errorf("jobTimeout must be > 0, got %d", c.JobTimeout)
	}
	if c.ApiTimeout <= c.JobTimeout {
		log.Printf("WARNING: apiTimeout (%d) should be greater than jobTimeout (%d)", c.ApiTimeout, c.JobTimeout)
	}
	if len(c.SupportedLanguages) == 0 {
		return fmt.Errorf("supportedLanguages must not be empty")
	}
	return nil
}

func Load(path string) *Config {
	data, err := os.ReadFile(path)
	if err != nil {
		log.Fatalf("Cannot find the configuration in %s!", path)
	}

	var conf Config
	err = yaml.Unmarshal(data, &conf)
	if err != nil {
		log.Fatalf("Cannot read the configuration!")
	}

	conf.applyDefaults()

	if err := conf.Validate(); err != nil {
		log.Fatalf("Invalid configuration: %v", err)
	}

	return &conf
}
