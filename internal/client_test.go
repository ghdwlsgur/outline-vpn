package internal

import (
	"fmt"
	"testing"
)

func TestGetAccessKeys(t *testing.T) {
	// assert := assert.New(t)

	test, err := GetAccessKeys("ap-northeast-1")
	if err != nil {
		fmt.Println(err)
	}

	fmt.Println(test)
}

func TestDeleteAccessKey(t *testing.T) {
	DeleteAccessKey("ap-northeast-1", "4")
}

func TestRenameAccessKey(t *testing.T) {
	RenameAccessKey("ap-northeast-1", 5, "jinhyeok")
}

func TestAddDataLimitAccessKey(t *testing.T) {
	AddDataLimitAccessKey("ap-northeast-1", 5, 123456)
}

func TestDeleteDataLimitAccessKey(t *testing.T) {
	DeleteDataLimitAccessKey("ap-northeast-1", 7)
}

func TestCreateAccessKey(t *testing.T) {
	CreateAccessKey("ap-northeast-1")
}

func TestGetSha256(t *testing.T) {
	result, err := GetCertSha256("ap-northeast-1")
	if err != nil {
		fmt.Println(err)
	}

	fmt.Println(result)
}

func TestGetApiURL(t *testing.T) {
	result, err := GetApiURL("ap-northeast-1")
	if err != nil {
		fmt.Println(err)
	}

	fmt.Println(result)
}
