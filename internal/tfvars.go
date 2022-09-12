package internal

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
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

	defaultInstanceType = "t2.micro"
)

type (
	Ami struct {
		Name          string
		ImageLocation string
	}

	AvailabilityZone struct {
		Name string
	}

	InstanceType struct {
		Name string
	}
)

func SaveTerraformVariable(jsonData map[string]interface{}, jsonFilePath string) (string, error) {
	jsonBuf, _ := json.Marshal(jsonData)
	err := os.WriteFile(jsonFilePath, jsonBuf, os.FileMode(0644))
	if err != nil {
		return "failed to save file", err
	}
	return "save file successfully", nil
}

func AskInstanceType(ctx context.Context, cfg aws.Config, az string) (*InstanceType, error) {
	var instanceTypes []string

	client := ec2.NewFromConfig(cfg)

	/* AWS CLI Command Reference (https://docs.aws.amazon.com/cli/latest/reference/ec2/describe-instance-type-offerings.html)
	* Example========================================================
	aws ec2 describe-instance-type-offerings \
		--location-type availability-zone \
		--filters Name=location,Values=us-east-1a \
		--region us-east-1
	=================================================================*/

	output, err := client.DescribeInstanceTypeOfferings(ctx,
		&ec2.DescribeInstanceTypeOfferingsInput{
			Filters: []ec2_types.Filter{
				{Name: aws.String("location"), Values: []string{az}},
			},
			LocationType: ec2_types.LocationType("availability-zone"),
		},
	)

	if err != nil {
		instanceTypes = make([]string, 1)
		copy(instanceTypes, []string{defaultInstanceType})
	} else {
		instanceTypes = make([]string, 0, len(output.InstanceTypeOfferings))
		for _, offering := range output.InstanceTypeOfferings {
			instanceTypes = append(instanceTypes, aws.ToString((*string)(&offering.InstanceType)))
		}
	}
	sort.Strings(instanceTypes)

	var ec2Type string
	prompt := &survey.Select{
		Message: "Choose a EC2 Instance Type in AWS:",
		Options: instanceTypes,
	}
	if err := survey.AskOne(prompt, &ec2Type, survey.WithIcons(func(icons *survey.IconSet) {
		icons.SelectFocus.Format = "green+hb"
	}), survey.WithPageSize(20)); err != nil {
		return nil, err
	}

	return &InstanceType{Name: ec2Type}, nil
}

func AskAvailabilityZone(ctx context.Context, cfg aws.Config) (*AvailabilityZone, error) {
	var availabilityZones []string

	client := ec2.NewFromConfig(cfg)

	/* AWS CLI Command Reference (https://docs.aws.amazon.com/cli/latest/reference/ec2/describe-availability-zones.html)
	* Example========================================================
	aws ec2 describe-availability-zones --region us-east-1
	=================================================================*/
	output, err := client.DescribeAvailabilityZones(ctx, &ec2.DescribeAvailabilityZonesInput{})

	if err != nil {
		availabilityZones = make([]string, 1)
		copy(availabilityZones, []string{fmt.Sprintf("%sa", cfg.Region)})
	} else {
		availabilityZones = make([]string, 0, len(output.AvailabilityZones))
		for _, az := range output.AvailabilityZones {
			availabilityZones = append(availabilityZones, aws.ToString(az.ZoneName))
		}
	}

	sort.Strings(availabilityZones)

	var az string
	prompt := &survey.Select{
		Message: fmt.Sprintf("Choose a Availability Zone in %s", cfg.Region),
		Options: availabilityZones,
	}
	if err := survey.AskOne(prompt, &az, survey.WithIcons(func(icons *survey.IconSet) {
		icons.SelectFocus.Format = "green+hb"
	}), survey.WithPageSize(20)); err != nil {
		return nil, err
	}

	return &AvailabilityZone{Name: az}, nil
}

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

	/* AWS CLI Command Reference (https://docs.aws.amazon.com/cli/latest/reference/ec2/describe-images.html)
	* Example========================================================
	aws ec2 describe-images \
		--region us-east-1 \
		--owners amazon \
		--filters "Name=state,Values=available" "Name=architecture,Values=x86_64" "Name=root-device-type,Values=ebs" \
		--query 'Images[*].[ImageId]'
	=================================================================*/
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
