package internal

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	ec2_types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/fatih/color"
)

const (
	defaultVpcTagName = "govpn-vpc"
)

type (
	DefaultVpc struct {
		New       bool
		Existence bool
		Id        string
	}
)

func CreateDefaultVpc(ctx context.Context, cfg aws.Config) (*DefaultVpc, error) {

	client := ec2.NewFromConfig(cfg)

	/* AWS CLI Command Reference (https://docs.aws.amazon.com/cli/latest/reference/ec2/create-default-vpc.html)
	* Example========================================================
	aws ec2 create-default-vpc --region us-east-1
	=================================================================*/
	output, err := client.CreateDefaultVpc(ctx, &ec2.CreateDefaultVpcInput{})
	if err != nil {
		return &DefaultVpc{}, err
	} else {
		err := CreateTags(ctx, cfg, output.Vpc.VpcId, defaultVpcTagName)
		if err != nil {
			return &DefaultVpc{New: true}, err
		}
	}

	return &DefaultVpc{New: true, Id: aws.ToString(output.Vpc.VpcId)}, nil
}

func ExistsDefaultVpc(ctx context.Context, cfg aws.Config) (*DefaultVpc, error) {

	client := ec2.NewFromConfig(cfg)

	/* AWS CLI Command Reference (https://docs.aws.amazon.com/cli/latest/reference/ec2/describe-vpcs.html)
	* Example========================================================
	aws ec2 describe-vpcs \
		--filters Name=is-default,Values=true Name=state,Values=available \
		--region us-east-1
	=================================================================*/
	output, err := client.DescribeVpcs(ctx,
		&ec2.DescribeVpcsInput{
			Filters: []ec2_types.Filter{
				{Name: aws.String("is-default"), Values: []string{"true"}},
				{Name: aws.String("state"), Values: []string{"available"}},
			},
		},
	)
	if err != nil {
		return &DefaultVpc{Existence: false, New: false}, fmt.Errorf(`
⚠️  [privacy] Direct permission modification is required.
1. Aws Console -> IAM -> Account Settings
2. Click Activate for the region where you want to create the default VPC.
		`)
	}

	if len(output.Vpcs) > 0 {
		return &DefaultVpc{Existence: true, New: false}, nil
	}

	return &DefaultVpc{Existence: false, New: false}, nil
}

func ExistsTagVpc(ctx context.Context, cfg aws.Config) (*DefaultVpc, error) {
	client := ec2.NewFromConfig(cfg)

	output, err := client.DescribeVpcs(ctx,
		&ec2.DescribeVpcsInput{
			Filters: []ec2_types.Filter{
				{Name: aws.String("tag:Name"), Values: []string{defaultVpcTagName}},
			},
		},
	)
	if err != nil {
		return nil, err
	}

	if len(output.Vpcs) > 0 {
		return &DefaultVpc{Id: aws.ToString(output.Vpcs[0].VpcId), Existence: true}, nil
	}

	return &DefaultVpc{Existence: false}, nil
}

func DeleteTagVpc(ctx context.Context, cfg aws.Config, vpcId string) (bool, error) {
	client := ec2.NewFromConfig(cfg)

	if _, err := DeleteIgws(ctx, cfg, vpcId); err != nil {
		return false, err
	}

	if _, err := DeleteSubnets(ctx, cfg, vpcId); err != nil {
		return false, err
	}

	if _, err := client.DeleteVpc(ctx,
		&ec2.DeleteVpcInput{VpcId: aws.String(vpcId)}); err != nil {
		return false, err
	}

	return true, nil
}

func DeleteSubnets(ctx context.Context, cfg aws.Config, vpcId string) (bool, error) {
	var subnetIds []string

	client := ec2.NewFromConfig(cfg)

	output, err := client.DescribeSubnets(ctx,
		&ec2.DescribeSubnetsInput{
			Filters: []ec2_types.Filter{
				{Name: aws.String("vpc-id"), Values: []string{vpcId}},
			},
		},
	)
	if err != nil {
		return false, err
	}

	subnetIds = make([]string, 0, len(output.Subnets))
	for _, subnet := range output.Subnets {
		subnetIds = append(subnetIds, aws.ToString(subnet.SubnetId))
	}

	for _, id := range subnetIds {
		if _, err := client.DeleteSubnet(ctx,
			&ec2.DeleteSubnetInput{
				SubnetId: aws.String(id),
			},
		); err != nil {
			return false, err
		}
	}

	return true, nil
}

func DeleteIgws(ctx context.Context, cfg aws.Config, vpcId string) (bool, error) {
	var igwIds []string

	client := ec2.NewFromConfig(cfg)

	output, err := client.DescribeInternetGateways(ctx,
		&ec2.DescribeInternetGatewaysInput{
			Filters: []ec2_types.Filter{
				{Name: aws.String("attachment.vpc-id"), Values: []string{vpcId}},
			},
		},
	)
	if err != nil {
		return false, err
	}

	igwIds = make([]string, 0, len(output.InternetGateways))
	for _, igw := range output.InternetGateways {
		igwIds = append(igwIds, aws.ToString(igw.InternetGatewayId))
	}

	for _, id := range igwIds {
		if _, detachErr := client.DetachInternetGateway(ctx,
			&ec2.DetachInternetGatewayInput{
				InternetGatewayId: aws.String(id),
				VpcId:             aws.String(vpcId),
			},
		); detachErr != nil {
			return false, detachErr
		}

		if _, deleteErr := client.DeleteInternetGateway(ctx,
			&ec2.DeleteInternetGatewayInput{
				InternetGatewayId: aws.String(id),
			},
		); deleteErr != nil {
			return false, deleteErr
		}
	}

	return true, nil
}

func AskCreateDefaultVpc() (string, error) {
	notice := color.New(color.Bold, color.FgHiRed).PrintFunc()
	notice("⚠️   Sorry, you cannot proceed without a default VPC.\n")

	return AskPrompt("Do You Create Default VPC (tag: govpn-vpc):", "Yes", "No (exit)")
}

func AskDeleteTagVpc() (string, error) {
	return AskPrompt("Do You Delete Default VPC (tag: govpn-vpc):", "Yes", "No")
}
