<div align="center">

<img width="30%" alt="govpn-logo" src="https://user-images.githubusercontent.com/77400522/191333400-257c67f7-d20f-4b44-a4c0-9e9ab9fe278c.png">

<br>
<br>

### Outline-VPN

> It can help you quickly provision a Shadowsocks-based VPN server on an AWS EC2 instance and assist you in using [Outline VPN](https://getoutline.org/) to use the VPN.

<br>

![GitHub tag (latest SemVer)](https://img.shields.io/github/v/tag/ghdwlsgur/outline-vpn?color=success&label=version&sort=semver)
[![Go Report Card](https://goreportcard.com/badge/github.com/ghdwlsgur/outline-vpn)](https://goreportcard.com/report/github.com/ghdwlsgur/outline-vpn)
[![Codacy Badge](https://app.codacy.com/project/badge/Grade/31be448d2ace4634a1dfe7ce2d083036)](https://www.codacy.com/gh/ghdwlsgur/govpn/dashboard?utm_source=github.com&utm_medium=referral&utm_content=ghdwlsgur/outline-vpn&utm_campaign=Badge_Grade)
[![Maintainability](https://api.codeclimate.com/v1/badges/6d14b66ab49c8d4b64c0/maintainability)](https://codeclimate.com/github/ghdwlsgur/outline-vpn/maintainability)
[![circle ci](https://circleci.com/gh/ghdwlsgur/govpn.svg?style=svg)](https://circleci.com/gh/ghdwlsgur/outline-vpn)

</div>

# Overview

Once the user selects a machine image, instance type, region, and availability zone, an EC2 instance is created in the default subnet within the selected availability zone in the default VPC. If you don't have a default VPC or default subnet, we can assist you in creating them. You can create one EC2 instance per region. To use the VPN service, simply paste the access key into the [Outline Client](https://getoutline.org/ko/get-started/#step-3) App.

[🤝 Join Telegram Outline Channel](https://t.me/outlinevpnofficial)

Outline is a Shadowsocks-based proxy created by Google. It lets you create and manage Shadowsocks servers easily. Outline clients are also well-designed applications available for all platforms.

> It usually works as below.

```bash
[Outline client] <-> [Outline server] <-> [Internet]
```

# Security

After creating the VPN server, the UDP and TCP ports of the security group are configured to allow access only from the public IP of the user who owns the VPN server to access the VPN service.

# Prerequisite

Provisioning speed may vary depending on instance type.

### EC2

- [required] ec2:CreateDefaultVpc, ec2:DescribeVpcs, ec2:DeleteVpc
- [required] ec2:CreateDefaultSubnet, ec2:DescribeSubnets, ec2:DeleteSubnet
- [required] ec2:DeleteInternetGateway, ec2:DescribeInternetGateways, ec2:DetachInternetGateway
- [required] ec2:CreateTags, ec2:DescribeInstances, ec2:DescribeInstanceTypeOfferings, ec2:DescribeAvailabilityZones, ec2:DescribeImages, ec2:DescribeRegions

### Client

- [required] AWS Configure

  > Execute command that `aws configure`

  ```bash
  $ aws configure
  AWS Access Key ID :
  AWS Secret Access Key :
  Default region name :
  Default output format :
  ```

- [optional] `~/.aws/credentials` or `~/.aws/credentials_temporary`

### Library / Program

- [required] jq

  ```bash
  brew install jq
  ```

- [required] rsync

  ```bash
  brew install rsync
  ```

- [required] terraform

  ```bash
  # install
  brew tap hashicorp/tap
  brew install hashicorp/tap/terraform

  # upgrade
  brew upgrade hashicorp/tap/terraform
  ```

- [required] [Outline Client](https://getoutline.org/ko/get-started/#step-3) (VPN connection purpose)

# Result

> example region: us-east-1

- [optional tag: `govpn-vpc`] default vpc
- [optional tag: `govpn-subnet`] default subnet
- [required tag: `govpn-ec2-us-east-1`] EC2
- [required tag: `govpn_us-east-1`] Key Pair and Pem file (.ssh/govpn_us-east-1.pem)
- [required tag: `govpn-sg-us-east-1`] Security Group

### All the resources you create can be tracked with the tag function provided by AWS. This thoroughly avoids unexpected cost of resources.

# Installation

### Homebrew

```bash
# [install]

brew tap ghdwlsgur/outline-vpn
brew install outline-vpn

# [upgrade]

brew upgrade outline-vpn
```

### [Download](https://github.com/ghdwlsgur/outline-vpn/releases)

# How to use (command)

### create

> Create a VPN server

```bash
$ outline-vpn create

# Provision EC2 in the us-east-1 region.
$ outline-vpn create -r us-east-1

# Provision EC2 in the ap-northeast-2 region.
$ outline-vpn create -r ap-northeast-2
```

> After executing the `outline-vpn create` command, register the received access key on the Outline Client App and connect.

<p align="center">
 <img src="https://user-images.githubusercontent.com/77400522/236384565-e86bde42-7d9d-4c50-a613-be773d7d315b.png" />
</p>

### destroy

> Delete a VPN server

```bash
$ outline-vpn destroy

# Terminate EC2 in the us-east-1 region.
$ outline-vpn destroy -r us-east-1

# Terminate EC2 in the ap-northeast-2 region.
$ outline-vpn destroy -r ap-northeast-2
```

<p align="center">
 <img src="https://user-images.githubusercontent.com/77400522/236384764-ecc410ea-930e-4dfa-9cff-62e3f5472e38.png" />
</p>

### find

> Find instances created using the outlinevpn CLI tool.

```bash
$ govpn find
```

<p align="center">
  <img src="https://user-images.githubusercontent.com/77400522/236385450-36c78250-23be-410c-b352-3c802922e55b.png" />
</p>

<p align="center">
  <img src="https://user-images.githubusercontent.com/77400522/236385538-1382e062-ac87-41cc-8d94-d2c4d9455ac3.png" />
</p>

# Trouble Shooting

while executing terraform init you might face the below error if you are working in a MAC with apple chip in it.

<img width="863" alt="image" src="https://user-images.githubusercontent.com/77400522/233235056-2b4941ee-137c-4989-9602-f646ef4baa24.png">

```bash
brew install kreuzwerker/taps/m1-terraform-provider-helper
m1-terraform-provider-helper activate
m1-terraform-provider-helper install hashicorp/template -v v2.2.0
```

# License

Outline-VPN is licensed under the [MIT](https://github.com/ghdwlsgur/outline-vpn/blob/master/LICENSE)
