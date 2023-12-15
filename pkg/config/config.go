package config

import (
	"os"

	"gopkg.in/yaml.v3"
)

type AwsConfig struct {
	RoleName    string `yaml:"role_name"`
	AccountID   string `yaml:"account_id"`
	SessionName string `yaml:"session_name"`
	Region      string `yaml:"region"`
}

type KeyValue struct {
	Key   string   `yaml:"key"`
	Value []string `yaml:"value"`
}

type TunnelConfig struct {
	Host string `yaml:"host"`
	Port int    `yaml:"port"`
}

type RdsConfig struct {
	Tunnel  TunnelConfig `yaml:"tunnel"`
	Filters []KeyValue   `yaml:"filters"`
}

type ElasticSearchConfig struct {
	Tunnel     TunnelConfig `yaml:"tunnel"`
	DomainName string       `yaml:"domain_name"`
}

type Ec2Config struct {
	Filters []KeyValue `yaml:"filters"`
}

type RootTunnelConfig struct {
	Aws           AwsConfig            `yaml:"aws"`
	Ec2           Ec2Config            `yaml:"ec2"`
	Rds           *RdsConfig           `yaml:"rds"`
	ElasticSearch *ElasticSearchConfig `yaml:"elasticsearch"`
}

func LoadTunnelConfig(path string) (*RootTunnelConfig, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var cfg RootTunnelConfig
	yaml.NewDecoder(file).Decode(&cfg)

	return &cfg, nil
}
