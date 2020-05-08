package main

import "github.com/aws/aws-sdk-go/aws"

func awsActionsFromPermissions(s tfsources) []*string {
	var actions []*string
	// Get all actions for `resource`s.
	//
	// Right now we only support the top-level of this, but later we'll
	// support true resource level constraints.
	for resourceType, _ := range s.resources {
		permissions, ok := actionMapping[resourceType]
		if !ok {
			continue
		}

		for _, permissionSet := range [][]string{permissions.read, permissions.write} {
			for _, action := range permissionSet {
				actions = append(actions, aws.String(action))
			}
		}
	}

	// Get read-only actions for `data` sources.
	//
	// Right now we only support the top-level of this, but later we'll
	// support true resource level constraints.
	for dataType, _ := range s.data {
		permissions, ok := actionMapping[dataType]
		if !ok {
			continue
		}

		for _, action := range permissions.read {
			actions = append(actions, aws.String(action))
		}
	}

	return actions
}

type permissions struct {
	read  []string
	write []string
}

var (
	actionMapping = map[string]permissions{
		"aws_lb": permissions{
			[]string{
				"elbv2:DescribeLoadBalancers",
			},
			[]string{
				"elbv2:CreateLoadBalancer",
			},
		},
		"aws_alb": permissions{
			[]string{
				"elbv2:DescribeLoadBalancers",
			},
			[]string{
				"elbv2:CreateLoadBalancer",
			},
		},
		"aws_ssm_parameter": permissions{
			read: []string{
				"ssm:GetParameter",
				"ssm:DescribeParameters",
			},
			write: []string{
				"ssm:DeleteParameter",
				"ssm:PutParameter",
			},
		},
	}
)
