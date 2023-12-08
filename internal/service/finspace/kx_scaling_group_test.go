// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package finspace_test

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/finspace"
	"github.com/aws/aws-sdk-go-v2/service/finspace/types"
	sdkacctest "github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/hashicorp/terraform-provider-aws/internal/acctest"
	"github.com/hashicorp/terraform-provider-aws/internal/conns"
	"github.com/hashicorp/terraform-provider-aws/internal/create"
	tffinspace "github.com/hashicorp/terraform-provider-aws/internal/service/finspace"
	"github.com/hashicorp/terraform-provider-aws/names"
)

func TestAccFinSpaceKxScalingGroup_basic(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping long-running test in short mode")
	}

	ctx := acctest.Context(t)
	var KxScalingGroup finspace.GetKxScalingGroupOutput
	rName := sdkacctest.RandomWithPrefix(acctest.ResourcePrefix)
	resourceName := "aws_finspace_kx_scaling_group.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			acctest.PreCheck(ctx, t)
			acctest.PreCheckPartitionHasService(t, finspace.ServiceID)
		},
		ErrorCheck:               acctest.ErrorCheck(t, finspace.ServiceID),
		ProtoV5ProviderFactories: acctest.ProtoV5ProviderFactories,
		CheckDestroy:             testAccCheckKxScalingGroupDestroy(ctx),
		Steps: []resource.TestStep{
			{
				Config: testAccKxScalingGroupConfig_basic(rName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckKxScalingGroupExists(ctx, resourceName, &KxScalingGroup),
					resource.TestCheckResourceAttr(resourceName, "name", rName),
					resource.TestCheckResourceAttr(resourceName, "status", string(types.KxScalingGroupStatusActive)),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccFinSpaceKxScalingGroup_dissappears(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping long-running test in short mode")
	}

	ctx := acctest.Context(t)
	var KxScalingGroup finspace.GetKxScalingGroupOutput
	rName := sdkacctest.RandomWithPrefix(acctest.ResourcePrefix)
	resourceName := "aws_finspace_kx_scaling_group.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			acctest.PreCheck(ctx, t)
			acctest.PreCheckPartitionHasService(t, finspace.ServiceID)
		},
		ErrorCheck:               acctest.ErrorCheck(t, finspace.ServiceID),
		ProtoV5ProviderFactories: acctest.ProtoV5ProviderFactories,
		CheckDestroy:             testAccCheckKxScalingGroupDestroy(ctx),
		Steps: []resource.TestStep{
			{
				Config: testAccKxScalingGroupConfig_basic(rName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckKxScalingGroupExists(ctx, resourceName, &KxScalingGroup),
					acctest.CheckResourceDisappears(ctx, acctest.Provider, tffinspace.ResourceKxScalingGroup(), resourceName),
				),
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

func testAccCheckKxScalingGroupDestroy(ctx context.Context) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		conn := acctest.Provider.Meta().(*conns.AWSClient).FinSpaceClient(ctx)

		for _, rs := range s.RootModule().Resources {
			if rs.Type != "aws_finspace_kx_scaling_group" {
				continue
			}

			input := &finspace.GetKxScalingGroupInput{
				ScalingGroupName: aws.String(rs.Primary.Attributes["name"]),
				EnvironmentId:    aws.String(rs.Primary.Attributes["environment_id"]),
			}
			_, err := conn.GetKxScalingGroup(ctx, input)
			if err != nil {
				var nfe *types.ResourceNotFoundException
				if errors.As(err, &nfe) {
					return nil
				}
				return err
			}

			return create.Error(names.FinSpace, create.ErrActionCheckingDestroyed, tffinspace.ResNameKxScalingGroup, rs.Primary.ID, errors.New("not destroyed"))
		}

		return nil
	}
}

func testAccCheckKxScalingGroupExists(ctx context.Context, name string, KxScalingGroup *finspace.GetKxScalingGroupOutput) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[name]
		if !ok {
			return create.Error(names.FinSpace, create.ErrActionCheckingExistence, tffinspace.ResNameKxScalingGroup, name, errors.New("not found"))
		}

		if rs.Primary.ID == "" {
			return create.Error(names.FinSpace, create.ErrActionCheckingExistence, tffinspace.ResNameKxScalingGroup, name, errors.New("not set"))
		}

		conn := acctest.Provider.Meta().(*conns.AWSClient).FinSpaceClient(ctx)
		resp, err := conn.GetKxScalingGroup(ctx, &finspace.GetKxScalingGroupInput{
			ScalingGroupName: aws.String(rs.Primary.Attributes["name"]),
			EnvironmentId:    aws.String(rs.Primary.Attributes["environment_id"]),
		})

		if err != nil {
			return create.Error(names.FinSpace, create.ErrActionCheckingExistence, tffinspace.ResNameKxScalingGroup, rs.Primary.ID, err)
		}

		*KxScalingGroup = *resp

		return nil
	}
}

