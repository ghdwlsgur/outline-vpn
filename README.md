<div align="center">

<img width="30%" alt="govpn-logo" src="https://user-images.githubusercontent.com/77400522/191333400-257c67f7-d20f-4b44-a4c0-9e9ab9fe278c.png">

<br>
<br>

### GoVPN

> It helps you quickly provision cloud servers for using [Outline VPN](https://getoutline.org/)

<br>

[![Stand With Ukraine](https://raw.githubusercontent.com/vshymanskyy/StandWithUkraine/main/badges/StandWithUkraine.svg)](https://stand-with-ukraine.pp.ua)
[![Go Report Card](https://goreportcard.com/badge/github.com/ghdwlsgur/govpn)](https://goreportcard.com/report/github.com/ghdwlsgur/govpn)
![GitHub tag (latest SemVer)](https://img.shields.io/github/v/tag/ghdwlsgur/govpn?color=success&label=version&sort=semver)
[![Codacy Badge](https://app.codacy.com/project/badge/Grade/31be448d2ace4634a1dfe7ce2d083036)](https://www.codacy.com/gh/ghdwlsgur/govpn/dashboard?utm_source=github.com&utm_medium=referral&utm_content=ghdwlsgur/govpn&utm_campaign=Badge_Grade)
[![Maintainability](https://api.codeclimate.com/v1/badges/6d14b66ab49c8d4b64c0/maintainability)](https://codeclimate.com/github/ghdwlsgur/govpn/maintainability)
[![circle ci](https://circleci.com/gh/ghdwlsgur/govpn.svg?style=svg)](https://circleci.com/gh/ghdwlsgur/govpn)

</div>

# Overview

After the user selects an `machine image`, `instance type`, `region`, and `availability zone`, an EC2 is created in the default subnet within the selected availability zone in the default vpc. If you don't have a default vpc or default subnet, we'll help you defulat vpc or default subnet. You can create one EC2 instance for each region. You can use the VPN service by downloading and pasting the [Outline Client](https://getoutline.org/ko/get-started/#step-3) App.

# Prerequisite

### EC2

- ⚠️ [required] ec2:CreateDefaultVpc, ec2:DescribeVpcs, ec2:DeleteVpc
- ⚠️ [required] ec2:CreateDefaultSubnet, ec2:DescribeSubnets, ec2:DeleteSubnet
- ⚠️ [required] ec2:DeleteInternetGateway, ec2:DescribeInternetGateways, ec2:DetachInternetGateway
- ⚠️ [required] ec2:CreateTags, ec2:DescribeInstances, ec2:DescribeInstanceTypeOfferings, ec2:DescribeAvailabilityZones, ec2:DescribeImages, ec2:DescribeRegions

### Client

- ⚠️ [required] AWS Configure

  > Execute command that `aws configure`

  ```bash
  $ aws configure
  AWS Access Key ID :
  AWS Secret Access Key :
  Default region name :
  Default output format :
  ```

- [optional] `~/.aws/credentials` or `~/.aws/credentials_temporary`

# Installation

### Homebrew

```bash
# [install]

brew tap ghdwlsgur/govpn
brew install govpn

# [upgrade]

brew upgrade govpn
```

### [Download](https://github.com/ghdwlsgur/govpn/releases)

# How to use

### command

```bash
$ govpn apply

# Provision EC2 in the us-east-1 region.

$ govpn apply -r us-east-1

# Provision EC2 in the ap-northeast-2 region.

$ govpn destroy -r ap-northeast-2
```

<p align="center">
<img src="https://user-images.githubusercontent.com/77400522/191327327-8e757d14-1d8b-4996-a69b-f5d04ec446fb.mov" width="680", height="550" />
</p>

```bash
$ govpn destroy

# Terminate EC2 in the us-east-1 region.

$ govpn destroy -r us-east-1

# Terminate EC2 in the ap-northeast-2 region.

$ govpn destroy -r ap-northeast-2
```

<p align="center">
<img src="https://user-images.githubusercontent.com/77400522/191327884-2f418d58-2da9-4252-ac20-edecd91eb7d2.mov" width="680", height="550" />
</p>

# License

GoVPN is licensed under the [MIT](https://github.com/ghdwlsgur/govpn/blob/master/LICENSE)
