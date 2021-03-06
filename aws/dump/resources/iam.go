package resources

import (
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/iam"
	"github.com/fatih/structs"
)

var (
	IAMService = Service{
		Name:     "iam",
		IsGlobal: true,
		Reports: map[string]Report{
			"users-and-access-keys":         IAMListUsersAndAccessKeys,
			"roles":                         IAMListRoles,
			"policies":                      IAMListPolicies,
			"groups":                        IAMListGroups,
			"instance-profiles":             IAMListInstanceProfiles,
			"account-authorization-details": IAMListAccountAuthorizationDetails,
		},
	}
)

type PolicyFetchFunc func(*Session, *iam.IAM, string, string) *ReportResult

func IAMListUserAttachedPolicies(session *Session, client *iam.IAM, userARN, userName string) *ReportResult {
	result := &ReportResult{}
	err := client.ListAttachedUserPoliciesPages(&iam.ListAttachedUserPoliciesInput{UserName: aws.String(userName)},
		func(page *iam.ListAttachedUserPoliciesOutput, lastPage bool) bool {
			for _, policy := range page.AttachedPolicies {
				r := Resource{
					ID:        fmt.Sprintf("%s_%s", userName, *policy.PolicyName),
					ARN:       "",
					AccountID: session.AccountID,
					Service:   "iam",
					Type:      "user-policy-attachment",
					Region:    *session.Config.Region,
					Metadata:  structs.Map(policy),
				}
				r.Metadata["UserArn"] = userARN
				result.Resources = append(result.Resources, r)
			}
			return true
		})
	result.Error = err
	return result
}

func IAMListUserPolicies(session *Session, client *iam.IAM, userARN, userName string) *ReportResult {
	result := &ReportResult{}
	err := client.ListUserPoliciesPages(&iam.ListUserPoliciesInput{UserName: aws.String(userName)},
		func(page *iam.ListUserPoliciesOutput, lastPage bool) bool {
			for _, policyName := range page.PolicyNames {

				policy, err := client.GetUserPolicy(&iam.GetUserPolicyInput{UserName: aws.String(userName), PolicyName: policyName})
				if err != nil {
					result.Error = err
					return false
				}

				r := Resource{
					ID:        fmt.Sprintf("%s_%s_inline", userName, *policy.PolicyName),
					ARN:       "",
					AccountID: session.AccountID,
					Service:   "iam",
					Type:      "user-policy-inline",
					Region:    *session.Config.Region,
					Metadata:  structs.Map(policy),
				}
				document, err := DecodeInlinePolicyDocument(*r.Metadata["PolicyDocument"].(*string))
				if err != nil {
					result.Error = err
					return false
				}
				r.Metadata["PolicyDocument"] = document
				r.Metadata["UserArn"] = userARN
				result.Resources = append(result.Resources, r)
			}
			return true
		})
	result.Error = err
	return result
}

func IAMListUsersAndAccessKeys(session *Session) *ReportResult {

	policiesFunctions := []PolicyFetchFunc{IAMListUserPolicies, IAMListUserAttachedPolicies}

	client := iam.New(session.Session, session.Config)
	accessKeys := []Resource{}
	arns := []*string{}
	result := &ReportResult{}
	result.Error = client.ListUsersPages(&iam.ListUsersInput{},
		func(page *iam.ListUsersOutput, lastPage bool) bool {
			for _, user := range page.Users {
				resource, err := NewResource(*user.Arn, user)
				if err != nil {
					result.Error = err
					return false
				}
				arns = append(arns, user.Arn)
				result.Resources = append(result.Resources, *resource)

				for _, fn := range policiesFunctions {
					policies := fn(session, client, *user.Arn, *user.UserName)
					if policies.Error != nil {
						result.Error = policies.Error
						return false
					}
					result.Resources = append(result.Resources, policies.Resources...)
				}

				keysResult := IAMListAccessKeys(session, client, *user.UserName)
				if keysResult.Error != nil {
					result.Error = keysResult.Error
					return false
				}
				accessKeys = append(accessKeys, keysResult.Resources...)
			}

			return true
		})

	if result.Error != nil {
		return result
	}

	jobIds, err := GenerateServiceLastAccessedDetails(client, arns)
	if err != nil {
		result.Error = err
		return result
	}
	AttachServiceLastAccessedDetails(client, result, jobIds)

	result.Resources = append(result.Resources, accessKeys...)
	return result
}