func testAccKxScalingGroupConfigBase(rName string) string {
	return fmt.Sprintf(`
	data "aws_caller_identity" "current" {}
	data "aws_partition" "current" {}

	output "account_id" {
  		value = data.aws_caller_identity.current.account_id
	}

	resource "aws_kms_key" "test" {
  		deletion_window_in_days = 7
	}

	resource "aws_finspace_kx_environment" "test" {
  		name       = %[1]q
  		kms_key_id = aws_kms_key.test.arn
	}

	data "aws_iam_policy_document" "key_policy" {
		statement {
		  actions = [
			"kms:Decrypt",
			"kms:GenerateDataKey"
		  ]
	  
		  resources = [
			aws_kms_key.test.arn,
		  ]
	  
		  principals {
			type        = "Service"
			identifiers = ["finspace.amazonaws.com"]
		  }
	  
		  condition {
			test     = "ArnLike"
			variable = "aws:SourceArn"
			values   = ["${aws_finspace_kx_environment.test.arn}/*"]
		  }
	  
		  condition {
			test     = "StringEquals"
			variable = "aws:SourceAccount"
			values   = [data.aws_caller_identity.current.account_id]
		  }
		}
	  
		statement {
		  actions = [
			"kms:*",
		  ]
	  
		  resources = [
			"*",
		  ]
	  
		  principals {
			type        = "AWS"
			identifiers = ["arn:${data.aws_partition.current.partition}:iam::${data.aws_caller_identity.current.account_id}:root"]
		  }
		}
	}
	  
	resource "aws_kms_key_policy" "test" {
		key_id = aws_kms_key.test.id
  		policy = data.aws_iam_policy_document.key_policy.json
	}

	resource "aws_vpc" "test" {
  		cidr_block           = "172.31.0.0/16"
  		enable_dns_hostnames = true
	}

	resource "aws_subnet" "test" {
  		vpc_id               = aws_vpc.test.id
  		cidr_block           = "172.31.32.0/20"
  		availability_zone_id = aws_finspace_kx_environment.test.availability_zones[0]
	}

	resource "aws_security_group" "test" {
  		name   = %[1]q
  		vpc_id = aws_vpc.test.id

  		ingress {
    		from_port   = 0
   	 		to_port     = 0
    		protocol    = "-1"
    		cidr_blocks = ["0.0.0.0/0"]
  		}

  		egress {
    		from_port   = 0
    		to_port     = 0
    		protocol    = "-1"
    		cidr_blocks = ["0.0.0.0/0"]
  		}
	}

	resource "aws_internet_gateway" "test" {
		vpc_id = aws_vpc.test.id
	}

	data "aws_route_tables" "rts" {
  		vpc_id = aws_vpc.test.id
	}

	resource "aws_route" "r" {
  		route_table_id         = tolist(data.aws_route_tables.rts.ids)[0]
  		destination_cidr_block = "0.0.0.0/0"
  		gateway_id             = aws_internet_gateway.test.id
	}
	`, rName)
}

func testAccKxScalingGroupConfig_basic(rName string) string {
	return acctest.ConfigCompose(
		testAccKxScalingGroupConfigBase(rName),
		fmt.Sprintf(`
		resource "aws_finspace_kx_scaling_group" "test" {
			name                 = %[1]q
			environment_id       = aws_finspace_kx_environment.test.id
			availability_zone_id = aws_finspace_kx_environment.test.availability_zones[0]
			host_type            = "kx.sg.4xlarge"
		}
		`, rName))
}
