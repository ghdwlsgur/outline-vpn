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

type (
	Workspace struct {
		Now       string
		List      []string
		Session   bool
		Existence bool
	}
)

func AskTerraformDestroy() (string, error) {
	prompt := &survey.Select{
		Message: "Do You Execute Terraform Destroy:",
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

func TerraformReady(ctx context.Context, ver string) (string, error) {
	installer := &releases.ExactVersion{
		Product: product.Terraform,
		Version: version.Must(version.NewVersion(ver)),
	}

	execPath, err := installer.Install(ctx)
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

func ExistsWorkspace(ctx context.Context, execPath, _defaultTerraformPath, regionName string) (*Workspace, error) {
	tf, err := SetRoot(execPath, _defaultTerraformPath)
	if err != nil {
		return &Workspace{}, err
	}
	list, name, err := tf.WorkspaceList(ctx)
	if err != nil {
		return &Workspace{}, err
	}

	for _, workspace := range list {
		if workspace == regionName {
			return &Workspace{List: list, Now: name, Existence: true}, nil
		}
	}

	return &Workspace{List: list, Now: name, Existence: false}, err
}

func SelectWorkspace(ctx context.Context, execPath, _defaultTerraformPath, regionName string, workspace *Workspace) (*Workspace, error) {

	tf, err := SetRoot(execPath, _defaultTerraformPath)
	if err != nil {
		return &Workspace{}, err
	}
	err = tf.WorkspaceSelect(ctx, regionName)
	if err != nil {
		return &Workspace{}, err
	}

	workspace.Now = regionName
	workspace.Session = true

	return workspace, nil
}

func CreateWorkspace(ctx context.Context, execPath, _defaultTerraformPath, regionName string) error {

	tf, _ := SetRoot(execPath, _defaultTerraformPath)
	err := tf.WorkspaceNew(ctx, regionName)
	if err != nil {
		return err
	}
	return nil
}
