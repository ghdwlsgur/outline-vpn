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
	"github.com/fatih/color"
	"github.com/hairyhenderson/go-which"
)

type (
	Region struct {
		Name string
	}
)

type OutlineInfo struct {
	ManagementUdpPort int    `json:"ManagementUdpPort"`
	VpnTcpUdpPort     int    `json:"VpnTcpUdpPort"`
	ApiURL            string `json:"ApiUrl"`
	CertSha256        string `json:"CertSha256"`
}

type DataLimit struct {
	Bytes int `json:"bytes"`
}

type AccessKey struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Password  string `json:"password"`
	Port      int    `json:"port"`
	Method    string `json:"method"`
	DataLimit DataLimit
	AccessURL string `json:"accessUrl"`
}

type AccessKeys struct {
	Keys []AccessKey `json:"accessKeys"`
}

func (l *AccessKeys) GetAccessUrlList() []string {
	var result []string
	result = make([]string, 0, len(l.Keys))
	for _, v := range l.Keys {
		result = append(result, v.AccessURL)
	}
	return result
}

func ReturnTerraformPath(region string) string {
	path := which.Which("outline-vpn")
	path = strings.Replace(path, "bin", "lib", -1)
	return path + "/outline-vpn/terraform.tfstate.d/" + region
}

func readOutlineInfo(region string) (*OutlineInfo, error) {
	outlineJsonPath := ReturnTerraformPath(region) + "/outline.json"

	b, err := os.ReadFile(outlineJsonPath)
	if err != nil {
		return nil, err
	}

	var outlineInfo OutlineInfo
	err = json.Unmarshal(b, &outlineInfo)
	if err != nil {
		return nil, err
	}

	return &outlineInfo, nil
}

func GetCertSha256(region string) (string, error) {
	outlineInfo, err := readOutlineInfo(region)
	if err != nil {
		return "", err
	}

	return outlineInfo.CertSha256, nil
}

func GetApiURL(region string) (string, error) {
	outlineInfo, err := readOutlineInfo(region)
	if err != nil {
		return "", err
	}
	return outlineInfo.ApiURL, nil
}

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

	answer, err := AskPromptOptionList("Choose a region in AWS:", regions, len(regions))
	if err != nil {
		return nil, err
	}

	return &Region{Name: answer}, nil
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
	fmt.Printf("%s %s %s [ %s: %s ]\n",
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

func AskPromptOptionList(Message string, Options []string, size int) (string, error) {
	prompt := &survey.Select{
		Message: Message,
		Options: Options,
	}

	answer := ""
	if err := survey.AskOne(prompt, &answer, survey.WithIcons(func(icons *survey.IconSet) {
		icons.SelectFocus.Format = "green+hb"
	}), survey.WithPageSize(size)); err != nil {
		return "No", err
	}

	return answer, nil
}

func AskTerraformExecution(Message string) (string, error) {
	prompt := &survey.Select{
		Message: Message,
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
