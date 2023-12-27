package credentials

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/sts"
)

type AwsConfig struct {
	RoleName    string `yaml:"role_name"`
	AccountID   string `yaml:"account_id"`
	SessionName string `yaml:"session_name"`
	Region      string `yaml:"region"`
}

func AssumeRole(cfg AwsConfig) aws.Config {
	ctx := context.Background()

	defaultConfig, err := config.LoadDefaultConfig(ctx, config.WithRegion(cfg.Region))
	if err != nil {
		panic(err)
	}

	stsConfig := sts.NewFromConfig(defaultConfig)
	roleCredentialProvider := NewRoleCredentialProvider(
		stsConfig,
		cfg,
		"dashboard",
	)

	roleConfig, err := config.LoadDefaultConfig(
		ctx,
		config.WithRegion(cfg.Region),
		config.WithCredentialsProvider(roleCredentialProvider),
	)

	if err != nil {
		panic(err)
	}

	return roleConfig
}
