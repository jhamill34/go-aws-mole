package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2instanceconnect"
	"github.com/aws/aws-sdk-go-v2/service/rds"
	"github.com/aws/aws-sdk-go-v2/service/sts"

	ec2Types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
	rdsTypes "github.com/aws/aws-sdk-go-v2/service/rds/types"
	localConfig "github.com/jhamill34/go-aws-mole/pkg/config"
	"github.com/jhamill34/go-aws-mole/pkg/providers/bastion"
	credential_provider "github.com/jhamill34/go-aws-mole/pkg/providers/credentials"
	"github.com/jhamill34/go-aws-mole/pkg/providers/endpoint"
	"github.com/jhamill34/go-mole/pkg/providers"
	"github.com/jhamill34/go-mole/pkg/runner"
	"github.com/jhamill34/go-mole/pkg/tunnels"
)

func main() {
	ctx := context.Background()

	ch := make(chan struct{})

	go func() {
		sig := make(chan os.Signal, 1)
		signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM, syscall.SIGKILL)

		<-sig
		close(ch)
	}()

	ConfigureBackgroundTunnel(os.Args[1]).Start(ctx, ch)
}

type BackgroundTunnel struct {
	mole *runner.Mole
}

func ConfigureBackgroundTunnel(configPath string) *BackgroundTunnel {
	log.Println("Starting tunnel for stage")

	cfg, err := localConfig.LoadTunnelConfig(configPath)
	if err != nil {
		panic(err)
	}

	roleConfig := roleConfig(cfg.Aws)

	var tunnels []runner.Tunneler

	if cfg.ElasticSearch != nil {
		tunnels = append(
			tunnels,
			createSocksTunnel(roleConfig, cfg.ElasticSearch.Tunnel.Port, cfg.Ec2.Filters),
		)
	}

	if cfg.Rds != nil {
		tunnels = append(
			tunnels,
			createRdsSshTunnel(roleConfig, cfg.Rds.Tunnel.Port, cfg.Ec2.Filters, cfg.Rds.Filters),
		)
	}

	return &BackgroundTunnel{
		mole: runner.NewMole(
			tunnels...,
		),
	}
}

func (self *BackgroundTunnel) Start(ctx context.Context, sig chan struct{}) {
	self.mole.Start(ctx)

	sig <- struct{}{}

	<-sig
	log.Println("Received signal, shutting down")

	self.mole.Stop()
}

func roleConfig(cfg localConfig.AwsConfig) aws.Config {
	ctx := context.Background()

	defaultConfig, err := config.LoadDefaultConfig(ctx, config.WithRegion(cfg.Region))
	if err != nil {
		panic(err)
	}

	stsConfig := sts.NewFromConfig(defaultConfig)
	roleCredentialProvider := credential_provider.NewRoleCredentialProvider(
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

func createSocksTunnel(
	roleConfig aws.Config,
	localPort int,
	ec2Filter []localConfig.KeyValue,
) *tunnels.SocksTunnel {
	ec2Client := ec2.NewFromConfig(roleConfig)

	instanceConnect := ec2instanceconnect.NewFromConfig(roleConfig)
	bastionService := bastion.NewEC2BastionService(
		ec2Client,
		instanceConnect,
		createEc2Filters(ec2Filter),
	)

	return tunnels.NewSocksTunnel(
		bastionService,
		providers.NewStaticKeyProvider(),
		localPort,
	)
}

func createRdsSshTunnel(
	roleConfig aws.Config,
	localPort int,
	ec2Filter []localConfig.KeyValue,
	rdsFilter []localConfig.KeyValue,
) *tunnels.SshTunnel {
	ec2Client := ec2.NewFromConfig(roleConfig)

	instanceConnect := ec2instanceconnect.NewFromConfig(roleConfig)
	bastionService := bastion.NewEC2BastionService(
		ec2Client,
		instanceConnect,
		createEc2Filters(ec2Filter),
	)

	rdsClient := rds.NewFromConfig(roleConfig)

	databaseInfoService := endpoint.NewRdsDatabaseInfoService(
		rdsClient,
		createRdsFilters(rdsFilter),
	)
	return tunnels.NewSshTunnel(
		bastionService,
		providers.NewStaticKeyProvider(),
		databaseInfoService,
		localPort,
	)
}

func createRdsFilters(filters []localConfig.KeyValue) []rdsTypes.Filter {
	var rdsFilters []rdsTypes.Filter

	for _, filter := range filters {
		rdsFilters = append(rdsFilters, rdsTypes.Filter{
			Name:   aws.String(filter.Key),
			Values: filter.Value,
		})
	}

	return rdsFilters
}

func createEc2Filters(filters []localConfig.KeyValue) []ec2Types.Filter {
	var ec2Filters []ec2Types.Filter

	for _, filter := range filters {
		ec2Filters = append(ec2Filters, ec2Types.Filter{
			Name:   aws.String(filter.Key),
			Values: filter.Value,
		})
	}

	return ec2Filters
}