func IAMListGroupAttachedPolicies(session *Session, client *iam.IAM, groupARN, groupName string) *ReportResult {
	result := &ReportResult{}
	err := client.ListAttachedGroupPoliciesPages(&iam.ListAttachedGroupPoliciesInput{GroupName: aws.String(groupName)},
		func(page *iam.ListAttachedGroupPoliciesOutput, lastPage bool) bool {
			for _, policy := range page.AttachedPolicies {
				r := Resource{
					ID:        fmt.Sprintf("%s_%s", groupName, *policy.PolicyName),
					ARN:       "",
					AccountID: session.AccountID,
					Service:   "iam",
					Type:      "group-policy-attachment",
					Region:    *session.Config.Region,
					Metadata:  structs.Map(policy),
				}
				r.Metadata["GroupArn"] = groupARN
				result.Resources = append(result.Resources, r)
			}
			return true
		})
	result.Error = err
	return result
}

func IAMListGroupPolicies(session *Session, client *iam.IAM, groupARN, groupName string) *ReportResult {
	result := &ReportResult{}
	err := client.ListGroupPoliciesPages(&iam.ListGroupPoliciesInput{GroupName: aws.String(groupName)},
		func(page *iam.ListGroupPoliciesOutput, lastPage bool) bool {
			for _, policyName := range page.PolicyNames {

				policy, err := client.GetGroupPolicy(&iam.GetGroupPolicyInput{GroupName: aws.String(groupName), PolicyName: policyName})
				if err != nil {
					result.Error = err
					return false
				}

				r := Resource{
					ID:        fmt.Sprintf("%s_%s_inline", groupName, *policy.PolicyName),
					ARN:       "",
					AccountID: session.AccountID,
					Service:   "iam",
					Type:      "group-policy-inline",
					Region:    *session.Config.Region,
					Metadata:  structs.Map(policy),
				}
				document, err := DecodeInlinePolicyDocument(*r.Metadata["PolicyDocument"].(*string))
				if err != nil {
					result.Error = err
					return false
				}
				r.Metadata["PolicyDocument"] = document
				r.Metadata["GroupArn"] = groupARN
				result.Resources = append(result.Resources, r)
			}
			return true
		})
	result.Error = err
	return result
}

func IAMListGroups(session *Session) *ReportResult {

	policiesFunctions := []PolicyFetchFunc{IAMListGroupPolicies, IAMListGroupAttachedPolicies}

	client := iam.New(session.Session, session.Config)
	arns := []*string{}
	result := &ReportResult{}
	result.Error = client.ListGroupsPages(&iam.ListGroupsInput{},
		func(page *iam.ListGroupsOutput, lastPage bool) bool {
			for _, group := range page.Groups {

				resource, err := NewResource(*group.Arn, group)
				if err != nil {
					result.Error = err
					return false
				}
				arns = append(arns, group.Arn)
				result.Resources = append(result.Resources, *resource)

				for _, fn := range policiesFunctions {
					policies := fn(session, client, *group.Arn, *group.GroupName)
					if policies.Error != nil {
						result.Error = policies.Error
						return false
					}
					result.Resources = append(result.Resources, policies.Resources...)
				}
			}

			return true
		})

	if result.Error != nil {
		return result
	}

	jobIds, err := GenerateServiceLastAccessedDetails(client, arns)
	if err != nil {
		result.Error = err
		return result
	}
	AttachServiceLastAccessedDetails(client, result, jobIds)

	return result
}

