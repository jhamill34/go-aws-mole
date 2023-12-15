package bastion

import (
	"context"
	"errors"
	"log"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	ec2Types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/aws/aws-sdk-go-v2/service/ec2instanceconnect"
	"github.com/jhamill34/go-mole/pkg/tunnels"
)

const Ec2User = "ec2-user"

var ErrNoInstances = errors.New("No bastion instances found")

type EC2BastionService struct {
	ec2Client       *ec2.Client
	instanceConnect *ec2instanceconnect.Client
	filters         []ec2Types.Filter
}

func NewEC2BastionService(
	ec2Client *ec2.Client,
	instanceConnect *ec2instanceconnect.Client,
	filters []ec2Types.Filter,
) *EC2BastionService {
	return &EC2BastionService{
		ec2Client:       ec2Client,
		instanceConnect: instanceConnect,
		filters:         filters,
	}
}

// GetBastion implements services.BastionService.
func (self *EC2BastionService) GetBastion(ctx context.Context) (*tunnels.BastionEntity, error) {
	instances, err := self.ec2Client.DescribeInstances(ctx, &ec2.DescribeInstancesInput{
		Filters: self.filters,
	})

	if err != nil {
		return nil, err
	}

	if len(instances.Reservations) == 0 || len(instances.Reservations[0].Instances) == 0 {
		return nil, ErrNoInstances
	}

	instance := instances.Reservations[0].Instances[0]

	return &tunnels.BastionEntity{
		Id:   *instance.InstanceId,
		IP:   *instance.PublicIpAddress,
		User: Ec2User,
	}, nil
}

// PushKey implements tunnels.BastionService.
func (self *EC2BastionService) PushKey(
	ctx context.Context,
	bastion *tunnels.BastionEntity,
	pub string,
) error {
	log.Printf("Pushing key to bastion %s", bastion.Id)
	_, err := self.instanceConnect.SendSSHPublicKey(ctx, &ec2instanceconnect.SendSSHPublicKeyInput{
		InstanceId:     &bastion.Id,
		InstanceOSUser: aws.String(Ec2User),
		SSHPublicKey:   &pub,
	})
	if err != nil {
		return err
	}

	return nil
}
