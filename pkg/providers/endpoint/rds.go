package endpoint

import (
	"context"
	"errors"

	"github.com/aws/aws-sdk-go-v2/service/rds"
	rdsTypes "github.com/aws/aws-sdk-go-v2/service/rds/types"
	"github.com/jhamill34/go-mole/pkg/tunnels"
)

var ErrNoDBEndpointFound = errors.New("no db endpoint found")

type RdsDatabaseInfoService struct {
	rdsClient *rds.Client
	filters  []rdsTypes.Filter
}

func NewRdsDatabaseInfoService(rdsClient *rds.Client, filters []rdsTypes.Filter) *RdsDatabaseInfoService {
	return &RdsDatabaseInfoService{
		rdsClient: rdsClient,
		filters:  filters,
	}
}

// GetEndpoint implements services.EndpointProvider.
func (self *RdsDatabaseInfoService) GetEndpoint(
	ctx context.Context,
) (*tunnels.EndpointInfo, error) {
	response, err := self.rdsClient.DescribeDBClusterEndpoints(
		ctx,
		&rds.DescribeDBClusterEndpointsInput{
			Filters: self.filters,
		},
	)

	if err != nil {
		return nil, err
	}

	if len(response.DBClusterEndpoints) == 0 {
		return nil, ErrNoDBEndpointFound
	}

	endpoint := response.DBClusterEndpoints[0]

	return &tunnels.EndpointInfo{
		Protocol: "tcp",
		Host:     *endpoint.Endpoint,
		Port:     3306,
	}, nil
}