func IAMListAccountAuthorizationDetails(session *Session) *ReportResult {
	client := iam.New(session.Session, session.Config)

	result := &ReportResult{}

	err := client.GetAccountAuthorizationDetailsPages(&iam.GetAccountAuthorizationDetailsInput{},
		func(page *iam.GetAccountAuthorizationDetailsOutput, lastPage bool) bool {

			for _, group := range page.GroupDetailList {
				resource := Resource{
					ID:        *group.GroupId,
					ARN:       *group.Arn,
					AccountID: session.AccountID,
					Service:   "iam",
					Type:      "account-authorization-details-group",
					Metadata:  structs.Map(group),
				}

				for _, policyI := range resource.Metadata["GroupPolicyList"].([]interface{}) {
					policy := policyI.(map[string]interface{})

					document, err := DecodeInlinePolicyDocument(*policy["PolicyDocument"].(*string))
					if err != nil {
						result.Error = err
						return false
					}
					policy["PolicyDocument"] = document
				}

				result.Resources = append(result.Resources, resource)
			}

			for _, user := range page.UserDetailList {
				resource := Resource{
					ID:        *user.UserId,
					ARN:       *user.Arn,
					AccountID: session.AccountID,
					Service:   "iam",
					Type:      "account-authorization-details-user",
					Metadata:  structs.Map(user),
				}

				for _, policyI := range resource.Metadata["UserPolicyList"].([]interface{}) {
					policy := policyI.(map[string]interface{})

					document, err := DecodeInlinePolicyDocument(*policy["PolicyDocument"].(*string))
					if err != nil {
						result.Error = err
						return false
					}
					policy["PolicyDocument"] = document
				}

				result.Resources = append(result.Resources, resource)
			}

			for _, role := range page.RoleDetailList {
				resource := Resource{
					ID:        *role.RoleId,
					ARN:       *role.Arn,
					AccountID: session.AccountID,
					Service:   "iam",
					Type:      "account-authorization-details-role",
					Metadata:  structs.Map(role),
				}

				document, err := DecodeInlinePolicyDocument(*resource.Metadata["AssumeRolePolicyDocument"].(*string))
				if err != nil {
					result.Error = err
					return false
				}
				resource.Metadata["AssumeRolePolicyDocument"] = document

				for _, instanceProfileI := range resource.Metadata["InstanceProfileList"].([]interface{}) {
					instanceProfile := instanceProfileI.(map[string]interface{})
					for _, roleI := range instanceProfile["Roles"].([]interface{}) {
						role := roleI.(map[string]interface{})
						document, err := DecodeInlinePolicyDocument(*role["AssumeRolePolicyDocument"].(*string))
						if err != nil {
							result.Error = err
							return false
						}
						role["AssumeRolePolicyDocument"] = document
					}
				}

				result.Resources = append(result.Resources, resource)
			}

			for _, policy := range page.Policies {
				resource := Resource{
					ID:        *policy.PolicyId,
					ARN:       *policy.Arn,
					AccountID: session.AccountID,
					Service:   "iam",
					Type:      "account-authorization-details-policy",
					Metadata:  structs.Map(policy),
				}

				for _, policyI := range resource.Metadata["PolicyVersionList"].([]interface{}) {
					policy := policyI.(map[string]interface{})

					document, err := DecodeInlinePolicyDocument(*policy["Document"].(*string))
					if err != nil {
						result.Error = err
						return false
					}
					policy["Document"] = document
				}

				result.Resources = append(result.Resources, resource)
			}

			return true
		})

	result.Error = err
	return result
}

func IAMListRoleAttachedPolicies(session *Session, client *iam.IAM, roleARN, roleName string) *ReportResult {
	result := &ReportResult{}
	err := client.ListAttachedRolePoliciesPages(&iam.ListAttachedRolePoliciesInput{RoleName: aws.String(roleName)},
		func(page *iam.ListAttachedRolePoliciesOutput, lastPage bool) bool {
			for _, policy := range page.AttachedPolicies {
				r := Resource{
					ID:        fmt.Sprintf("%s_%s", roleName, *policy.PolicyName),
					ARN:       "",
					AccountID: session.AccountID,
					Service:   "iam",
					Type:      "role-policy-attachment",
					Region:    *session.Config.Region,
					Metadata:  structs.Map(policy),
				}
				r.Metadata["RoleArn"] = roleARN
				result.Resources = append(result.Resources, r)
			}
			return true
		})
	result.Error = err
	return result
}

func IAMListRolePolicies(session *Session, client *iam.IAM, roleARN, roleName string) *ReportResult {
	result := &ReportResult{}
	err := client.ListRolePoliciesPages(&iam.ListRolePoliciesInput{RoleName: aws.String(roleName)},
		func(page *iam.ListRolePoliciesOutput, lastPage bool) bool {
			for _, policyName := range page.PolicyNames {

				policy, err := client.GetRolePolicy(&iam.GetRolePolicyInput{RoleName: aws.String(roleName), PolicyName: policyName})
				if err != nil {
					result.Error = err
					return false
				}

				r := Resource{
					ID:        fmt.Sprintf("%s_%s_inline", roleName, *policy.PolicyName),
					ARN:       "",
					AccountID: session.AccountID,
					Service:   "iam",
					Type:      "role-policy-inline",
					Region:    *session.Config.Region,
					Metadata:  structs.Map(policy),
				}
				document, err := DecodeInlinePolicyDocument(*r.Metadata["PolicyDocument"].(*string))
				if err != nil {
					result.Error = err
					return false
				}
				r.Metadata["PolicyDocument"] = document
				r.Metadata["RoleArn"] = roleARN
				result.Resources = append(result.Resources, r)
			}
			return true
		})
	result.Error = err
	return result
}

