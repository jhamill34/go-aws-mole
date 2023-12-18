package credentials

import (
	"context"

	localConfig "github.com/jhamill34/go-aws-mole/pkg/config"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/sts"
)

func AssumeRole(cfg localConfig.AwsConfig) aws.Config {
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
