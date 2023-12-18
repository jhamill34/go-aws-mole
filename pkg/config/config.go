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

type Reference struct {
	Ref string `yaml:"$ref"`
}

type KeyValue struct {
	Key   string   `yaml:"key"`
	Value []string `yaml:"value"`
}

type BastionConfig struct {
	Host Reference `yaml:"host"`
	Key  Reference `yaml:"key"`
}

type SshTunnelConfig struct {
	Destination Reference `yaml:"destination"`
	Port        int       `yaml:"port"`
}

type SocksConfig struct {
	Allowlist []Reference `yaml:"allowlist"`
	Port      int         `yaml:"port"`
}

type ElasticSearchConfig struct {
	DomainName string `yaml:"domain_name"`
}

type RdsConfig struct {
	Filters []KeyValue `yaml:"filters"`
}

type Ec2Config struct {
	Filters []KeyValue `yaml:"filters"`
}

type OnetimeConfig struct{}

type ProvidersConfig struct {
	ElasticSearch map[string]ElasticSearchConfig `yaml:"elasticsearch"`
	Rds           map[string]RdsConfig           `yaml:"rds"`
	Ec2           map[string]Ec2Config           `yaml:"ec2"`
	Onetime       map[string]OnetimeConfig       `yaml:"onetime"`
}

type RootTunnelConfig struct {
	Aws        AwsConfig         `yaml:"aws"`
	Bastion    BastionConfig     `yaml:"bastion"`
	SshTunnel  []SshTunnelConfig `yaml:"ssh_tunnel"`
	SocksProxy SocksConfig       `yaml:"socks_proxy"`
	Providers  ProvidersConfig   `yaml:"providers"`
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
