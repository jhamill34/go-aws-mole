package credentials

import (
	"context"
	"crypto/rand"
	"encoding/base32"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sts"
)

type RoleCredentialProvider struct {
	stsClient     *sts.Client
	awsConfig     AwsConfig
	sessionPrefix string
}

func NewRoleCredentialProvider(
	stsClient *sts.Client,
	awsConfig AwsConfig,
	sessionPrefix string,
) *RoleCredentialProvider {
	return &RoleCredentialProvider{
		stsClient:     stsClient,
		awsConfig:     awsConfig,
		sessionPrefix: sessionPrefix,
	}
}

// Retrieve implements aws.CredentialsProvider.
func (self *RoleCredentialProvider) Retrieve(ctx context.Context) (aws.Credentials, error) {
	roleArn := fmt.Sprintf(
		"arn:aws:iam::%s:role/%s",
		self.awsConfig.AccountID,
		self.awsConfig.RoleName,
	)

	sessionName := self.sessionPrefix + "-" + randomString(10)

	response, err := self.stsClient.AssumeRole(ctx, &sts.AssumeRoleInput{
		RoleArn:         aws.String(roleArn),
		RoleSessionName: aws.String(sessionName),
	})
	if err != nil {
		return aws.Credentials{}, err
	}

	return aws.Credentials{
		AccessKeyID:     *response.Credentials.AccessKeyId,
		SecretAccessKey: *response.Credentials.SecretAccessKey,
		SessionToken:    *response.Credentials.SessionToken,
		Source:          "RoleCredentialProvider",
	}, nil
}

func randomString(n int) string {
	bytes := make([]byte, n)
	_, err := rand.Read(bytes)
	if err != nil {
		panic(err)
	}

	return base32.StdEncoding.EncodeToString(bytes)
}

