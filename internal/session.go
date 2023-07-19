package internal

import (
	"context"
	"embed"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"runtime"
	"strings"
	"syscall"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
)

type (
	Target struct {
		Name          string
		PublicDomain  string
		PrivateDomain string
	}
)

func CreateStartSession(ctx context.Context, cfg aws.Config, input *ssm.StartSessionInput) (*ssm.StartSessionOutput, error) {
	client := ssm.NewFromConfig(cfg)

	return client.StartSession(ctx, input)
}

func CallProcess(process string, args ...string) error {
	call := exec.Command(process, args...)
	call.Stderr = os.Stderr
	call.Stdout = os.Stdout
	call.Stdin = os.Stdin

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT)
	done := make(chan bool, 1)
	go func() {
		for {
			select {
			case <-sigs:
			case <-done:
				return
			}
		}
	}()
	defer close(done)

	if err := call.Run(); err != nil {
		return WrapError(err)
	}

	return nil
}

//go:embed assets/*
var assets embed.FS

func GetAsset(filename string) ([]byte, error) {
	return assets.ReadFile("assets/" + filename)
}

func GetSSMPluginName() string {
	if strings.ToLower(runtime.GOOS) == "windows" {
		return "session-manager-plugin.exe"
	} else {
		return "session-manager-plugin"
	}
}

func GetSSMPlugin() ([]byte, error) {
	return GetAsset(getSSMPluginKey())
}

func getSSMPluginKey() string {
	return fmt.Sprintf("plugin/%s_%s/%s",
		strings.ToLower(runtime.GOOS), strings.ToLower(runtime.GOARCH), GetSSMPluginName())
}
