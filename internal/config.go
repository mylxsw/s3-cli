package internal

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

type Config struct {
	DefaultServer string              `yaml:"default_server"`
	Servers       map[string]S3Server `yaml:"servers"`
}

type S3Server struct {
	Endpoint        string `yaml:"endpoint"`
	Region          string `yaml:"region"`
	Bucket          string `yaml:"bucket"`
	AccessKeyID     string `yaml:"access_key_id"`
	SecretAccessKey string `yaml:"secret_access_key"`
	SessionToken    string `yaml:"session_token,omitempty"`
	ForcePathStyle  bool   `yaml:"force_path_style,omitempty"`
	CDNBaseURL      string `yaml:"cdn_base_url"`
	BaseDir         string `yaml:"base_dir"`
	PathFormat      string `yaml:"path_format"`
	ACL             string `yaml:"acl,omitempty"`
}

func DefaultConfigPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".s3-cli", "config.yaml"), nil
}

func LoadConfig(path string) (Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Config{}, err
	}
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return Config{}, err
	}
	if len(cfg.Servers) == 0 {
		return Config{}, errors.New("no servers configured")
	}
	return cfg, nil
}

func (c Config) SelectServer(name string) (string, S3Server, error) {
	if name != "" {
		server, ok := c.Servers[name]
		if !ok {
			return "", S3Server{}, fmt.Errorf("server %q not found", name)
		}
		return name, server, nil
	}
	if c.DefaultServer != "" {
		server, ok := c.Servers[c.DefaultServer]
		if !ok {
			return "", S3Server{}, fmt.Errorf("default server %q not found", c.DefaultServer)
		}
		return c.DefaultServer, server, nil
	}
	if len(c.Servers) == 1 {
		for key, server := range c.Servers {
			return key, server, nil
		}
	}
	return "", S3Server{}, errors.New("multiple servers configured, please choose one with -server")
}

func (s S3Server) Validate() error {
	if s.Region == "" {
		return errors.New("region is required")
	}
	if s.Bucket == "" {
		return errors.New("bucket is required")
	}
	if s.AccessKeyID == "" || s.SecretAccessKey == "" {
		return errors.New("access_key_id and secret_access_key are required")
	}
	if s.CDNBaseURL == "" {
		return errors.New("cdn_base_url is required")
	}
	return nil
}
