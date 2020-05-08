package main

import (
	"fmt"
	"net/url"

	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/iam"
)

func getAvailableAWSPermissions() ([]iamPolicy, error) {
	svc := iam.New(session.New())
	input := &iam.GetUserInput{}

	result, err := svc.GetUser(input)
	if err != nil {
		return nil, err
	}

	userName := result.User.UserName

	var policies [][]string

	userPolicies, err := svc.ListUserPolicies(&iam.ListUserPoliciesInput{
		UserName: userName,
	})
	if err != nil {
		return nil, err
	}

	for _, policyName := range userPolicies.PolicyNames {
		// Now that we know the user, we need to look for what inline policies
		// they may have and what groups they're a member of.
		policyResult, err := svc.GetUserPolicy(&iam.GetUserPolicyInput{
			UserName:   userName,
			PolicyName: policyName,
		})
		if err != nil {
			return nil, err
		}

		if policyResult.PolicyDocument != nil {
			policies = append(policies, []string{
				*policyResult.PolicyName,
				*policyResult.PolicyDocument,
			})
		}
	}

	// now groups, ignore paging for now
	groupResult, err := svc.ListGroupsForUser(&iam.ListGroupsForUserInput{
		UserName: userName,
	})
	if err != nil {
		return nil, err
	}

	for _, group := range groupResult.Groups {
		fmt.Println("retrieving policies for group:", *group.GroupName)
		groupPolicyResult, err := svc.ListGroupPolicies(&iam.ListGroupPoliciesInput{
			GroupName: group.GroupName,
		})
		if err != nil {
			return nil, err
		}

		// TODO: more pagination to deal with in the list group
		// policies endpoint
		for _, policyName := range groupPolicyResult.PolicyNames {
			policyResult, err := svc.GetGroupPolicy(&iam.GetGroupPolicyInput{
				GroupName:  group.GroupName,
				PolicyName: policyName,
			})
			if err != nil {
				return nil, err
			}
			policies = append(policies,
				[]string{
					*policyResult.PolicyName,
					*policyResult.PolicyDocument,
				},
			)
		}

		// Now let's get managed policies.
		attachedGroupPolicyResult, err := svc.ListAttachedGroupPolicies(
			&iam.ListAttachedGroupPoliciesInput{
				GroupName: group.GroupName,
			},
		)
		if err != nil {
			return nil, err
		}

		// TODO: more pagination to deal with in the list group
		// policies endpoint
		for _, policy := range attachedGroupPolicyResult.AttachedPolicies {
			policyResult, err := svc.GetPolicy(&iam.GetPolicyInput{
				PolicyArn: policy.PolicyArn,
			})
			if err != nil {
				return nil, err
			}

			policyVersion, err := svc.GetPolicyVersion(&iam.GetPolicyVersionInput{
				PolicyArn: policyResult.Policy.Arn,
				VersionId: policyResult.Policy.DefaultVersionId,
			})

			policies = append(policies,
				[]string{
					*policyResult.Policy.PolicyName,
					*policyVersion.PolicyVersion.Document,
				},
			)
		}
	}

	var iamPolicies = make([]iamPolicy, len(policies))

	for i, policy := range policies {
		policyDoc, _ := url.PathUnescape(policy[1])
		iamPolicies[i] = iamPolicy{
			name:     policy[0],
			document: policyDoc,
		}
	}

	return iamPolicies, nil
}

type iamPolicy struct {
	name     string
	document string
}
