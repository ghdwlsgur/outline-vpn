package internal

import (
	"context"
	"fmt"
	"sort"

	"github.com/AlecAivazis/survey/v2"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	ec2_types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/fatih/color"
)

type (
	Region struct {
		Name string
	}
)

func AskRegion(ctx context.Context, cfg aws.Config) (*Region, error) {
	var regions []string
	client := ec2.NewFromConfig(cfg)

	// legacy [all region]
	// output, err := client.DescribeRegions(ctx, &ec2.DescribeRegionsInput{
	// 	AllRegions: aws.Bool(true),
	// })

	// sts: available
	output, err := client.DescribeRegions(ctx,
		&ec2.DescribeRegionsInput{
			Filters: []ec2_types.Filter{
				{Name: aws.String("endpoint"), Values: []string{"*"}},
			},
		})

	if err != nil {
		return nil, err
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

func CreateTags(ctx context.Context, cfg aws.Config, id *string, tagName string) error {
	client := ec2.NewFromConfig(cfg)

	_, err := client.CreateTags(ctx,
		&ec2.CreateTagsInput{
			Resources: []string{aws.ToString(id)},
			Tags: []ec2_types.Tag{
				{Key: aws.String("Name"), Value: aws.String(tagName)},
			},
		},
	)
	if err != nil {
		return err
	}

	return nil
}

func PrintReady(cmd, region, title, content string) {
	fmt.Printf("%s %s %s [%s: %s]\n",
		color.GreenString(cmd),
		color.HiBlackString("region:"),
		color.HiBlackString(region),
		color.HiBlackString(title),
		color.HiGreenString(content))
}

func PrintProvisioning(cmd, title, content string) {
	fmt.Printf("%s %s %s\n",
		color.HiMagentaString(cmd),
		color.HiBlackString(title),
		color.HiGreenString(content))
}

func AskPrompt(Message, AnswerOne, AnswerTwo string) (string, error) {

	prompt := &survey.Select{
		Message: Message,
		Options: []string{AnswerOne, AnswerTwo},
	}

	answer := ""
	if err := survey.AskOne(prompt, &answer, survey.WithIcons(func(icons *survey.IconSet) {
		icons.SelectFocus.Format = "green+hb"
	}), survey.WithPageSize(2)); err != nil {
		return "No", err
	}

	return answer, nil
}
