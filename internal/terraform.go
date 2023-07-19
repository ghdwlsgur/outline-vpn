package internal

import (
	"context"
	"fmt"
	"os"

	"github.com/hashicorp/go-version"
	"github.com/hashicorp/hc-install/product"
	"github.com/hashicorp/hc-install/releases"
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/hashicorp/terraform-exec/tfexec"
	"github.com/zclconf/go-cty/cty"
)

type (
	Workspace struct {
		Now       string
		List      []string
		Session   bool
		Existence bool
		Path      string
	}
)

const (
	rootModule    = "ghdwlsgur/outline-vpn/ghdwlsgur"
	moduleVersion = "1.0.11"
	moduleName    = "outline-vpn"
)

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

func GetWorkspaceList(ctx context.Context, execPath, _defaultTerraformPath string) ([]string, error) {
	tf, err := SetRoot(execPath, _defaultTerraformPath)
	if err != nil {
		return nil, err
	}

	list, _, err := tf.WorkspaceList(ctx)
	if err != nil {
		return nil, err
	}

	// exclude default workspace
	return list[1:], nil
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
	return tf.WorkspaceNew(ctx, regionName)
}

func CreateTf(workSpacePath string, region, ami, instanceType, az string) error {

	err := CreateMainDotTf(workSpacePath, region, ami, instanceType, az)
	if err != nil {
		return err
	}

	err = CreateProviderDotTf(workSpacePath, region)
	if err != nil {
		return err
	}

	err = CreateOutputDotTf(workSpacePath)
	if err != nil {
		return err
	}

	err = CreateKeyDotTf(workSpacePath)
	if err != nil {
		return err
	}

	return nil
}

func CreateMainDotTf(workSpacePath string, region, ami, instanceType, az string) error {
	var fileName = fmt.Sprintf(workSpacePath + "/main.tf")

	f := hclwrite.NewEmptyFile()
	rootBody := f.Body()

	moduleBlock := rootBody.AppendNewBlock("module", []string{moduleName})
	moduleBody := moduleBlock.Body()
	moduleBody.SetAttributeValue("source", cty.StringVal(rootModule))
	moduleBody.SetAttributeValue("version", cty.StringVal(moduleVersion))
	moduleBody.SetAttributeValue("aws_region", cty.StringVal(region))
	moduleBody.SetAttributeValue("ec2_ami", cty.StringVal(ami))
	moduleBody.SetAttributeValue("instance_type", cty.StringVal(instanceType))
	moduleBody.SetAttributeValue("availability_zone", cty.StringVal(az))
	moduleBody.SetAttributeTraversal("key_name", hcl.Traversal{
		hcl.TraverseRoot{Name: "aws_key_pair"},
		hcl.TraverseAttr{Name: "govpn_key"},
		hcl.TraverseAttr{Name: "key_name"},
	})
	moduleBody.SetAttributeTraversal("private_key_openssh", hcl.Traversal{
		hcl.TraverseRoot{Name: "tls_private_key"},
		hcl.TraverseAttr{Name: "tls"},
		hcl.TraverseAttr{Name: "private_key_openssh"},
	})
	moduleBody.SetAttributeTraversal("private_key_pem", hcl.Traversal{
		hcl.TraverseRoot{Name: "tls_private_key"},
		hcl.TraverseAttr{Name: "tls"},
		hcl.TraverseAttr{Name: "private_key_pem"},
	})

	return os.WriteFile(fileName, f.Bytes(), 0644)
}

func CreateProviderDotTf(workSpacePath string, region string) error {
	var fileName = fmt.Sprintf(workSpacePath + "/provider.tf")

	f := hclwrite.NewEmptyFile()
	rootBody := f.Body()

	providerBlock := rootBody.AppendNewBlock("provider", []string{"aws"})
	providerBody := providerBlock.Body()
	providerBody.SetAttributeValue("region", cty.StringVal(region))

	return os.WriteFile(fileName, f.Bytes(), 0644)
}

func CreateOutputDotTf(workSpacePath string) error {
	var fileName = fmt.Sprintf(workSpacePath + "/output.tf")

	f := hclwrite.NewEmptyFile()
	rootBody := f.Body()

	sshPrivateKeyBlock := rootBody.AppendNewBlock("output", []string{"ssh_private_key"})
	sshPrivateKeyBody := sshPrivateKeyBlock.Body()
	sshPrivateKeyBody.SetAttributeTraversal("value", hcl.Traversal{
		hcl.TraverseRoot{Name: "tls_private_key"},
		hcl.TraverseAttr{Name: "tls"},
		hcl.TraverseAttr{Name: "private_key_pem"},
	})
	sshPrivateKeyBody.SetAttributeValue("sensitive", cty.BoolVal(true))

	rootBody.AppendNewline()

	accessKeyBlock := rootBody.AppendNewBlock("output", []string{"access_key"})
	accessKeyBody := accessKeyBlock.Body()
	accessKeyBody.SetAttributeTraversal("value", hcl.Traversal{
		hcl.TraverseRoot{Name: "module"},
		hcl.TraverseAttr{Name: moduleName},
		hcl.TraverseAttr{Name: "OutlineClientAccessKey"},
	})

	return os.WriteFile(fileName, f.Bytes(), 0644)
}

func CreateKeyDotTf(workSpacePath string) error {
	var fileName = fmt.Sprintf(workSpacePath + "/key.tf")

	f := hclwrite.NewEmptyFile()
	rootBody := f.Body()

	tlsBlock := rootBody.AppendNewBlock("resource", []string{"tls_private_key", "tls"})
	tlsBody := tlsBlock.Body()
	tlsBody.SetAttributeValue("algorithm", cty.StringVal("RSA"))
	tlsBody.SetAttributeValue("rsa_bits", cty.NumberIntVal(4096))

	rootBody.AppendNewline()

	govpnBlock := rootBody.AppendNewBlock("resource", []string{"aws_key_pair", "govpn_key"})
	govpnBody := govpnBlock.Body()
	govpnBody.SetAttributeTraversal("key_name", hcl.Traversal{
		hcl.TraverseRoot{Name: "\"govpn_${module.outline-vpn.Region}\""},
	})
	govpnBody.SetAttributeTraversal("public_key", hcl.Traversal{
		hcl.TraverseRoot{Name: "tls_private_key"},
		hcl.TraverseAttr{Name: "tls"},
		hcl.TraverseAttr{Name: "public_key_openssh"},
	})

	return os.WriteFile(fileName, f.Bytes(), 0644)
}
