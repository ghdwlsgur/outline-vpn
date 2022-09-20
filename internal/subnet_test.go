package internal

import (
	"context"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/stretchr/testify/assert"
)

func TestCreateDefaultSubnet(t *testing.T) {
	assert := assert.New(t)

	cfg, err := NewConfig(context.Background(), "", "", "", "ap-northeast-2", "")
	assert.NoError(err)

	/*=========================================
	ap-northeast-2 (Seoul) - have default VPC

	Availability Zone 		default Subnet
	ap-northeast-2a			Inactive
	ap-northeast-2b			Inactive
	ap-northeast-2c			Active
	=========================================*/

	tests := map[string]struct {
		ctx      context.Context
		cfg      aws.Config
		isErr    bool
		isDelete bool
	}{
		"success": {
			ctx:      context.Background(),
			cfg:      cfg,
			isErr:    false,
			isDelete: true,
		},
	}

	for _, t := range tests {

		az_a := cfg.Region + "a"
		if az_a == "ap-northeast-2a" {
			existsDefaultSubnet, err := ExistsDefaultSubnet(t.ctx, t.cfg, az_a)
			assert.Equal(t.isErr, err != nil)
			assert.Equal(false, existsDefaultSubnet.Existence)

			createSubnet, err := CreateDefaultSubnet(t.ctx, t.cfg, az_a)
			assert.Equal(t.isErr, err != nil)

			exitsSubnet, err := ExistsTagSubnet(t.ctx, t.cfg)
			assert.Equal(t.isErr, err != nil)
			assert.Equal(true, exitsSubnet.Existence)

			deleteSubnet, err := DeleteTagSubnet(t.ctx, t.cfg, createSubnet.Id)
			assert.Equal(t.isErr, err != nil)
			assert.Equal(t.isDelete, deleteSubnet)
		}

		az_b := cfg.Region + "b"
		if az_b == "ap-northeast-2b" {
			existsDefaultSubnet, err := ExistsDefaultSubnet(t.ctx, t.cfg, az_b)
			assert.Equal(t.isErr, err != nil)
			assert.Equal(false, existsDefaultSubnet.Existence)

			createSubnet, err := CreateDefaultSubnet(t.ctx, t.cfg, az_b)
			assert.Equal(t.isErr, err != nil)

			existsSubnet, err := ExistsTagSubnet(t.ctx, t.cfg)
			assert.Equal(t.isErr, err != nil)
			assert.Equal(true, existsSubnet.Existence)

			deleteSubnet, err := DeleteTagSubnet(t.ctx, t.cfg, createSubnet.Id)
			assert.Equal(t.isErr, err != nil)
			assert.Equal(t.isDelete, deleteSubnet)
		}

		az_c := cfg.Region + "c"
		if az_c == "ap-northeast-2c" {
			existsDefaultSubnet, err := ExistsDefaultSubnet(t.ctx, t.cfg, az_c)
			assert.Equal(t.isErr, err != nil)
			assert.Equal(true, existsDefaultSubnet.Existence)
		}
	}
}
