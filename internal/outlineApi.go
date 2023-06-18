package internal

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"os"

	"github.com/go-resty/resty/v2"
)

func checkOutlineJsonExists(list []string) []string {
	workspace := make([]string, 0)

	for _, region := range list {
		outlineJsonPath := func(path, workspace string) string {
			return path + workspace + "/outline.json"
		}("/opt/homebrew/lib/outline-vpn/outline-vpn/terraform.tfstate.d/", region)

		_, err := os.Stat(outlineJsonPath)
		if !os.IsNotExist(err) {
			workspace = append(workspace, region)
		}
	}

	return workspace
}

func ValidateOutlineJson(ctx context.Context, terraformVersion, _defaultTerraformPath string) ([]string, error) {

	execPath, err := TerraformReady(ctx, terraformVersion)
	if err != nil {
		return nil, err
	}

	list, err := GetWorkspaceList(ctx, execPath, _defaultTerraformPath)
	if err != nil {
		return nil, err
	}

	if len(list) > 0 {
		list = checkOutlineJsonExists(list)
	}

	return list, nil
}

func CreateAccessKey(region string) (*AccessKey, error) {
	apiURL, err := GetApiURL(region)
	if err != nil {
		return nil, err
	}
	url := fmt.Sprintf("%s/%s", apiURL, "access-keys")

	client := resty.New()
	client.SetTLSClientConfig(&tls.Config{InsecureSkipVerify: true})
	resp, err := client.R().
		SetHeader("Content-Type", "application/json").
		Post(url)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode() == 201 {
		var result AccessKey
		err := json.Unmarshal(resp.Body(), &result)
		if err != nil {
			return nil, err
		}

		return &result, err
	}

	return nil, err
}

func GetAccessKeys(region string) (*AccessKeys, error) {
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

	// fmt.Println(accessKeys.Keys[0].ID)
	// return accessKeys.GetAccessUrlList(), nil

	return &accessKeys, nil
}

func DeleteAccessKey(region string, id string) error {
	apiURL, err := GetApiURL(region)
	if err != nil {
		return err
	}
	url := fmt.Sprintf("%s/%s/%s", apiURL, "access-keys", id)

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
