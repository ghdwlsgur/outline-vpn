package internal

import (
	"context"

	"github.com/AlecAivazis/survey/v2"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	ec2_types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/fatih/color"
)

var (
	defaultSubnetTagName = "govpn-subnet"
)

type (
	DefaultSubnet struct {
		New       bool
		Existence bool
		Id        string
	}
)

func ExistsTagSubnet(ctx context.Context, cfg aws.Config) (*DefaultSubnet, error) {
	client := ec2.NewFromConfig(cfg)

	output, err := client.DescribeSubnets(ctx,
		&ec2.DescribeSubnetsInput{
			Filters: []ec2_types.Filter{
				{Name: aws.String("tag:Name"), Values: []string{defaultSubnetTagName}},
			},
		},
	)
	if err != nil {
		return nil, err
	}

	if len(output.Subnets) > 0 {
		return &DefaultSubnet{Id: aws.ToString(output.Subnets[0].SubnetId), Existence: true}, nil
	}

	return &DefaultSubnet{Existence: false}, nil
}

// aws ec2 create-default-subnet --availability-zone ap-northeast-2a

func CreateDefaultSubnet(ctx context.Context, cfg aws.Config, az string) (*DefaultSubnet, error) {

	client := ec2.NewFromConfig(cfg)

	output, err := client.CreateDefaultSubnet(ctx,
		&ec2.CreateDefaultSubnetInput{
			AvailabilityZone: aws.String(az),
		},
	)
	if err != nil {
		return &DefaultSubnet{}, err
	}

	_, err = client.CreateTags(ctx,
		&ec2.CreateTagsInput{
			Resources: []string{aws.ToString(output.Subnet.SubnetId)},
			Tags: []ec2_types.Tag{
				{Key: aws.String("Name"), Value: aws.String(defaultSubnetTagName)},
			},
		},
	)
	if err != nil {
		return &DefaultSubnet{New: true}, err
	}

	return &DefaultSubnet{New: true, Id: aws.ToString(output.Subnet.SubnetId)}, nil
}

func DeleteTagSubnet(ctx context.Context, cfg aws.Config, subnetId string) (bool, error) {
	client := ec2.NewFromConfig(cfg)

	if _, err := client.DeleteSubnet(ctx,
		&ec2.DeleteSubnetInput{SubnetId: aws.String(subnetId)}); err != nil {
		return false, err
	}

	return true, nil
}

func ExistsDefaultSubnet(ctx context.Context, cfg aws.Config, az string) (*DefaultSubnet, error) {

	client := ec2.NewFromConfig(cfg)

	/* AWS CLI Command Reference (https://docs.aws.amazon.com/cli/latest/reference/ec2/describe-subnets.html)
	* Example========================================================
	aws ec2 describe-subnets \
		--filter Name=default-for-az,Values=true Name=availability-zone,Values=ap-northeast-2a
	=================================================================
	*/

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
		return &DefaultSubnet{}, err
	}

	if len(output.Subnets) > 0 {
		return &DefaultSubnet{Existence: true}, nil
	}

	return &DefaultSubnet{Existence: false}, nil
}

func AskCreateDefaultSubnet() (string, error) {
	notice := color.New(color.Bold, color.FgHiRed).PrintFunc()
	notice("⚠️   Sorry, you cannot proceed without a default Subnet.\n")

	prompt := &survey.Select{
		Message: "Do You Create Default Subnet (tag: govpn-subnet):",
		Options: []string{"Yes", "No (exit)"},
	}

	answer := ""
	if err := survey.AskOne(prompt, &answer, survey.WithIcons(func(icons *survey.IconSet) {
		icons.SelectFocus.Format = "green+hb"
	}), survey.WithPageSize(2)); err != nil {
		return "No", err
	}

	return answer, nil
}

func AskDeleteTagSubnet() (string, error) {
	prompt := &survey.Select{
		Message: "Do You Delete Default SUBNET (tag: govpn-subnet):",
		Options: []string{"Yes", "No"},
	}

	answer := ""
	if err := survey.AskOne(prompt, &answer, survey.WithIcons(func(icons *survey.IconSet) {
		icons.SelectFocus.Format = "green+hb"
	}), survey.WithPageSize(2)); err != nil {
		return "No", err
	}
	return answer, nil
}
