package aws

import (
	"github.com/galaxy-future/BridgX/internal/logs"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"

	"github.com/galaxy-future/BridgX/pkg/cloud"
)

//CreateVPC output missing field: RequestId
func (p *AwsCloud) CreateVPC(req cloud.CreateVpcRequest) (cloud.CreateVpcResponse, error) {
	input := &ec2.CreateVpcInput{
		CidrBlock: aws.String(req.CidrBlock),
		TagSpecifications: append([]*ec2.TagSpecification{}, &ec2.TagSpecification{
			ResourceType: aws.String(_resourceTypeVpc),
			Tags: append([]*ec2.Tag{}, &ec2.Tag{
				Key:   aws.String(_tagKeyVpcName),
				Value: aws.String(req.VpcName),
			}),
		}),
	}
	output, err := p.ec2Client.CreateVpc(input)
	if err != nil {
		logs.Logger.Errorf("CreateVPC AwsCloud failed.err:[%v] req:[%v]", err, req)
		return cloud.CreateVpcResponse{}, err
	}
	return cloud.CreateVpcResponse{VpcId: aws.StringValue(output.Vpc.VpcId)}, nil
}

// GetVPC output missing field: SwitchIds、CreateAt
func (p *AwsCloud) GetVPC(req cloud.GetVpcRequest) (cloud.GetVpcResponse, error) {
	//The parameter VpcIds cannot be used with the parameter MaxResults
	pageSize := _pageSize * 10
	var awsVpcs = make([]*ec2.Vpc, 0, pageSize)
	input := &ec2.DescribeVpcsInput{
		//VpcIds: []*string{aws.String(req.VpcId)},
		MaxResults: aws.Int64(int64(pageSize)),
	}
	if req.VpcId != "" {
		input.Filters = append([]*ec2.Filter{}, &ec2.Filter{
			Name:   aws.String(_filterNameVpcId),
			Values: []*string{aws.String(req.VpcId)},
		})
	} else if req.VpcName != "" {
		input.Filters = append([]*ec2.Filter{}, &ec2.Filter{
			Name:   aws.String("tag:vpc-name"),
			Values: []*string{aws.String(req.VpcName)},
		})
	} else {
		return cloud.GetVpcResponse{}, _errInvalidParameter
	}
	err := p.ec2Client.DescribeVpcsPages(input, func(output *ec2.DescribeVpcsOutput, b bool) bool {
		awsVpcs = append(awsVpcs, output.Vpcs...)
		return output.NextToken != nil
	})
	if err != nil {
		logs.Logger.Errorf("GetVPC AwsCloud failed.err:[%v] req:[%v]", err, req)
		return cloud.GetVpcResponse{}, err
	}
	if len(awsVpcs) == 0 {
		logs.Logger.Errorf("GetVPC AwsCloud failed. req:[%v] len(awsVpcs) is zero", req)
		return cloud.GetVpcResponse{}, nil
	}
	awsVpc := awsVpcs[0]
	vpc := buildVpc(req.RegionId, awsVpc)
	return cloud.GetVpcResponse{Vpc: vpc}, nil
}

// DescribeVpcs output missing field: SwitchIds、CreateAt
func (p *AwsCloud) DescribeVpcs(req cloud.DescribeVpcsRequest) (cloud.DescribeVpcsResponse, error) {
	pageSize := _pageSize * 10
	var awsVpcs = make([]*ec2.Vpc, 0, pageSize)
	input := &ec2.DescribeVpcsInput{
		MaxResults: aws.Int64(int64(pageSize)),
	}
	err := p.ec2Client.DescribeVpcsPages(input, func(output *ec2.DescribeVpcsOutput, b bool) bool {
		awsVpcs = append(awsVpcs, output.Vpcs...)
		return output.NextToken != nil
	})
	if err != nil {
		logs.Logger.Errorf("DescribeVpcs AwsCloud failed.err:[%v] req:[%v]", err, req)
		return cloud.DescribeVpcsResponse{}, err
	}
	var vpcs = make([]cloud.VPC, 0, len(awsVpcs))
	for _, awsVpc := range awsVpcs {
		vpcs = append(vpcs, buildVpc(req.RegionId, awsVpc))
	}
	return cloud.DescribeVpcsResponse{Vpcs: vpcs}, nil
}

func buildVpc(regionId string, awsVpc *ec2.Vpc) cloud.VPC {
	var vpcName string
	for _, tag := range awsVpc.Tags {
		if aws.StringValue(tag.Key) == _tagKeyVpcName {
			vpcName = aws.StringValue(tag.Value)
		}
	}
	return cloud.VPC{
		VpcId:     aws.StringValue(awsVpc.VpcId),
		VpcName:   vpcName,
		CidrBlock: aws.StringValue(awsVpc.CidrBlock),
		//SwitchIds:
		RegionId: regionId,
		Status:   _vpcStatus[aws.StringValue(awsVpc.State)],
		//CreateAt:
	}
}

