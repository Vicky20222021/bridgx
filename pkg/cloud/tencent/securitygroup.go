package tencent

import (
	"fmt"

	"github.com/galaxy-future/BridgX/internal/logs"
	"github.com/galaxy-future/BridgX/pkg/cloud"
	"github.com/spf13/cast"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common"
	vpc "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/vpc/v20170312"
)

func (p *TencentCloud) CreateSecurityGroup(req cloud.CreateSecurityGroupRequest) (cloud.CreateSecurityGroupResponse, error) {
	request := vpc.NewCreateSecurityGroupRequest()
	request.GroupName = common.StringPtr(req.SecurityGroupName)
	request.GroupDescription = common.StringPtr(req.RegionId)
	// The tags are a filter for DescribeSecurityGroups`
	request.Tags = []*vpc.Tag{
		{
			Key:   common.StringPtr("VpcId"),
			Value: common.StringPtr(req.VpcId),
		},
		{
			Key:   common.StringPtr("SecurityGroupType"),
			Value: common.StringPtr(req.SecurityGroupType),
		},
	}
	response, err := p.vpcClient.CreateSecurityGroup(request)
	if err != nil {
		logs.Logger.Errorf("CreateSecurityGroup TencentCloud failed.err: [%v], req[%v]", err, req)
		return cloud.CreateSecurityGroupResponse{}, err
	}
	if response != nil && response.Response != nil {
		return cloud.CreateSecurityGroupResponse{
			SecurityGroupId: *response.Response.SecurityGroup.SecurityGroupId,
			RequestId:       *response.Response.RequestId,
		}, nil
	}
	return cloud.CreateSecurityGroupResponse{}, err
}

// AddIngressSecurityGroupRule 入参各云得统一
func (p *TencentCloud) AddIngressSecurityGroupRule(req cloud.AddSecurityGroupRuleRequest) error {
	request := vpc.NewCreateSecurityGroupPoliciesRequest()
	securityGroupId := common.StringPtr(req.SecurityGroupId)
	request.SecurityGroupId = securityGroupId
	request.SecurityGroupPolicySet = &vpc.SecurityGroupPolicySet{
		Ingress: []*vpc.SecurityGroupPolicy{
			{
				Protocol:          common.StringPtr(_protocol[req.IpProtocol]),
				Action:            common.StringPtr("ACCEPT"),
				PolicyDescription: common.StringPtr(req.VpcId),
			},
		},
	}
	if (req.IpProtocol == cloud.ProtocolTcp || req.IpProtocol == cloud.ProtocolUdp) && req.PortFrom > 0 {
		portRange := cast.ToString(req.PortFrom)
		if req.PortFrom != req.PortTo {
			portRange = fmt.Sprintf("%d-%d", req.PortFrom, req.PortTo)
		}
		request.SecurityGroupPolicySet.Ingress[0].Port = &portRange
	}
	if req.CidrIp != "" {
		request.SecurityGroupPolicySet.Ingress[0].CidrBlock = common.StringPtr(req.CidrIp)
	}
	if req.GroupId != "" {
		request.SecurityGroupPolicySet.Ingress[0].SecurityGroupId = common.StringPtr(req.GroupId)
	}

	_, err := p.vpcClient.CreateSecurityGroupPolicies(request)
	if err != nil {
		logs.Logger.Errorf("AddIngressSecurityGroupRule AlibabaCloud failed.err: [%v], req[%v]", err, req)
		return err
	}
	return nil
}

func (p *TencentCloud) AddEgressSecurityGroupRule(req cloud.AddSecurityGroupRuleRequest) error {
	request := vpc.NewCreateSecurityGroupPoliciesRequest()
	securityGroupId := common.StringPtr(req.SecurityGroupId)
	request.SecurityGroupId = securityGroupId
	request.SecurityGroupPolicySet = &vpc.SecurityGroupPolicySet{
		Egress: []*vpc.SecurityGroupPolicy{
			{
				Protocol:          common.StringPtr(_protocol[req.IpProtocol]),
				Action:            common.StringPtr("ACCEPT"),
				PolicyDescription: common.StringPtr(req.VpcId),
			},
		},
	}
	if (req.IpProtocol == cloud.ProtocolTcp || req.IpProtocol == cloud.ProtocolUdp) && req.PortFrom > 0 {
		portRange := cast.ToString(req.PortFrom)
		if req.PortFrom != req.PortTo {
			portRange = fmt.Sprintf("%d-%d", req.PortFrom, req.PortTo)
		}
		request.SecurityGroupPolicySet.Egress[0].Port = common.StringPtr(portRange)
	}
	if req.CidrIp != "" {
		request.SecurityGroupPolicySet.Egress[0].CidrBlock = common.StringPtr(req.CidrIp)
	}
	if req.GroupId != "" {
		request.SecurityGroupPolicySet.Egress[0].SecurityGroupId = common.StringPtr(req.GroupId)
	}

	_, err := p.vpcClient.CreateSecurityGroupPolicies(request)
	if err != nil {
		logs.Logger.Errorf("AddEgressSecurityGroupRule AlibabaCloud failed.err: [%v], req[%v]", err, req)
		return err
	}
	return nil
}

