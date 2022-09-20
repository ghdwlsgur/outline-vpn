package internal

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/aws/aws-sdk-go-v2/config"
)

var (
	mockProfile   string
	mockAwsKey    string
	mockAwsSecret string
	mockRegion    string

	_credentialWithTemporary = fmt.Sprintf("%s_temporary", config.DefaultSharedCredentialsFilename())
)

func TestMain(m *testing.M) {
	if os.Getenv("CIRCLECI") != "" {
		os.Exit(0)
	}

	mockProfile = "default"

	if _, err := os.Stat(_credentialWithTemporary); os.IsNotExist(err) {
		filename := filepath.Join(os.Getenv("HOME"), "/.aws/credentials")

		if _, err := os.Stat(filename); os.IsNotExist(err) {
			os.Exit(0)
		} else {
			cfg, err := config.LoadDefaultConfig(context.Background(),
				config.WithSharedConfigProfile(mockProfile),
				config.WithSharedConfigFiles([]string{config.DefaultSharedConfigFilename()}),
				config.WithSharedCredentialsFiles([]string{config.DefaultSharedCredentialsFilename()}),
			)
			if err != nil {
				panic(err)
			}

			cred, err := cfg.Credentials.Retrieve(context.Background())
			if err != nil {
				panic(err)
			}

			mockAwsKey = cred.AccessKeyID
			mockAwsSecret = cred.SecretAccessKey
			mockRegion = cfg.Region
			os.Exit(m.Run())
		}

	} else {
		os.Setenv("AWS_SHARED_CREDENTIALS_FILE", _credentialWithTemporary)

		cfg, err := config.LoadDefaultConfig(context.Background(),
			config.WithSharedConfigProfile(mockProfile),
			config.WithSharedConfigFiles([]string{}),
			config.WithSharedCredentialsFiles([]string{_credentialWithTemporary}),
		)
		if err != nil {
			panic(err)
		}

		cred, err := cfg.Credentials.Retrieve(context.Background())
		if err != nil {
			panic(err)
		}

		mockAwsKey = cred.AccessKeyID
		mockAwsSecret = cred.SecretAccessKey
		mockRegion = cfg.Region
		os.Exit(m.Run())
	}
}