// CreateSwitch req:GatewayIp isn't use
func (p *AwsCloud) CreateSwitch(req cloud.CreateSwitchRequest) (cloud.CreateSwitchResponse, error) {
	input := &ec2.CreateSubnetInput{
		AvailabilityZoneId: aws.String(req.ZoneId),
		CidrBlock:          aws.String(req.CidrBlock),
		VpcId:              aws.String(req.VpcId),
		TagSpecifications: append([]*ec2.TagSpecification{}, &ec2.TagSpecification{
			ResourceType: aws.String(_resourceTypeSubnet),
			Tags: append([]*ec2.Tag{}, &ec2.Tag{
				Key:   aws.String(_tagKeyswitchname),
				Value: aws.String(req.VSwitchName),
			}),
		}),
	}
	output, err := p.ec2Client.CreateSubnet(input)
	if err != nil {
		logs.Logger.Errorf("CreateSwitch AwsCloud failed. err:[%v] req:[%v]", err, req)
		return cloud.CreateSwitchResponse{}, err
	}
	if output == nil || output.Subnet == nil {
		logs.Logger.Warnf("CreateSwitch AwsCloud failed. req:[%v] output:[%v]", err, req)
		return cloud.CreateSwitchResponse{}, err
	}
	//inputModify := &ec2.ModifySubnetAttributeInput{
	//	MapPublicIpOnLaunch: &ec2.AttributeBooleanValue{
	//		Value: aws.Bool(true),
	//	},
	//	SubnetId: output.Subnet.SubnetId,
	//}
	//_, err = p.ec2Client.ModifySubnetAttribute(inputModify)
	//if err != nil {
	//	logs.Logger.Errorf("ModifySubnetAttribute AwsCloud failed. err:[%v] req:[%v]", err, req)
	//	return cloud.CreateSwitchResponse{}, err
	//}
	return cloud.CreateSwitchResponse{SwitchId: aws.StringValue(output.Subnet.SubnetId)}, nil
}

// GetSwitch output missing field: CreateAt、GatewayIp
func (p *AwsCloud) GetSwitch(req cloud.GetSwitchRequest) (cloud.GetSwitchResponse, error) {
	input := &ec2.DescribeSubnetsInput{
		SubnetIds: append([]*string{}, &req.SwitchId),
	}
	output, err := p.ec2Client.DescribeSubnets(input)
	if err != nil {
		logs.Logger.Errorf("GetSwitch AwsCloud failed.err:[%v] req:[%v]", err, req)
		return cloud.GetSwitchResponse{}, err
	}
	if output == nil || len(output.Subnets) == 0 {
		logs.Logger.Errorf("GetSwitch AwsCloud failed. req:[%v] output:[%v]", req, output)
		return cloud.GetSwitchResponse{}, nil
	}
	awsSubnet := output.Subnets[0]
	subnet := buildSwitch(awsSubnet)
	return cloud.GetSwitchResponse{Switch: subnet}, nil
}

// DescribeSwitches output missing field: CreateAt、GatewayIp
func (p *AwsCloud) DescribeSwitches(req cloud.DescribeSwitchesRequest) (cloud.DescribeSwitchesResponse, error) {
	pageSize := _pageSize * 10
	var awsSubnets = make([]*ec2.Subnet, 0, pageSize)
	input := &ec2.DescribeSubnetsInput{
		Filters: []*ec2.Filter{
			{
				Name:   aws.String(_filterNameVpcId),
				Values: []*string{&req.VpcId},
			},
		},
		MaxResults: aws.Int64(int64(pageSize)),
	}
	err := p.ec2Client.DescribeSubnetsPages(input, func(output *ec2.DescribeSubnetsOutput, b bool) bool {
		awsSubnets = append(awsSubnets, output.Subnets...)
		return output.NextToken != nil
	})
	if err != nil {
		logs.Logger.Errorf("DescribeSwitches AwsCloud failed.err: [%v] req[%v]", err, req)
		return cloud.DescribeSwitchesResponse{}, err
	}
	if len(awsSubnets) == 0 {
		logs.Logger.Errorf("DescribeSwitches AwsCloud failed. req[%v] len(awsSubnets) is zero", req)
		return cloud.DescribeSwitchesResponse{}, nil
	}
	var subnets = make([]cloud.Switch, 0, len(awsSubnets))
	for _, awsSubnet := range awsSubnets {
		subnets = append(subnets, buildSwitch(awsSubnet))
	}
	return cloud.DescribeSwitchesResponse{Switches: subnets}, nil
}

func buildSwitch(awsSubnet *ec2.Subnet) cloud.Switch {
	var switchName string
	for _, tag := range awsSubnet.Tags {
		if aws.StringValue(tag.Key) == _tagKeyswitchname {
			switchName = aws.StringValue(tag.Value)
		}
	}
	return cloud.Switch{
		VpcId:                   aws.StringValue(awsSubnet.VpcId),
		SwitchId:                aws.StringValue(awsSubnet.SubnetId),
		Name:                    switchName,
		IsDefault:               _subnetIsDefault[aws.BoolValue(awsSubnet.DefaultForAz)],
		AvailableIpAddressCount: int(aws.Int64Value(awsSubnet.AvailableIpAddressCount)),
		VStatus:                 _subnetStatus[aws.StringValue(awsSubnet.State)],
		ZoneId:                  aws.StringValue(awsSubnet.AvailabilityZoneId),
		CidrBlock:               aws.StringValue(awsSubnet.CidrBlock),
		//CreateAt:
		//GatewayIp:
	}
}
