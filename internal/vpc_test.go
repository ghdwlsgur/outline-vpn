package internal

import (
	"context"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/stretchr/testify/assert"
)

func TestCreateDefaultVpc(t *testing.T) {
	assert := assert.New(t)

	cfg, err := NewConfig(context.Background(), "", "", "", "ap-southeast-2", "")
	assert.NoError(err)

	/*============================================================================================================================
	My Endpoint STS status
	Region				Status				Default Vpc		result

	ap-northeast-1		Active				Active			[err] A Default VPC already exists for this account in this region.
	af-south-1			Inactive							[err] AWS was not able to validate the provided access credentials
	ap-southeast-2		Active				Inactive		[success]
	============================================================================================================================*/

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
		createVpc, err := CreateDefaultVpc(t.ctx, t.cfg)

		if cfg.Region == "ap-northeast-2" {
			assert.Equal(t.isErr, err == nil)
		}
		if cfg.Region == "af-south-1" {
			assert.Equal(t.isErr, err == nil)
		}

		if cfg.Region == "ap-southeast-2" {
			assert.Equal(t.isErr, err != nil)
			assert.Equal(true, createVpc.New)

			existsTagVpc, err := ExistsTagVpc(t.ctx, t.cfg)
			assert.Equal(t.isErr, err != nil)
			assert.Equal(true, existsTagVpc.Existence)

			deleteVpc, err := DeleteTagVpc(t.ctx, t.cfg, createVpc.Id)
			assert.Equal(t.isErr, err != nil)
			assert.Equal(t.isDelete, deleteVpc)

		}
	}
}
