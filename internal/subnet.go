package internal

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	ec2_types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/fatih/color"
)

var (
	// The tag name for the default subnet is govpn-subnet.
	// Track resources created by govpn applications with the aws tag feature.
	defaultSubnetTagName = "govpn-subnet"
)

type (
	// A struct with information about the default subnet.
	DefaultSubnet struct {
		New       bool   // Indicates whether to create a new subnet.
		Existence bool   // Indicates whether a default subnet exists.
		Id        string // Save the id value of the default subnet.
	}
)

// Create a default subnet.
func CreateDefaultSubnet(ctx context.Context, cfg aws.Config, az string) (*DefaultSubnet, error) {

	client := ec2.NewFromConfig(cfg)

	/* AWS CLI Command Reference (https://docs.aws.amazon.com/cli/latest/reference/ec2/create-default-subnet.html)
	* Example========================================================
	aws ec2 create-default-subnet --availability-zone ap-northeast-2a
	=================================================================*/
	output, err := client.CreateDefaultSubnet(ctx,
		&ec2.CreateDefaultSubnetInput{
			AvailabilityZone: aws.String(az),
		},
	)
	if err != nil {
		// When an error occurs, only an error is returned.
		return nil, err
	} else {
		// Tag govpn-subnet.
		err := CreateTags(ctx, cfg, output.Subnet.SubnetId, defaultSubnetTagName)
		if err != nil {
			return &DefaultSubnet{New: true}, err
		}
	}

	// We created a new subnet, so New: returns true.
	return &DefaultSubnet{New: true, Id: aws.ToString(output.Subnet.SubnetId)}, nil
}

// Make sure the default subnet exists.
func ExistsDefaultSubnet(ctx context.Context, cfg aws.Config, az string) (*DefaultSubnet, error) {

	client := ec2.NewFromConfig(cfg)

	/* AWS CLI Command Reference (https://docs.aws.amazon.com/cli/latest/reference/ec2/describe-subnets.html)
	* Example========================================================
	aws ec2 describe-subnets \
		--filter Name=default-for-az,Values=true Name=availability-zone,Values=ap-northeast-2a
	=================================================================*/
	output, err := client.DescribeSubnets(ctx,
		&ec2.DescribeSubnetsInput{
			Filters: []ec2_types.Filter{
				{Name: aws.String("default-for-az"), Values: []string{"true"}},
				{Name: aws.String("availability-zone"), Values: []string{az}},
				{Name: aws.String("state"), Values: []string{"available"}},
			},
		},
	)
	if err != nil {
		// When an error occurs, only an error is returned.
		return nil, err
	}

	// If the cli query result is 0 or more, the default subnet exists.
	if len(output.Subnets) > 0 {
		// Returns true because the default subnet exists.
		return &DefaultSubnet{Existence: true}, nil
	}

	// It returns false because the default subnet does not exist.
	return &DefaultSubnet{Existence: false}, nil
}

// Verify that tagged subnets exist.
func ExistsTagSubnet(ctx context.Context, cfg aws.Config) (*DefaultSubnet, error) {
	client := ec2.NewFromConfig(cfg)

	/* AWS CLI Command Reference (https://docs.aws.amazon.com/cli/latest/reference/ec2/describe-subnets.html)
	* Example========================================================
	aws ec2 describe-subnets \
		--filter Name=tag-key,Values=govpn-subnet
	=================================================================*/
	output, err := client.DescribeSubnets(ctx,
		&ec2.DescribeSubnetsInput{
			Filters: []ec2_types.Filter{
				{Name: aws.String("tag:Name"), Values: []string{defaultSubnetTagName}},
			},
		},
	)
	if err != nil {
		// When an error occurs, only an error is returned.
		return nil, err
	}

	// Since the tagged subnet exists, the subnet's ID and Existence: returns true.
	if len(output.Subnets) > 0 {
		return &DefaultSubnet{Id: aws.ToString(output.Subnets[0].SubnetId), Existence: true}, nil
	}

	// Because the tagged subnet does not exist, it returns Existence: false.
	return &DefaultSubnet{Existence: false}, nil
}

// Delete the tagged subnet.
func DeleteTagSubnet(ctx context.Context, cfg aws.Config, subnetId string) (bool, error) {
	client := ec2.NewFromConfig(cfg)

	/* AWS CLI Command Reference (https://docs.aws.amazon.com/cli/latest/reference/ec2/delete-subnet.html)
	* Example========================================================
	aws ec2 delete-subnet --subnet-id subnet-9d4a7b6c
	=================================================================*/
	_, err := client.DeleteSubnet(ctx, &ec2.DeleteSubnetInput{SubnetId: aws.String(subnetId)})
	if err != nil {
		// Returns false if an error occurs.
		return false, err
	}

	// Returns true if deletion is successful.
	return true, nil
}

// It receives input from the user whether or not to create a default subnet.
func AskCreateDefaultSubnet() (string, error) {
	notice := color.New(color.Bold, color.FgHiRed).PrintFunc()
	notice("⚠️   Sorry, you cannot proceed without a default Subnet.\n")

	return AskPrompt("Do You Create Default Subnet (tag: govpn-subnet):", "Yes", "No (exit)")
}

// Get a response from the user asking if they want to delete the govpn-subnet.
func AskDeleteTagSubnet() (string, error) {
	return AskPrompt("Do You Delete Default SUBNET (tag: govpn-subnet):", "Yes", "No")
}
