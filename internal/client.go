package internal

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"os"
	"sort"

	"github.com/AlecAivazis/survey/v2"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	ec2_types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/fatih/color"
	"github.com/go-resty/resty/v2"
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

func readOutlineInfo(region string) (*OutlineInfo, error) {
	outlineJsonPath := func(path, region string) string {
		return path + region + "/outline.json"
	}("/opt/homebrew/lib/outline-vpn/govpn-terraform/terraform.tfstate.d/", region)

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

func CreateAccessKey(region string) error {
	apiURL, err := GetApiURL(region)
	if err != nil {
		return err
	}
	url := fmt.Sprintf("%s/%s", apiURL, "access-keys")

	client := resty.New()
	client.SetTLSClientConfig(&tls.Config{InsecureSkipVerify: true})
	resp, err := client.R().
		SetHeader("Content-Type", "application/json").
		Post(url)
	if err != nil {
		return err
	}

	if resp.StatusCode() == 200 {
		return nil
	}
	return err
}

func GetAccessKeys(region string) ([]string, error) {
	apiURL, err := GetApiURL(region)
	if err != nil {
		return nil, err
	}
	url := fmt.Sprintf("%s/%s", apiURL, "access-keys")

	client := resty.New()
	client.SetTLSClientConfig(&tls.Config{InsecureSkipVerify: true})
	resp, err := client.R().
		SetHeader("Content-Type", "application/json").
		Get(url)
	if err != nil {
		return nil, err
	}

	var accessKeys AccessKeys
	if resp.StatusCode() == 200 {
		err = json.Unmarshal(resp.Body(), &accessKeys)
		if err != nil {
			return nil, err
		}
	}

	return accessKeys.GetAccessUrlList(), nil
}

func DeleteAccessKey(region string, id int) error {
	apiURL, err := GetApiURL(region)
	if err != nil {
		return err
	}
	url := fmt.Sprintf("%s/%s/%d", apiURL, "access-keys", id)

	client := resty.New()
	client.SetTLSClientConfig(&tls.Config{InsecureSkipVerify: true})

	resp, err := client.R().Delete(url)
	if err != nil {
		return err
	}

	if resp.StatusCode() == 204 {
		return nil
	}
	return err
}

func RenameAccessKey(region string, id int, name string) error {
	apiURL, err := GetApiURL(region)
	if err != nil {
		return err
	}
	url := fmt.Sprintf("%s/%s/%d/%s", apiURL, "access-keys", id, "name")

	client := resty.New()
	client.SetTLSClientConfig(&tls.Config{InsecureSkipVerify: true})

	putData := map[string]string{
		"name": name,
	}
	resp, err := client.R().
		SetHeader("Content-Type", "application/json").
		SetBody(putData).
		Put(url)
	if err != nil {
		return err
	}

	if resp.StatusCode() == 204 {
		return nil
	}
	return err
}

func AddDataLimitAccessKey(region string, id int, limit int) error {
	apiURL, err := GetApiURL(region)
	if err != nil {
		return err
	}
	url := fmt.Sprintf("%s/%s/%d/%s", apiURL, "access-keys", id, "data-limit")

	client := resty.New()
	client.SetTLSClientConfig(&tls.Config{InsecureSkipVerify: true})

	putData := map[string]map[string]int{
		"limit": {
			"bytes": limit,
		},
	}
	resp, err := client.R().
		SetHeader("Content-Type", "application/json").
		SetBody(putData).
		Put(url)
	if err != nil {
		return err
	}

	if resp.StatusCode() == 204 {
		return nil
	}
	return err
}

func DeleteDataLimitAccessKey(region string, id int) error {
	apiURL, err := GetApiURL(region)
	if err != nil {
		return err
	}
	url := fmt.Sprintf("%s/%s/%d/%s", apiURL, "access-keys", id, "data-limit")

	client := resty.New()
	client.SetTLSClientConfig(&tls.Config{InsecureSkipVerify: true})

	resp, err := client.R().Delete(url)
	if err != nil {
		return err
	}

	if resp.StatusCode() == 204 {
		return nil
	}
	return err
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
