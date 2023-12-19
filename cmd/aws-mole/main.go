package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"

	"github.com/aws/aws-sdk-go-v2/aws"

	"github.com/aws/aws-sdk-go-v2/service/ec2"
	ec2Types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/aws/aws-sdk-go-v2/service/ec2instanceconnect"
	"github.com/aws/aws-sdk-go-v2/service/elasticsearchservice"
	"github.com/aws/aws-sdk-go-v2/service/rds"
	rdsTypes "github.com/aws/aws-sdk-go-v2/service/rds/types"
	localConfig "github.com/jhamill34/go-aws-mole/pkg/config"
	"github.com/jhamill34/go-aws-mole/pkg/providers/bastion"
	"github.com/jhamill34/go-aws-mole/pkg/providers/credentials"
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

	var wg sync.WaitGroup
	wg.Add(1)

	go func() {
		defer wg.Done()
		ConfigureBackgroundTunnel(os.Args[1]).Start(ctx, ch)
	}()

	<-ch

	wg.Wait()
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

	roleConfig := credentials.AssumeRole(cfg.Aws)
	ec2Client := ec2.NewFromConfig(roleConfig)
	ec2InstanceConnect := ec2instanceconnect.NewFromConfig(roleConfig)
	rdsClient := rds.NewFromConfig(roleConfig)
	esClient := elasticsearchservice.NewFromConfig(roleConfig)

	var bastionService tunnels.BastionService
	var parts []string
	parts = strings.Split(cfg.Bastion.Host.Ref, "/")
	parts = parts[1:]
	if parts[0] != "providers" {
		panic("Invalid bastion host reference")
	}

	if parts[1] == "ec2" {
		bastionService = bastion.NewEC2BastionService(
			ec2Client,
			ec2InstanceConnect,
			createEc2Filters(cfg.Providers.Ec2[parts[2]].Filters),
		)
	} else {
		panic("Invalid bastion host reference")
	}

	var keyProvider tunnels.KeyProvider
	parts = strings.Split(cfg.Bastion.Key.Ref, "/")
	parts = parts[1:]
	if parts[0] != "providers" {
		panic("Invalid bastion key reference")
	}

	if parts[1] == "onetime" {
		keyProvider = providers.NewStaticKeyProvider()
	} else {
		panic("Invalid bastion key reference")
	}

	tunnelers := make([]runner.Tunneler, 0, len(cfg.SshTunnel))
	for _, tunnel := range cfg.SshTunnel {
		parts = strings.Split(tunnel.Destination.Ref, "/")
		parts = parts[1:]
		if parts[0] != "providers" {
			panic("Invalid ssh tunnel destination reference")
		}

		var endpointProvider tunnels.EndpointProvider
		if parts[1] == "rds" {
			endpointProvider = endpoint.NewRdsDatabaseInfoService(
				rdsClient,
				createRdsFilters(cfg.Providers.Rds[parts[2]].Filters),
			)
		} else if parts[1] == "elasticsearch" {
			endpointProvider = endpoint.NewESInfoService(
				esClient,
				cfg.Providers.ElasticSearch[parts[2]].DomainName,
			)
		} else {
			panic("Invalid ssh tunnel destination reference")
		}
		tunnelers = append(tunnelers, tunnels.NewSshTunnel(
			bastionService,
			keyProvider,
			endpointProvider,
			tunnel.Port,
		))
	}

	allowList := make([]tunnels.EndpointProvider, 0, len(cfg.SocksProxy.Allowlist))
	for _, allow := range cfg.SocksProxy.Allowlist {
		parts = strings.Split(allow.Ref, "/")
		parts = parts[1:]
		if parts[0] != "providers" {
			panic("Invalid socks proxy allowlist reference")
		}

		var endpointProvider tunnels.EndpointProvider
		if parts[1] == "rds" {
			endpointProvider = endpoint.NewRdsDatabaseInfoService(
				rdsClient,
				createRdsFilters(cfg.Providers.Rds[parts[2]].Filters),
			)
		} else if parts[1] == "elasticsearch" {
			endpointProvider = endpoint.NewESInfoService(
				esClient,
				cfg.Providers.ElasticSearch[parts[2]].DomainName,
			)
		} else {
			panic("Invalid socks proxy allowlist reference")
		}
		allowList = append(allowList, endpointProvider)
	}

	tunnelers = append(tunnelers, tunnels.NewSocksTunnel(
		bastionService,
		keyProvider,
		allowList,
		cfg.SocksProxy.Port,
	))

	return &BackgroundTunnel{
		mole: runner.NewMole(
			tunnelers...,
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
