package internal

import (
	"context"
	"os"
	"os/exec"

	"github.com/AlecAivazis/survey/v2"
	"github.com/hashicorp/go-version"
	"github.com/hashicorp/hc-install/product"
	"github.com/hashicorp/hc-install/releases"
	"github.com/hashicorp/terraform-exec/tfexec"
)

func TerraformReady(ver string) (string, error) {
	installer := &releases.ExactVersion{
		Product: product.Terraform,
		Version: version.Must(version.NewVersion(ver)),
	}

	execPath, err := installer.Install(context.Background())
	if err != nil {
		return "", err
	}
	return execPath, nil
}

func SetRoot(execPath, terraformPath string) (*tfexec.Terraform, error) {
	tf, err := tfexec.NewTerraform(terraformPath, execPath)
	if err != nil {
		return nil, err
	}

	return tf, nil
}

func TerraformApply(terraformPath string) error {
	cmd := exec.Command("sh", "command/deploy.sh")
	cmd.Dir = terraformPath
	cmd.Stdout = os.Stdout

	if err := cmd.Run(); err != nil {
		return err
	}
	return nil
}

func TerraformDestroy(terraformPath string) error {
	cmd := exec.Command("sh", "command/destroy.sh")
	cmd.Dir = terraformPath
	cmd.Stdout = os.Stdout

	if err := cmd.Run(); err != nil {
		return err
	}
	return nil
}

func AskTerraformApply() (string, error) {
	prompt := &survey.Select{
		Message: "Do You Provision EC2 Instance:",
		Options: []string{"Yes", "No"},
	}

	answer := ""
	if err := survey.AskOne(prompt, &answer, survey.WithIcons(func(icons *survey.IconSet) {
		icons.SelectFocus.Format = "green+hb"
	}), survey.WithPageSize(2)); err != nil {
		return "No", err
	}

	return answer, nil
}
