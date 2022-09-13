package internal

import (
	"context"
	"fmt"
	"sort"

	"github.com/AlecAivazis/survey/v2"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	ec2_types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
	"github.com/fatih/color"
)

var (
	defaultAwsRegions = []string{
		"af-south-1",
		"ap-east-1", "ap-northeast-1", "ap-northeast-2", "ap-northeast-3", "ap-south-1", "ap-southeast-1", "ap-southeast-2", "ap-southeast-3",
		"ca-central-1",
		"cn-north-1", "cn-northwest-1",
		"eu-central-1", "eu-north-1", "eu-south-1", "eu-west-1", "eu-west-2", "eu-west-3",
		"me-south-1", "me-central-1",
		"sa-east-1",
		"us-east-1", "us-east-2", "us-west-1", "us-west-2",
	}

	instanceList = make(map[string]*EC2)
)

type (
	Region struct {
		Name string
	}
)

func FindTagInstanceAllRegion(ctx context.Context, cfg aws.Config) (map[string]*EC2, error) {

	client := ec2.NewFromConfig(cfg)

	for _, region := range defaultAwsRegions {
		defaultInstanceTagName = fmt.Sprintf("govpn-ec2-%s", region)

		output, err := client.DescribeInstances(ctx,
			&ec2.DescribeInstancesInput{
				Filters: []ec2_types.Filter{
					{Name: aws.String("instance-state-name"), Values: []string{"running"}},
					{Name: aws.String("tag:Name"), Values: []string{defaultInstanceTagName}},
				},
			},
		)
		if err != nil {
			return instanceList, err
		}

		for _, reservations := range output.Reservations {
			for _, ec2 := range reservations.Instances {
				instanceList[region] = &EC2{
					Existence:    true,
					Id:           aws.ToString(ec2.InstanceId),
					PublicIP:     aws.ToString(ec2.PublicIpAddress),
					LaunchTime:   aws.ToTime(ec2.LaunchTime),
					InstanceType: aws.ToString((*string)(&ec2.InstanceType)),
				}
			}
		}
	}

	for region, instance := range instanceList {
		fmt.Printf("Instance List:\n[%s]\n%s\n%s\n%s\n%s\n", region, color.GreenString(instance.Id), instance.InstanceType, instance.PublicIP, instance.LaunchTime)
	}

	return instanceList, nil
}

func AskRegion(ctx context.Context, cfg aws.Config) (*Region, error) {
	var regions []string
	client := ec2.NewFromConfig(cfg)

	output, err := client.DescribeRegions(ctx, &ec2.DescribeRegionsInput{
		AllRegions: aws.Bool(true),
	})
	if err != nil {
		regions = make([]string, len(defaultAwsRegions))
		copy(regions, defaultAwsRegions)
	} else {
		regions = make([]string, 0, len(output.Regions))
		for _, region := range output.Regions {
			regions = append(regions, aws.ToString(region.RegionName))
		}
	}
	sort.Strings(regions)

	var region string
	prompt := &survey.Select{
		Message: "Choose a region in AWS:",
		Options: regions,
	}

	if err := survey.AskOne(prompt, &region, survey.WithIcons(func(icons *survey.IconSet) {
		icons.SelectFocus.Format = "green+hb"
	}), survey.WithPageSize(20)); err != nil {
		return nil, err
	}

	return &Region{Name: region}, nil
}

func CreateStartSession(ctx context.Context, cfg aws.Config, input *ssm.StartSessionInput) (*ssm.StartSessionOutput, error) {
	client := ssm.NewFromConfig(cfg)

	return client.StartSession(ctx, input)
}

func DeleteStartSession(ctx context.Context, cfg aws.Config, input *ssm.TerminateSessionInput) error {
	client := ssm.NewFromConfig(cfg)
	fmt.Printf("%s %s \n", color.YellowString("Delete Session"),
		color.YellowString(aws.ToString(input.SessionId)))

	_, err := client.TerminateSession(ctx, input)
	return err
}

func PrintReady(cmd, region, title, content string) {
	fmt.Printf("%s region: %s, %s: %s\n", color.GreenString(cmd), color.HiYellowString(region), title, color.HiGreenString(content))
}
