package internal

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/AlecAivazis/survey/v2"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	ec2_types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
)

var (
	defaultAmis = []string{
		"ami-00f881f027a6d74a0", // ca-central-1
		"ami-02d1e544b84bf7502", // us-east-2
		"ami-0cff7528ff583bf9a", // us-east-1
		"ami-0d9858aa3c6322f73", // us-west-1
		"ami-098e42ae54c764c35", // us-west-2
		"ami-08df646e18b182346", // ap-south-1
		"ami-0c66c8e259df7ec04", // ap-northeast-3
		"ami-0fd0765afb77bcca7", // ap-northeast-2
		"ami-0b7546e839d7ace12", // ap-northeast-1
		"ami-0c802847a7dd848c0", // ap-southeast-1
		"ami-07620139298af599e", // ap-southeast-2
		"ami-037c192f0fa52a358", // sa-east-1
	}
)

type (
	Ami struct {
		Name          string
		ImageLocation string
	}
)

func AskAmi(ctx context.Context, cfg aws.Config) (*Ami, error) {
	var amis []string
	var (
		client     = ec2.NewFromConfig(cfg)
		table      = make(map[string]*Ami)
		outputFunc = func(table map[string]*Ami, output *ec2.DescribeImagesOutput) {
			for _, ami := range output.Images {

				if strings.Contains(*ami.ImageLocation, "amzn-ami-hvm") {
					table[fmt.Sprintf("%s\t(%s)", *ami.ImageId, *ami.ImageLocation)] = &Ami{
						Name:          aws.ToString(ami.ImageId),
						ImageLocation: aws.ToString(ami.ImageLocation),
					}
				}
			}
		}
	)

	output, err := client.DescribeImages(ctx,
		&ec2.DescribeImagesInput{
			Owners: []string{"amazon"},
			Filters: []ec2_types.Filter{
				{Name: aws.String("state"), Values: []string{"available"}},
				{Name: aws.String("architecture"), Values: []string{"x86_64"}},
				{Name: aws.String("root-device-type"), Values: []string{"ebs"}},
				{Name: aws.String("is-public"), Values: []string{"true"}},
			},
		},
	)

	if err != nil {
		amis = make([]string, len(defaultAmis))
		copy(amis, defaultAmis)
	} else {
		amis = make([]string, 0, len(table))
		outputFunc(table, output)
	}

	for idwithLocation := range table {
		amis = append(amis, idwithLocation)
	}
	sort.Sort(sort.Reverse(sort.StringSlice(amis)))

	if len(amis) == 0 {
		return nil, fmt.Errorf("not found Amazon Machine Image")
	}

	prompt := &survey.Select{
		Message: "Choose a Amazon Machine Image in AWS:",
		Options: amis,
	}

	selectKey := ""
	if err := survey.AskOne(prompt, &selectKey, survey.WithIcons(func(icons *survey.IconSet) {
		icons.SelectFocus.Format = "green+hb"
	}), survey.WithPageSize(20)); err != nil {
		return nil, err
	}
	return table[selectKey], nil
}
