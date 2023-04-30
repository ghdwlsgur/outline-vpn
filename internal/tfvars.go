package internal

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/AlecAivazis/survey/v2"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	ec2_types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/fatih/color"
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
	defaultIpv4Url      = "http://ipv4.icanhazip.com"
)

var defaultInstanceTagName string
var defaultKeyPairPath string

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

	EC2 struct {
		Existence    bool
		Id           string
		PublicIP     string
		LaunchTime   time.Time
		InstanceType string
	}
)

func CheckOutlineConnect(instance *EC2) (bool, error) {

	resp, err := http.Get(defaultIpv4Url)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	buf, err := io.ReadAll(resp.Body)
	currentIpv4 := strings.TrimSpace(string(buf))
	if err != nil {
		return false, err
	}
	return instance.PublicIP == currentIpv4, nil
}

func DeleteKeyPair() error {
	err := os.Remove(defaultKeyPairPath)
	if err != nil {
		return err
	}
	return nil
}

func ExistsKeyPair() bool {
	defaultKeyPairPath = fmt.Sprintf("%s/%s", os.Getenv("HOME"), ".ssh/vpn_ec2_key.pem")
	if _, err := os.Stat(defaultKeyPairPath); err == nil {
		return true
	}
	return false
}

func SaveTerraformVariable(jsonData map[string]interface{}, jsonFilePath string) (string, error) {
	jsonBuf, _ := json.Marshal(jsonData)
	err := os.WriteFile(jsonFilePath, jsonBuf, os.FileMode(0644))
	if err != nil {
		return "failed to save file", err
	}
	return "save file successfully", nil
}

func FindSpecificTagInstance(ctx context.Context, cfg aws.Config, region string) (*EC2, error) {
	client := ec2.NewFromConfig(cfg)
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
		return &EC2{}, err
	}

	if len(output.Reservations) > 0 {
		for _, reservations := range output.Reservations {
			for _, ec2 := range reservations.Instances {
				return &EC2{
					Existence:    true,
					Id:           aws.ToString(ec2.InstanceId),
					PublicIP:     aws.ToString(ec2.PublicIpAddress),
					LaunchTime:   aws.ToTime(ec2.LaunchTime),
					InstanceType: aws.ToString((*string)(&ec2.InstanceType)),
				}, nil
			}
		}
	}
	return &EC2{Existence: false}, nil
}

func FindTagInstance(ctx context.Context, cfg aws.Config) (*EC2, error) {

	client := ec2.NewFromConfig(cfg)
	defaultInstanceTagName = fmt.Sprintf("govpn-ec2-%s", cfg.Region)

	output, err := client.DescribeInstances(ctx,
		&ec2.DescribeInstancesInput{
			Filters: []ec2_types.Filter{
				{Name: aws.String("instance-state-name"), Values: []string{"running"}},
				{Name: aws.String("tag:Name"), Values: []string{defaultInstanceTagName}},
			},
		},
	)
	if err != nil {
		return &EC2{}, err
	}

	if len(output.Reservations) > 0 {
		for _, reservations := range output.Reservations {
			for _, ec2 := range reservations.Instances {
				return &EC2{
					Existence:    true,
					Id:           aws.ToString(ec2.InstanceId),
					PublicIP:     aws.ToString(ec2.PublicIpAddress),
					LaunchTime:   aws.ToTime(ec2.LaunchTime),
					InstanceType: aws.ToString((*string)(&ec2.InstanceType)),
				}, nil
			}
		}
	}
	return &EC2{Existence: false}, nil
}

func AskNewTfVars(region, az, instanceType, ami string) (string, error) {

	notice := color.New(color.Bold, color.FgHiCyan).PrintfFunc()
	notice("[detect tfvars file]==================================\n\nRegion:\t\t\t%s\nAvailability Zone:\t%s\nInstance Type:\t\t%s\nAMI:\t\t\t%s\n\n======================================================\n\n", region, az, instanceType, ami)

	prompt := &survey.Select{
		Message: "Do you want to proceed as above:",
		Options: []string{"Yes, I will it.", "No, I will change it."},
	}

	answer := ""
	if err := survey.AskOne(prompt, &answer, survey.WithIcons(func(icons *survey.IconSet) {
		icons.SelectFocus.Format = "green+hb"
	}), survey.WithPageSize(2)); err != nil {
		return "No, I will change it.", err
	}
	return answer, nil
}

func AskInstanceType(ctx context.Context, cfg aws.Config, az string) (*InstanceType, error) {
	var instanceTypesPerLocation []string
	var instanceTypesPerArchitecture []string
	var instanceTypes []string

	client := ec2.NewFromConfig(cfg)

	/* AWS CLI Command Reference (https://docs.aws.amazon.com/cli/latest/reference/ec2/describe-instance-type-offerings.html)
	* Example========================================================
	aws ec2 describe-instance-type-offerings \
		--location-type availability-zone \
		--filters Name=location,Values=us-east-1a \
		--region us-east-1

	aws ec2 describe-instance-types \
	--filters "Name=processor-info.supported-architecture,Values=x86_64" \
	--filters "Name=current-generation,Values=true"
	=================================================================*/

	outputByArchitecture, err := client.DescribeInstanceTypes(ctx,
		&ec2.DescribeInstanceTypesInput{
			Filters: []ec2_types.Filter{
				{Name: aws.String("processor-info.supported-architecture"), Values: []string{"x86_64"}},
				{Name: aws.String("current-generation"), Values: []string{"true"}},
			},
		},
	)

	if err != nil {
		instanceTypesPerArchitecture = make([]string, 1)
		copy(instanceTypesPerArchitecture, []string{defaultInstanceType})
	} else {
		instanceTypesPerArchitecture = make([]string, 0, len(outputByArchitecture.InstanceTypes))
		for _, offering := range outputByArchitecture.InstanceTypes {
			instanceTypesPerArchitecture = append(instanceTypesPerArchitecture, aws.ToString((*string)(&offering.InstanceType)))
		}
	}
	outputByLocation, err := client.DescribeInstanceTypeOfferings(ctx,
		&ec2.DescribeInstanceTypeOfferingsInput{
			Filters: []ec2_types.Filter{
				{Name: aws.String("location"), Values: []string{az}},
			},
			LocationType: ec2_types.LocationType("availability-zone"),
		},
	)

	if err != nil {
		instanceTypesPerLocation = make([]string, 1)
		copy(instanceTypesPerLocation, []string{defaultInstanceType})
	} else {
		instanceTypesPerLocation = make([]string, 0, len(outputByLocation.InstanceTypeOfferings))
		for _, offering := range outputByLocation.InstanceTypeOfferings {
			instanceTypesPerLocation = append(instanceTypesPerLocation, aws.ToString((*string)(&offering.InstanceType)))
		}
	}

	for _, v1 := range instanceTypesPerArchitecture {
		for _, v2 := range instanceTypesPerLocation {
			if v1 == v2 {
				instanceTypes = append(instanceTypes, v1)
			}
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