func IAMListRoles(session *Session) *ReportResult {

	policiesFunctions := []PolicyFetchFunc{IAMListRolePolicies, IAMListRoleAttachedPolicies}

	client := iam.New(session.Session, session.Config)
	arns := []*string{}
	result := &ReportResult{}
	result.Error = client.ListRolesPages(&iam.ListRolesInput{},
		func(page *iam.ListRolesOutput, lastPage bool) bool {
			for _, role := range page.Roles {
				resource, err := NewResource(*role.Arn, role)
				if err != nil {
					result.Error = err
					return false
				}

				document, err := DecodeInlinePolicyDocument(*resource.Metadata["AssumeRolePolicyDocument"].(*string))
				if err != nil {
					result.Error = err
					return false
				}
				resource.Metadata["AssumeRolePolicyDocument"] = document

				resource.ID = *role.RoleId
				arns = append(arns, role.Arn)
				result.Resources = append(result.Resources, *resource)

				policies := IAMListRolePolicies(session, client, *role.Arn, *role.RoleName)
				if policies.Error != nil {
					result.Error = policies.Error
					return false
				}
				result.Resources = append(result.Resources, policies.Resources...)

				for _, fn := range policiesFunctions {
					policies := fn(session, client, *role.Arn, *role.RoleName)
					if policies.Error != nil {
						result.Error = policies.Error
						return false
					}
					result.Resources = append(result.Resources, policies.Resources...)
				}
			}

			return true
		})

	if result.Error != nil {
		return result
	}

	jobIds, err := GenerateServiceLastAccessedDetails(client, arns)
	if err != nil {
		result.Error = err
		return result
	}
	AttachServiceLastAccessedDetails(client, result, jobIds)

	return result
}

func IAMListPolicyVersions(session *Session, client *iam.IAM, policyArn string) *ReportResult {
	result := &ReportResult{}
	err := client.ListPolicyVersionsPages(&iam.ListPolicyVersionsInput{PolicyArn: aws.String(policyArn)},
		func(page *iam.ListPolicyVersionsOutput, lastPage bool) bool {
			for _, resource := range page.Versions {

				policyVersion, err := client.GetPolicyVersion(&iam.GetPolicyVersionInput{PolicyArn: aws.String(policyArn), VersionId: resource.VersionId})
				if err != nil {
					result.Error = err
					return false
				}

				document, err := DecodeInlinePolicyDocument(*policyVersion.PolicyVersion.Document)
				if err != nil {
					result.Error = err
					return false
				}

				metadata := structs.Map(policyVersion.PolicyVersion)
				metadata["Document"] = document

				arn := fmt.Sprintf("%s:%s", policyArn, *resource.VersionId)
				r := Resource{
					ID:        arn,
					ARN:       arn,
					AccountID: session.AccountID,
					Service:   "iam",
					Type:      "policy-version",
					Region:    *session.Config.Region,
					Metadata:  metadata,
				}
				result.Resources = append(result.Resources, r)
			}
			return true
		})

	if result.Error != nil {
		return result
	}

	result.Error = err
	return result
}

func IAMListPolicies(session *Session) *ReportResult {
	client := iam.New(session.Session, session.Config)
	arns := []*string{}
	result := &ReportResult{}
	result.Error = client.ListPoliciesPages(&iam.ListPoliciesInput{Scope: aws.String("Local")},
		func(page *iam.ListPoliciesOutput, lastPage bool) bool {
			for _, policy := range page.Policies {
				resource, err := NewResource(*policy.Arn, policy)
				if err != nil {
					result.Error = err
					return false
				}

				arns = append(arns, policy.Arn)

				policyVersions := IAMListPolicyVersions(session, client, *policy.Arn)
				if policyVersions.Error != nil {
					result.Error = policyVersions.Error
					return false
				}

				result.Resources = append(result.Resources, *resource)
				result.Resources = append(result.Resources, policyVersions.Resources...)
			}

			return true
		})

	if result.Error != nil {
		return result
	}

	jobIds, err := GenerateServiceLastAccessedDetails(client, arns)
	if err != nil {
		result.Error = err
		return result
	}
	AttachServiceLastAccessedDetails(client, result, jobIds)
	return result
}

