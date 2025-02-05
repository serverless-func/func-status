package config

import (
	"fmt"
	"github.com/apolloconfig/agollo/v4"
	"github.com/apolloconfig/agollo/v4/constant"
	apollo "github.com/apolloconfig/agollo/v4/env/config"
	"github.com/apolloconfig/agollo/v4/extension"
	"github.com/serverless-aliyun/func-status/client/core"
	"gopkg.in/yaml.v3"
	"log"
	"os"
)

// Config is the main configuration structure
type Config struct {
	// Debug Whether to enable debug logs
	Debug bool `yaml:"debug,omitempty"`

	// MaxDays of results to keep
	MaxDays int `yaml:"maxDays,omitempty"`

	// Database DSN
	DSN string `yaml:"dsn,omitempty"`

	// Endpoints List of endpoints to monitor
	Endpoints []*core.Endpoint `yaml:"endpoints,omitempty"`
}

func LoadConfiguration(cfgPath string) (*Config, error) {
	// Read the file
	configBytes, err := os.ReadFile(cfgPath)
	if err != nil {
		log.Printf("Error reading configuration from %s: %s", cfgPath, err)
		return nil, fmt.Errorf("error reading configuration from file %s: %w", cfgPath, err)
	}

	var config *Config
	err = yaml.Unmarshal(configBytes, &config)
	if err != nil {
		log.Printf("Error parse configuration from %s: %s", cfgPath, err)
		return nil, fmt.Errorf("error parse configuration from file %s: %w", cfgPath, err)
	}
	if config.MaxDays == 0 {
		config.MaxDays = 30
	}
	return config, err
}

func LoadApolloConfiguration() (*Config, error) {

	c := &apollo.AppConfig{
		AppID:          os.Getenv("APOLLO_APP_ID"),
		Cluster:        "default",
		IP:             os.Getenv("APOLLO_HOST"),
		NamespaceName:  os.Getenv("APOLLO_NAMESPACE"),
		IsBackupConfig: true,
		Secret:         os.Getenv("APOLLO_TOKEN"),
	}
	extension.AddFormatParser(constant.YAML, &Parser{})
	client, _ := agollo.StartWithConfig(func() (*apollo.AppConfig, error) {
		return c, nil
	})

	remoteConfig := client.GetConfig(c.NamespaceName).GetContent()
	log.Printf("Success Load Remote Config: %s\n", remoteConfig)
	var config *Config
	err := yaml.Unmarshal([]byte(remoteConfig), &config)
	return config, err
}

// Parser properties转换器
type Parser struct {
}

// Parse 内存内容=>yml文件转换器
func (d *Parser) Parse(configContent interface{}) (map[string]interface{}, error) {
	m := make(map[string]interface{})
	m["content"] = configContent
	return m, nil
}
