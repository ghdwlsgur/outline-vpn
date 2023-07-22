package internal

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"regexp"
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

	countryMapInKorean := map[string]string{
		"us-east-1":      "미국 동부 (버지니아 북부)",
		"us-east-2":      "미국 동부 (오하이오)",
		"us-west-1":      "미국 서부 (캘리포니아)",
		"us-west-2":      "미국 서부 (오레곤)",
		"ap-east-1":      "아시아 태평양 (홍콩)",
		"ap-south-2":     "아시아 태평양 (인도 - 하이데라바드)",
		"ap-south-1":     "아시아 태펴양 (인도 - 뭄바이)",
		"ap-northeast-3": "아시아 태평양 (일본 - 오사카)",
		"ap-northeast-1": "아시아 태평양 (일본 - 도쿄)",
		"ap-northeast-2": "아시아 태평양 (서울)",
		"ap-southeast-1": "아시아 태평양 (싱가포르)",
		"ap-southeast-2": "아시아 태평양 (호주 - 시드니)",
		"ap-southeast-4": "아시아 태평양 (호주 - 멜버른)",
		"ap-southeast-3": "아시아 태평양 (인도네시아 - 자카르타)",
		"ca-central-1":   "캐나다 (중부)",
		"eu-central-1":   "유럽 (독일 - 프랑크푸르트)",
		"eu-west-1":      "유럽 (아일랜드)",
		"eu-west-2":      "유럽 (영국 - 런던)",
		"eu-west-3":      "유럽 (프랑스 - 파리)",
		"eu-north-1":     "유럽 (스웨덴 - 스톡홀름)",
		"me-central-1":   "중동 (아랍에미리트)",
		"sa-east-1":      "남아메리카 (브라질 - 상파울루)",
		"af-south-1":     "아프리카 (남아프리카공화국 - 케이프타운)",
		"eu-south-1":     "유럽 (이탈리아 - 밀라노)",
		"eu-south-2":     "유럽 (스페인)",
		"eu-central-2":   "유럽 (스위스 - 취리히)",
		"me-south-1":     "중동 (바레인)",
	}

	if err != nil {
		return nil, err
	} else {
		regions = make([]string, 0, len(output.Regions))
		for _, region := range output.Regions {
			if len(*region.RegionName) <= 12 {
				regions = append(regions, aws.ToString(region.RegionName)+"\t\t"+countryMapInKorean[*region.RegionName])
			} else {
				regions = append(regions, aws.ToString(region.RegionName)+"\t"+countryMapInKorean[*region.RegionName])
			}
		}
	}
	sort.Strings(regions)

	answerWithKorean, err := AskPromptOptionList("Choose a region in AWS:", regions, len(regions))
	if err != nil {
		return nil, err
	}

	re := regexp.MustCompile(`^[^\t]+`)
	region := re.FindString(answerWithKorean)

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