func IAMListAccessKeys(session *Session, client *iam.IAM, username string) *ReportResult {
	result := &ReportResult{}
	result.Error = client.ListAccessKeysPages(&iam.ListAccessKeysInput{
		UserName: aws.String(username),
	},
		func(page *iam.ListAccessKeysOutput, lastPage bool) bool {
			for _, accessKey := range page.AccessKeyMetadata {
				resource := Resource{
					ID:        *accessKey.AccessKeyId,
					AccountID: session.AccountID,
					Service:   "iam",
					Type:      "access-key",
					Metadata:  structs.Map(accessKey),
				}

				lastUsed, err := client.GetAccessKeyLastUsed(&iam.GetAccessKeyLastUsedInput{AccessKeyId: accessKey.AccessKeyId})
				if err != nil {
					result.Error = err
					return false
				}
				resource.Metadata["AccessKeyLastUsed"] = structs.Map(lastUsed.AccessKeyLastUsed)
				resource.Metadata["LastUsed"] = lastUsed.AccessKeyLastUsed.LastUsedDate
				result.Resources = append(result.Resources, resource)
			}

			return true
		})

	return result
}

func GenerateServiceLastAccessedDetails(client *iam.IAM, arns []*string) ([]*string, error) {
	jobIds := []*string{}
	for _, arn := range arns {
		job, err := client.GenerateServiceLastAccessedDetails(&iam.GenerateServiceLastAccessedDetailsInput{
			Arn: arn,
		})
		if err != nil {
			return nil, err
		}
		jobIds = append(jobIds, job.JobId)
	}
	return jobIds, nil
}

func AttachServiceLastAccessedDetails(client *iam.IAM, result *ReportResult, jobIds []*string) {
	for i := 0; i < len(jobIds); {
		jobId := jobIds[i]
		lastUsed, err := client.GetServiceLastAccessedDetails(&iam.GetServiceLastAccessedDetailsInput{JobId: jobId})
		if err != nil {
			result.Error = err
			return
		}
		if *lastUsed.JobStatus == "IN_PROGRESS" {
			time.Sleep(1 * time.Second)
			continue
		}
		if *lastUsed.JobStatus == "COMPLETED" {
			result.Resources[i].Metadata["ServiceLastAccessed"] = lastUsed.ServicesLastAccessed
			var lastUsedAt *time.Time
			for _, serviceLastAccessed := range lastUsed.ServicesLastAccessed {
				if serviceLastAccessed.LastAuthenticated == nil {
					continue
				}
				if lastUsedAt == nil || serviceLastAccessed.LastAuthenticated.After(*lastUsedAt) {
					lastUsedAt = serviceLastAccessed.LastAuthenticated
				}
			}
			result.Resources[i].Metadata["LastUsed"] = lastUsedAt

		}
		i++
	}
}

func IAMListInstanceProfiles(session *Session) *ReportResult {

	client := iam.New(session.Session, session.Config)

	result := &ReportResult{}
	err := client.ListInstanceProfilesPages(&iam.ListInstanceProfilesInput{},
		func(page *iam.ListInstanceProfilesOutput, lastPage bool) bool {
			for _, instanceProfile := range page.InstanceProfiles {
				resource := Resource{
					ID:        *instanceProfile.InstanceProfileId,
					ARN:       *instanceProfile.Arn,
					AccountID: session.AccountID,
					Service:   "iam",
					Type:      "instance-profile",
					Region:    *session.Config.Region,
					Metadata:  structs.Map(instanceProfile),
				}

				roles := resource.Metadata["Roles"].([]interface{})
				for _, irole := range roles {
					role := irole.(map[string]interface{})
					document, err := DecodeInlinePolicyDocument(*role["AssumeRolePolicyDocument"].(*string))
					if err != nil {
						result.Error = err
						return false
					}
					role["AssumeRolePolicyDocument"] = document
				}

				result.Resources = append(result.Resources, resource)
			}

			return true
		})

	if result.Error != nil {
		return result
	}
	result.Error = err
	return result
}