func (p *TencentCloud) DescribeSecurityGroups(req cloud.DescribeSecurityGroupsRequest) (cloud.DescribeSecurityGroupsResponse, error) {
	var page int32 = 1
	var pageSize int32 = 100
	groups := make([]cloud.SecurityGroup, 0, 128)

	request := vpc.NewDescribeSecurityGroupsRequest()
	request.Filters = []*vpc.Filter{
		{
			Name:   common.StringPtr("tag:VpcId"),
			Values: common.StringPtrs([]string{req.VpcId}),
		},
	}
	request.Limit = common.StringPtr(cast.ToString(pageSize))
	for {
		request.Offset = common.StringPtr(cast.ToString((page - 1) * pageSize))
		response, err := p.vpcClient.DescribeSecurityGroups(request)
		if err != nil {
			return cloud.DescribeSecurityGroupsResponse{}, err
		}
		if response != nil && response.Response != nil && response.Response.SecurityGroupSet != nil {
			for _, group := range response.Response.SecurityGroupSet {
				var vpcId, SecurityGroupType *string
				for _, tag := range group.TagSet {
					if *tag.Key == "VpcId" {
						vpcId = tag.Value
					} else if *tag.Key == "SecurityGroupType" {
						SecurityGroupType = tag.Value
					}
				}
				groups = append(groups, cloud.SecurityGroup{
					SecurityGroupId:   *group.SecurityGroupId,
					SecurityGroupType: *SecurityGroupType,
					SecurityGroupName: *group.SecurityGroupName,
					CreateAt:          *group.CreatedTime,
					VpcId:             *vpcId,
					RegionId:          req.RegionId,
				})
			}
			if *response.Response.TotalCount > uint64(page*pageSize) {
				page++
			} else {
				break
			}
		} else {
			return cloud.DescribeSecurityGroupsResponse{}, fmt.Errorf("response is nil")
		}
	}
	return cloud.DescribeSecurityGroupsResponse{Groups: groups}, nil
}

func (p *TencentCloud) DescribeGroupRules(req cloud.DescribeGroupRulesRequest) (cloud.DescribeGroupRulesResponse, error) {
	rules := make([]cloud.SecurityGroupRule, 0, 128)
	request := vpc.NewDescribeSecurityGroupPoliciesRequest()
	request.SecurityGroupId = common.StringPtr(req.SecurityGroupId)
	response, err := p.vpcClient.DescribeSecurityGroupPolicies(request)
	if err != nil {
		logs.Logger.Errorf("DescribeGroupRules AlibabaCloud failed.err: [%v], req[%v]", err, req)
		return cloud.DescribeGroupRulesResponse{}, err
	}
	if response != nil && response.Response != nil && response.Response.SecurityGroupPolicySet != nil {
		policySet := response.Response.SecurityGroupPolicySet
		egress := policySet.Egress
		if egress != nil {
			for _, policy := range egress {
				rules = append(rules, cloud.SecurityGroupRule{
					VpcId:           *policy.PolicyDescription,
					SecurityGroupId: *policy.SecurityGroupId,
					PortRange:       *policy.Port,
					Protocol:        *policy.Protocol,
					Direction:       cloud.SecGroupRuleOut,
					GroupId:         "",
					CidrIp:          *policy.CidrBlock,
					PrefixListId:    "",
				})
			}
		}
		ingress := policySet.Ingress
		if ingress != nil {
			for _, policy := range ingress {
				rules = append(rules, cloud.SecurityGroupRule{
					VpcId:           *policy.PolicyDescription,
					SecurityGroupId: *policy.SecurityGroupId,
					PortRange:       *policy.Port,
					Protocol:        *policy.Protocol,
					Direction:       cloud.SecGroupRuleIn,
					GroupId:         "",
					CidrIp:          *policy.CidrBlock,
					PrefixListId:    "",
				})
			}
		}
	}
	return cloud.DescribeGroupRulesResponse{Rules: rules}, nil
}
