package endpoint 

import (
	"context"
	"errors"
	"log"

	"github.com/aws/aws-sdk-go-v2/service/elasticsearchservice"
	"github.com/jhamill34/go-mole/pkg/tunnels"
)

type ESVPCInfoService struct {
	esClient *elasticsearchservice.Client
	domainName string
}

func NewESInfoService(esClient *elasticsearchservice.Client, domainName string) *ESVPCInfoService {
	return &ESVPCInfoService{esClient, domainName}
}

// GetEndpoint implements services.EndpointProvider.
func (self *ESVPCInfoService) GetEndpoint(ctx context.Context) (*tunnels.EndpointInfo, error) {
	domains, err := self.esClient.ListDomainNames(ctx, &elasticsearchservice.ListDomainNamesInput{})
	if err != nil {
		return nil, err
	}

	var ksDomain string
	for i := 0; i < len(domains.DomainNames); i++ {
		domain := domains.DomainNames[i]
		if *domain.DomainName == self.domainName {
			ksDomain = *domain.DomainName
			break
		}
	}

	if ksDomain == "" {
		return nil, errors.New("Could not find ks domain")
	}

	endpoint, err := self.esClient.DescribeElasticsearchDomain(
		ctx,
		&elasticsearchservice.DescribeElasticsearchDomainInput{
			DomainName: &ksDomain,
		},
	)

	ksEndpoint, ok := endpoint.DomainStatus.Endpoints["vpc"]
	if !ok {
		return nil, errors.New("Could not find vpc endpoint")
	}

	log.Println("Endpoint: ", ksEndpoint)

	return &tunnels.EndpointInfo{
		Protocol: "https",
		Host:     ksEndpoint,
		Port:     443,
	}, nil
}

