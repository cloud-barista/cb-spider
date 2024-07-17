package resources

import (
	"errors"
	"fmt"
	"reflect"
	"strconv"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/autoscaling"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/eks"
	"github.com/aws/aws-sdk-go/service/iam"
	call "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/call-log"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
)

type AwsClusterHandler struct {
	Region      idrv.RegionInfo
	Client      *eks.EKS
	EC2Client   *ec2.EC2
	Iam         *iam.IAM
	AutoScaling *autoscaling.AutoScaling
}

const (
	NODEGROUP_TAG string = "nodegroup"
)

//------ Cluster Management

/*
	AWS Cluster는 Role이 필수임.
	우선, roleName=spider-eks-role로 설정, 생성 시 Role의 ARN을 조회하여 사용
*/

// ------ Cluster Management
func (ClusterHandler *AwsClusterHandler) CreateCluster(clusterReqInfo irs.ClusterInfo) (irs.ClusterInfo, error) {
	// validation check

	reqSecurityGroupIds := clusterReqInfo.Network.SecurityGroupIIDs
	var securityGroupIds []*string
	for _, securityGroupIID := range reqSecurityGroupIds {
		securityGroupIds = append(securityGroupIds, aws.String(securityGroupIID.SystemId))
	}

	reqSubnetIds := clusterReqInfo.Network.SubnetIIDs
	var subnetIds []*string
	for _, subnetIID := range reqSubnetIds {
		subnetIds = append(subnetIds, aws.String(subnetIID.SystemId))
	}

	//AWS의 경우 사전에 Role의 생성이 필요하며, 현재는 role 이름을 다음 이름으로 일치 시킨다.(추후 개선)
	//예시) cluster : cloud-barista-spider-eks-cluster-role, Node : cloud-barista-spider-eks-nodegroup-role
	eksRoleName := "cloud-barista-spider-eks-cluster-role"
	// get Role Arn
	eksRole, err := ClusterHandler.getRole(irs.IID{SystemId: eksRoleName})
	if err != nil {
		cblogger.Error(err)
		// role 은 required 임.
		return irs.ClusterInfo{}, err
	}
	roleArn := eksRole.Role.Arn

	reqK8sVersion := clusterReqInfo.Version

	tagsMap, err := ConvertTagListToTagsMap(clusterReqInfo.TagList, clusterReqInfo.IId.NameId)
	if err != nil {
		return irs.ClusterInfo{}, fmt.Errorf("failed to convert tags map: %w", err)
	}

	// create cluster
	input := &eks.CreateClusterInput{
		Name: aws.String(clusterReqInfo.IId.NameId),
		ResourcesVpcConfig: &eks.VpcConfigRequest{
			SecurityGroupIds: securityGroupIds,
			SubnetIds:        subnetIds,
		},
		//RoleArn: aws.String("arn:aws:iam::012345678910:role/eks-service-role-AWSServiceRoleForAmazonEKS-J7ONKE3BQ4PI"),
		//RoleArn: aws.String(roleArn),
		RoleArn: roleArn,
		Tags:    tagsMap,
	}

	//EKS버전 처리(Spider 입력 값 형태 : "1.23.4" / AWS 버전 형태 : "1.23")
	if reqK8sVersion != "" {
		arrVer := strings.Split(reqK8sVersion, ".")
		switch len(arrVer) {
		case 2: // 그대로 적용
			input.Version = aws.String(reqK8sVersion)
			break
		case 3: // 앞의 2자리만 취함. (정상적인 입력 형태)
			input.Version = aws.String(arrVer[0] + "." + arrVer[1])
			break
		default: // 위 2가지 외에는 CSP의 기본값(최신버전)을 적용 함.
			break
		}
	}

	cblogger.Debug(input)

	// logger for HisCall
	callogger := call.GetLogger("HISCALL")
	callLogInfo := call.CLOUDLOGSCHEMA{
		CloudOS:      call.AWS,
		RegionZone:   ClusterHandler.Region.Zone,
		ResourceType: call.CLUSTER,
		ResourceName: clusterReqInfo.IId.NameId,
		CloudOSAPI:   "CreateCluster()",
		ElapsedTime:  "",
		ErrorMSG:     "",
	}
	callLogStart := call.Start()

	result, err := ClusterHandler.Client.CreateCluster(input)
	callLogInfo.ElapsedTime = call.Elapsed(callLogStart)

	if err != nil {
		callLogInfo.ErrorMSG = err.Error()
		callogger.Info(call.String(callLogInfo))

		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			case eks.ErrCodeResourceInUseException:
				cblogger.Error(eks.ErrCodeResourceInUseException, aerr.Error())
			case eks.ErrCodeResourceLimitExceededException:
				cblogger.Error(eks.ErrCodeResourceLimitExceededException, aerr.Error())
			case eks.ErrCodeInvalidParameterException:
				cblogger.Error(eks.ErrCodeInvalidParameterException, aerr.Error())
			case eks.ErrCodeClientException:
				cblogger.Error(eks.ErrCodeClientException, aerr.Error())
			case eks.ErrCodeServerException:
				cblogger.Error(eks.ErrCodeServerException, aerr.Error())
			case eks.ErrCodeServiceUnavailableException:
				cblogger.Error(eks.ErrCodeServiceUnavailableException, aerr.Error())
			case eks.ErrCodeUnsupportedAvailabilityZoneException:
				cblogger.Error(eks.ErrCodeUnsupportedAvailabilityZoneException, aerr.Error())
			default:
				cblogger.Error(aerr.Error())
			}
		} else {
			cblogger.Error(err.Error())
		}
		return irs.ClusterInfo{}, err
	}
	callogger.Info(call.String(callLogInfo))

	cblogger.Debug(result)

	/*// Sync Call에서 Async Call로 변경 - 이슈:#716
	//----- wait until Status=COMPLETE -----//  :  cluster describe .status 로 확인
	errWait := ClusterHandler.WaitUntilClusterActive(result.Cluster.Identity.String())
	if errWait != nil {
		cblogger.Error(errWait)
		return irs.ClusterInfo{}, errWait
	}
	*/

	/*
		//노드그룹 추가
		clusterIID := irs.IID{NameId: clusterReqInfo.IId.NameId, SystemId: result.Cluster.Identity.String()}
		nodeGroupInfoList := clusterReqInfo.NodeGroupList
		for _, nodeGroupInfo := range nodeGroupInfoList {
			resultNodeGroupInfo, nodeGroupErr := ClusterHandler.AddNodeGroup(clusterIID, nodeGroupInfo)
			if nodeGroupErr != nil {
				cblogger.Error(err.Error())
			}
			cblogger.Debug(resultNodeGroupInfo)

		//----- wait until Status=COMPLETE -----//  :  Nodegroup이 모두 생성되면 조회
	*/

	clusterReqInfo.IId.SystemId = *result.Cluster.Name
	clusterInfo, errClusterInfo := ClusterHandler.GetCluster(clusterReqInfo.IId)
	if errClusterInfo != nil {
		cblogger.Error(errClusterInfo.Error())
		return irs.ClusterInfo{}, errClusterInfo
	}
	clusterInfo.IId.NameId = clusterReqInfo.IId.NameId
	return clusterInfo, nil
}

// Nodegroup이 Activty 상태일때까지 대기함.
func (ClusterHandler *AwsClusterHandler) WaitUntilNodegroupActive(clusterName string, nodegroupName string) error {
	cblogger.Debugf("Cluster Name : [%s] / NodegroupName : [%s]", clusterName, nodegroupName)
	input := &eks.DescribeNodegroupInput{
		ClusterName:   aws.String(clusterName),
		NodegroupName: aws.String(nodegroupName),
	}

	err := ClusterHandler.Client.WaitUntilNodegroupActive(input)
	if err != nil {
		cblogger.Errorf("failed to wait until Nodegroup Active : %v", err)
		return err
	}
	cblogger.Debug("=========WaitUntilNodegroupActive() 종료")
	return nil
}

// Cluster가 Activty 상태일때까지 대기함.
func (ClusterHandler *AwsClusterHandler) WaitUntilClusterActive(clusterName string) error {
	cblogger.Debugf("Cluster Name : [%s]", clusterName)
	input := &eks.DescribeClusterInput{
		Name: aws.String(clusterName),
	}

	err := ClusterHandler.Client.WaitUntilClusterActive(input)
	if err != nil {
		cblogger.Errorf("failed to wait until cluster Active: %v", err)
		return err
	}
	cblogger.Debug("=========WaitUntilClusterActive() ended")
	return nil
}

func (ClusterHandler *AwsClusterHandler) ListCluster() ([]*irs.ClusterInfo, error) {
	//return irs.ClusterInfo{}, nil
	if ClusterHandler == nil {
		cblogger.Error("ClusterHandlerIs nil")
		return nil, errors.New("ClusterHandler is nil")

	}

	cblogger.Debug(ClusterHandler)
	if ClusterHandler.Client == nil {
		cblogger.Error(" ClusterHandler.Client Is nil")
		return nil, errors.New("ClusterHandler is nil")
	}

	input := &eks.ListClustersInput{}
	// logger for HisCall
	callogger := call.GetLogger("HISCALL")
	callLogInfo := call.CLOUDLOGSCHEMA{
		CloudOS:      call.AWS,
		RegionZone:   ClusterHandler.Region.Zone,
		ResourceType: call.CLUSTER,
		ResourceName: "List()",
		CloudOSAPI:   "ListClusters()",
		ElapsedTime:  "",
		ErrorMSG:     "",
	}
	callLogStart := call.Start()

	result, err := ClusterHandler.Client.ListClusters(input)
	callLogInfo.ElapsedTime = call.Elapsed(callLogStart)

	if err != nil {
		callLogInfo.ErrorMSG = err.Error()
		callogger.Info(call.String(callLogInfo))
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			case eks.ErrCodeInvalidParameterException:
				cblogger.Error(eks.ErrCodeInvalidParameterException, aerr.Error())
			case eks.ErrCodeClientException:
				cblogger.Error(eks.ErrCodeClientException, aerr.Error())
			case eks.ErrCodeServerException:
				cblogger.Error(eks.ErrCodeServerException, aerr.Error())
			case eks.ErrCodeServiceUnavailableException:
				cblogger.Error(eks.ErrCodeServiceUnavailableException, aerr.Error())
			default:
				cblogger.Error(aerr.Error())
			}
		} else {
			// Print the error, cast err to awserr.Error to get the Code and
			// Message from an error.
			cblogger.Error(err.Error())
		}
		return nil, err
	}
	callogger.Info(call.String(callLogInfo))

	cblogger.Debug(result)

	clusterList := []*irs.ClusterInfo{}
	for _, clusterName := range result.Clusters {

		clusterInfo, err := ClusterHandler.GetCluster(irs.IID{SystemId: *clusterName})
		if err != nil {
			cblogger.Error(err)
			continue //	에러가 나면 일단 skip시킴.
		}
		clusterList = append(clusterList, &clusterInfo)

	}
	return clusterList, nil

}

func (ClusterHandler *AwsClusterHandler) GetCluster(clusterIID irs.IID) (irs.ClusterInfo, error) {
	input := &eks.DescribeClusterInput{
		Name: aws.String(clusterIID.SystemId),
	}

	cblogger.Debug(input)

	// logger for HisCall
	callogger := call.GetLogger("HISCALL")
	callLogInfo := call.CLOUDLOGSCHEMA{
		CloudOS:      call.AWS,
		RegionZone:   ClusterHandler.Region.Zone,
		ResourceType: call.CLUSTER,
		ResourceName: clusterIID.SystemId,
		CloudOSAPI:   "DescribeCluster()",
		ElapsedTime:  "",
		ErrorMSG:     "",
	}
	callLogStart := call.Start()

	result, err := ClusterHandler.Client.DescribeCluster(input)
	callLogInfo.ElapsedTime = call.Elapsed(callLogStart)
	if err != nil {
		callLogInfo.ErrorMSG = err.Error()
		callogger.Info(call.String(callLogInfo))

		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			case eks.ErrCodeResourceNotFoundException:
				cblogger.Error(eks.ErrCodeResourceNotFoundException, aerr.Error())
			case eks.ErrCodeClientException:
				cblogger.Error(eks.ErrCodeClientException, aerr.Error())
			case eks.ErrCodeServerException:
				cblogger.Error(eks.ErrCodeServerException, aerr.Error())
			case eks.ErrCodeServiceUnavailableException:
				cblogger.Error(eks.ErrCodeServiceUnavailableException, aerr.Error())
			default:
				cblogger.Error(aerr.Error())
			}
		} else {
			cblogger.Error(err.Error())
		}
		return irs.ClusterInfo{}, err
	}
	callogger.Info(call.String(callLogInfo))

	cblogger.Debug(result)

	clusterInfo := irs.ClusterInfo{
		IId:         irs.IID{NameId: "", SystemId: *result.Cluster.Name},
		Version:     *result.Cluster.Version,
		CreatedTime: *result.Cluster.CreatedAt,
		Status:      irs.ClusterStatus(*result.Cluster.Status),
		//AccessInfo:  irs.AccessInfo{Endpoint: *result.Cluster.Endpoint},
		AccessInfo: irs.AccessInfo{},
	}
	/*
		NodeGroupList []NodeGroupInfo
		Addons        AddonsInfo
	*/

	if !reflect.ValueOf(result.Cluster.Endpoint).IsNil() {
		clusterInfo.AccessInfo.Endpoint = *result.Cluster.Endpoint
	}
	if !reflect.ValueOf(result.Cluster.CertificateAuthority.Data).IsNil() {
		clusterInfo.AccessInfo.Kubeconfig = getKubeConfig(result)
	}

	if !reflect.ValueOf(result.Cluster.ResourcesVpcConfig).IsNil() {
		clusterInfo.Network.VpcIID = irs.IID{SystemId: *result.Cluster.ResourcesVpcConfig.VpcId}

		//서브넷 처리
		//SubnetIds: ["subnet-0d30ee6b367974a39","subnet-06d5c04b32019b81f","subnet-05c5d26bd2f014591"],
		if len(result.Cluster.ResourcesVpcConfig.SubnetIds) > 0 {
			for _, curSubnetId := range result.Cluster.ResourcesVpcConfig.SubnetIds {
				clusterInfo.Network.SubnetIIDs = append(clusterInfo.Network.SubnetIIDs, irs.IID{SystemId: *curSubnetId})
			}
		}

		//클러스터 보안그룹 처리
		// ClusterSecurityGroupId: "sg-0bb02bf07fe5f42f0",
		//@TODO - 클러스터 생성시 자동으로 추가되는 보안 그룹이라서 일단 CB보안그룹 목록에 포함은 시키지 않았음.
		if !reflect.ValueOf(result.Cluster.ResourcesVpcConfig.ClusterSecurityGroupId).IsNil() {
			//if *result.Cluster.ResourcesVpcConfig.ClusterSecurityGroupId != "" {
			/*
				for _, curSecurityGroupId := range result.Cluster.ResourcesVpcConfig.SecurityGroupIds {
					clusterInfo.Network.SecurityGroupIIDs = append(clusterInfo.Network.SecurityGroupIIDs, irs.IID{SystemId: *curSecurityGroupId})
				}
			*/
		}

		//보안그룹 처리 : "추가 보안 그룹"에 해당하는 듯
		if len(result.Cluster.ResourcesVpcConfig.SecurityGroupIds) > 0 {
			for _, curSecurityGroupId := range result.Cluster.ResourcesVpcConfig.SecurityGroupIds {
				clusterInfo.Network.SecurityGroupIIDs = append(clusterInfo.Network.SecurityGroupIIDs, irs.IID{SystemId: *curSecurityGroupId})
			}
		}
	}

	keyValueList := []irs.KeyValue{
		{Key: "Status", Value: *result.Cluster.Status},
		{Key: "Arn", Value: *result.Cluster.Arn},
		{Key: "RoleArn", Value: *result.Cluster.RoleArn},
	}
	clusterInfo.KeyValueList = keyValueList

	//노드 그룹 처리
	resNodeGroupList, errNodeGroup := ClusterHandler.ListNodeGroup(clusterInfo.IId)
	if errNodeGroup != nil {
		cblogger.Error(errNodeGroup)
		return irs.ClusterInfo{}, errNodeGroup
	}

	cblogger.Debug(resNodeGroupList)

	//노드 그룹 타입 변환
	for _, curNodeGroup := range resNodeGroupList {
		cblogger.Debugf("Nod Group : [%s]", curNodeGroup.IId.NameId)
		clusterInfo.NodeGroupList = append(clusterInfo.NodeGroupList, *curNodeGroup)
	}

	cblogger.Debug(clusterInfo)

	return clusterInfo, nil
}

func getKubeConfig(clusterDesc *eks.DescribeClusterOutput) string {

	cluster := clusterDesc.Cluster

	kubeconfigContent := fmt.Sprintf(`apiVersion: v1
clusters:
- cluster:
    server: %s
    certificate-authority-data: %s
  name: kubernetes
contexts:
- context:
    cluster: kubernetes
    user: aws
  name: aws
current-context: aws
kind: Config
preferences: {}
users:
- name: aws
  user:
    exec:
      apiVersion: client.authentication.k8s.io/v1beta1
      command: aws
      args:
        - "eks"
        - "get-token"
        - "--cluster-name"
        - "%s"
`, *cluster.Endpoint, *cluster.CertificateAuthority.Data, *cluster.Name)

	return kubeconfigContent
}

func (ClusterHandler *AwsClusterHandler) DeleteCluster(clusterIID irs.IID) (bool, error) {
	cblogger.Infof("Cluster Name : %s", clusterIID.SystemId)
	input := &eks.DeleteClusterInput{
		Name: aws.String(clusterIID.SystemId),
	}

	cblogger.Debug(input)

	// logger for HisCall
	callogger := call.GetLogger("HISCALL")
	callLogInfo := call.CLOUDLOGSCHEMA{
		CloudOS:      call.AWS,
		RegionZone:   ClusterHandler.Region.Zone,
		ResourceType: call.CLUSTER,
		ResourceName: clusterIID.SystemId,
		CloudOSAPI:   "DeleteCluster()",
		ElapsedTime:  "",
		ErrorMSG:     "",
	}
	callLogStart := call.Start()

	result, err := ClusterHandler.Client.DeleteCluster(input)
	callLogInfo.ElapsedTime = call.Elapsed(callLogStart)

	if err != nil {
		callLogInfo.ErrorMSG = err.Error()
		callogger.Info(call.String(callLogInfo))

		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			case eks.ErrCodeResourceInUseException:
				cblogger.Error(eks.ErrCodeResourceInUseException, aerr.Error())
			case eks.ErrCodeResourceNotFoundException:
				cblogger.Error(eks.ErrCodeResourceNotFoundException, aerr.Error())
			case eks.ErrCodeClientException:
				cblogger.Error(eks.ErrCodeClientException, aerr.Error())
			case eks.ErrCodeServerException:
				cblogger.Error(eks.ErrCodeServerException, aerr.Error())
			case eks.ErrCodeServiceUnavailableException:
				cblogger.Error(eks.ErrCodeServiceUnavailableException, aerr.Error())
			default:
				cblogger.Error(aerr.Error())
			}
		} else {
			// Print the error, cast err to awserr.Error to get the Code and
			// Message from an error.
			cblogger.Error(err.Error())
		}
		return false, err
	}
	callogger.Info(call.String(callLogInfo))

	cblogger.Debug(result)
	/*
		waitInput := &eks.DescribeClusterInput{
			Name: aws.String(clusterIID.SystemId),
		}
		err = ClusterHandler.Client.WaitUntilClusterDeleted(waitInput)
		if err != nil {
			return false, err
		}
	*/
	return true, nil
}

// ------ NodeGroup Management

/*
Cluster.NetworkInfo 설정과 동일 서브넷으로 설정
NodeGroup 추가시에는 대상 Cluster 정보 획득하여 설정
NodeGroup에 다른 Subnet 설정이 꼭 필요시 추후 재논의
//https://github.com/cloud-barista/cb-spider/wiki/Provider-Managed-Kubernetes-and-Driver-API
*/
func (ClusterHandler *AwsClusterHandler) AddNodeGroup(clusterIID irs.IID, nodeGroupReqInfo irs.NodeGroupInfo) (irs.NodeGroupInfo, error) {
	// validation check
	if nodeGroupReqInfo.MaxNodeSize < 1 { // nodeGroupReqInfo.MaxNodeSize 는 최소가 1이다.
		return irs.NodeGroupInfo{}, awserr.New(CUSTOM_ERR_CODE_BAD_REQUEST, "The MaxNodeSize value must be greater than or equal to 1.", nil)
	}

	// get Role Arn
	eksRoleName := "cloud-barista-spider-eks-nodegroup-role"
	eksRole, err := ClusterHandler.getRole(irs.IID{SystemId: eksRoleName})
	if err != nil {
		cblogger.Error(err)
		// role 은 required 임.
		return irs.NodeGroupInfo{}, err
	}
	roleArn := eksRole.Role.Arn

	clusterInfo, err := ClusterHandler.GetCluster(clusterIID)
	if err != nil {
		cblogger.Error(err)
		return irs.NodeGroupInfo{}, err
	}

	networkInfo := clusterInfo.Network
	var subnetList []*string
	for _, subnet := range networkInfo.SubnetIIDs {
		subnetId := subnet.SystemId // 포인터라서 subnet.SystemId를 직접 Append하면 안 됨.
		subnetList = append(subnetList, &subnetId)
	}

	cblogger.Debug("Final Subnet List")
	// 이 부분에서 VPC subnet ID 를 바탕으로 리스트 순회하며 ModifySubnetAttribute 를 통해 Auto-assign public IPv4 address를 활성화
	for _, subnetIdPtr := range subnetList {
		input := &ec2.ModifySubnetAttributeInput{
			MapPublicIpOnLaunch: &ec2.AttributeBooleanValue{
				Value: aws.Bool(true),
			},
			SubnetId: subnetIdPtr,
		}
		_, err := ClusterHandler.EC2Client.ModifySubnetAttribute(input)
		if err != nil {
			errmsg := "error during ModifySubnetAttribute to MapPublicIpOnLaunch=TRUE on subnet : " + *subnetIdPtr
			cblogger.Error(err)
			cblogger.Error(errmsg)
			// return irs.NodeGroupInfo{}, errors.New(errmsg) // 서브넷 순회 이므로 나머지 서브넷은 진행하도록 주석처리함.
		}
	}

	cblogger.Debug("Subnet list")
	cblogger.Debug(subnetList)

	var nodeSecurityGroupList []*string
	for _, securityGroup := range networkInfo.SecurityGroupIIDs {
		nodeSecurityGroupList = append(nodeSecurityGroupList, &securityGroup.SystemId)
	}

	tags := map[string]string{}
	tags["key"] = NODEGROUP_TAG
	tags["value"] = nodeGroupReqInfo.IId.NameId

	input := &eks.CreateNodegroupInput{
		//AmiType: "", // Valid Values: AL2_x86_64 | AL2_x86_64_GPU | AL2_ARM_64 | CUSTOM | BOTTLEROCKET_ARM_64 | BOTTLEROCKET_x86_64, Required: No
		//CapacityType: aws.String("ON_DEMAND"),//Valid Values: ON_DEMAND | SPOT, Required: No

		//ClusterName:   aws.String("cb-eks-cluster"),              //uri, required
		ClusterName:   aws.String(clusterIID.SystemId),         //uri, required
		NodegroupName: aws.String(nodeGroupReqInfo.IId.NameId), // required
		Tags:          aws.StringMap(tags),
		//NodeRole:      aws.String(eksRoleName), // roleName, required
		NodeRole: roleArn,
		ScalingConfig: &eks.NodegroupScalingConfig{
			DesiredSize: aws.Int64(int64(nodeGroupReqInfo.DesiredNodeSize)),
			MaxSize:     aws.Int64(int64(nodeGroupReqInfo.MaxNodeSize)),
			MinSize:     aws.Int64(int64(nodeGroupReqInfo.MinNodeSize)),
		},
		Subnets: subnetList,

		//DiskSize: 0,
		//InstanceTypes: ["",""],
		//Labels : {"key": "value"},
		//LaunchTemplate: {
		//	Id: "",
		//	Name: "",
		//	Version: ""
		//},

		//ReleaseVersion: "",
		RemoteAccess: &eks.RemoteAccessConfig{
			Ec2SshKey:            &nodeGroupReqInfo.KeyPairIID.SystemId,
			SourceSecurityGroups: nodeSecurityGroupList,
		},

		//Taints: [{
		//	Effect:"",
		//	Key : "",
		//	Value :""
		//}],
		//UpdateConfig: {
		//	MaxUnavailable: 0,
		//	MaxUnavailablePercentage: 0
		//},
		//Version: ""
	}

	// 필수 외에 넣을 항목들 set
	rootDiskSize, _ := strconv.ParseInt(nodeGroupReqInfo.RootDiskSize, 10, 64)
	if rootDiskSize > 0 {
		input.DiskSize = aws.Int64(rootDiskSize)
	}

	if !strings.EqualFold(nodeGroupReqInfo.VMSpecName, "") {
		var nodeSpec []string
		nodeSpec = append(nodeSpec, nodeGroupReqInfo.VMSpecName) //"p2.xlarge"
		input.InstanceTypes = aws.StringSlice(nodeSpec)
	}

	cblogger.Debug(input)

	result, err := ClusterHandler.Client.CreateNodegroup(input) // 비동기
	if err != nil {
		cblogger.Error(err)
		return irs.NodeGroupInfo{}, err
	}

	cblogger.Debug(result)

	nodegroupName := result.Nodegroup.NodegroupName

	/*// Sync Call에서 Async Call로 변경 - 이슈:#716
	//노드 그룹이 활성화될 때까지 대기
	errWait := ClusterHandler.WaitUntilNodegroupActive(clusterIID.SystemId, *nodegroupName)
	if errWait != nil {
		cblogger.Error(errWait)
		return irs.NodeGroupInfo{}, errWait
	}
	*/

	nodeGroup, err := ClusterHandler.GetNodeGroup(clusterIID, irs.IID{NameId: nodeGroupReqInfo.IId.NameId, SystemId: *nodegroupName})
	if err != nil {
		cblogger.Error(err)
		return irs.NodeGroupInfo{}, err
	}
	nodeGroup.IId.NameId = nodeGroupReqInfo.IId.NameId
	return nodeGroup, nil
}

func (ClusterHandler *AwsClusterHandler) ListNodeGroup(clusterIID irs.IID) ([]*irs.NodeGroupInfo, error) {
	input := &eks.ListNodegroupsInput{
		ClusterName: aws.String(clusterIID.SystemId),
	}
	cblogger.Debug(input)

	result, err := ClusterHandler.Client.ListNodegroups(input)
	if err != nil {
		cblogger.Error(err)
		return nil, err
	}
	cblogger.Debug(result)
	nodeGroupInfoList := []*irs.NodeGroupInfo{}
	for _, nodeGroupName := range result.Nodegroups {
		nodeGroupInfo, err := ClusterHandler.GetNodeGroup(clusterIID, irs.IID{SystemId: *nodeGroupName})
		if err != nil {
			cblogger.Error(err)
			//return nil, err
			continue
		}
		nodeGroupInfoList = append(nodeGroupInfoList, &nodeGroupInfo)
	}
	return nodeGroupInfoList, nil
}

func (ClusterHandler *AwsClusterHandler) GetNodeGroup(clusterIID irs.IID, nodeGroupIID irs.IID) (irs.NodeGroupInfo, error) {
	cblogger.Debugf("Cluster SystemId : [%s] / NodeGroup SystemId : [%s]", clusterIID.SystemId, nodeGroupIID.SystemId)

	input := &eks.DescribeNodegroupInput{
		//AmiType: "", // Valid Values: AL2_x86_64 | AL2_x86_64_GPU | AL2_ARM_64 | CUSTOM | BOTTLEROCKET_ARM_64 | BOTTLEROCKET_x86_64, Required: No
		//CapacityType: aws.String("ON_DEMAND"),//Valid Values: ON_DEMAND | SPOT, Required: No
		ClusterName:   aws.String(clusterIID.SystemId),   //required
		NodegroupName: aws.String(nodeGroupIID.SystemId), // required
	}

	result, err := ClusterHandler.Client.DescribeNodegroup(input)
	cblogger.Debug("===> Node Group Invocation Result")
	cblogger.Debug(result)
	if err != nil {
		cblogger.Error(err)
		return irs.NodeGroupInfo{}, err
	}

	nodeGroupInfo, err := ClusterHandler.convertNodeGroup(result)
	if err != nil {
		cblogger.Error(err)
		return irs.NodeGroupInfo{}, err
	}
	return nodeGroupInfo, nil
}

func (ClusterHandler *AwsClusterHandler) GetAutoScalingGroups(autoScalingGroupName string) ([]irs.IID, error) {
	input := &autoscaling.DescribeAutoScalingGroupsInput{
		AutoScalingGroupNames: []*string{
			aws.String(autoScalingGroupName),
		},
	}

	result, err := ClusterHandler.AutoScaling.DescribeAutoScalingGroups(input)
	cblogger.Debug(result)

	if err != nil {
		cblogger.Error(err)
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			case autoscaling.ErrCodeInvalidNextToken:
				cblogger.Error(autoscaling.ErrCodeInvalidNextToken, aerr.Error())
			case autoscaling.ErrCodeResourceContentionFault:
				cblogger.Error(autoscaling.ErrCodeResourceContentionFault, aerr.Error())
			default:
				cblogger.Error(aerr.Error())
			}
		} else {
			// Print the error, cast err to awserr.Error to get the Code and
			// Message from an error.
			cblogger.Error(err.Error())
		}
		return nil, err
	}

	//nodeList := []*irs.IID{}
	nodeList := []irs.IID{}
	//AutoScalingGroups
	if len(result.AutoScalingGroups) > 0 {
		for _, curGroup := range result.AutoScalingGroups[0].Instances {
			cblogger.Debugf("   ====> [%s]", *curGroup.InstanceId)
			nodeList = append(nodeList, irs.IID{SystemId: *curGroup.InstanceId})
		}
	}

	cblogger.Debug("**VM Instance List**")
	cblogger.Debug(nodeList)
	return nodeList, nil
}

/*
AutoScaling 이라는 별도의 메뉴가 있음.
*/
func (ClusterHandler *AwsClusterHandler) SetNodeGroupAutoScaling(clusterIID irs.IID, nodeGroupIID irs.IID, on bool) (bool, error) {
	return false, nil
}

func (ClusterHandler *AwsClusterHandler) ChangeNodeGroupScaling(clusterIID irs.IID, nodeGroupIID irs.IID,
	DesiredNodeSize int, MinNodeSize int, MaxNodeSize int) (irs.NodeGroupInfo, error) {
	cblogger.Infof("Cluster SystemId : [%s] / NodeGroup SystemId : [%s] / DesiredNodeSize : [%d] / MinNodeSize : [%d] / MaxNodeSize : [%d]", clusterIID.SystemId, nodeGroupIID.SystemId, DesiredNodeSize, MinNodeSize, MaxNodeSize)

	// clusterIID로 cluster 정보를 조회
	// nodeGroupIID로 nodeGroup 정보를 조회
	// 		nodeGroup에 AutoScaling 그룹 이름이 있음.

	// TODO : 공통으로 뺄 것
	input := &eks.DescribeNodegroupInput{
		ClusterName:   aws.String(clusterIID.SystemId),   //required
		NodegroupName: aws.String(nodeGroupIID.SystemId), // required
	}

	result, err := ClusterHandler.Client.DescribeNodegroup(input)
	cblogger.Debug(result.Nodegroup)
	if err != nil {
		cblogger.Error(err)
		return irs.NodeGroupInfo{}, err
	}

	nodeGroupName := result.Nodegroup.NodegroupName
	nodeGroupResources := result.Nodegroup.Resources.AutoScalingGroups
	for _, autoScalingGroup := range nodeGroupResources {
		input := &autoscaling.UpdateAutoScalingGroupInput{
			AutoScalingGroupName: aws.String(*autoScalingGroup.Name),

			MaxSize:         aws.Int64(int64(MaxNodeSize)),
			MinSize:         aws.Int64(int64(MinNodeSize)),
			DesiredCapacity: aws.Int64(int64(DesiredNodeSize)),
		}

		updateResult, err := ClusterHandler.AutoScaling.UpdateAutoScalingGroup(input)
		if err != nil {
			cblogger.Error(err)
			return irs.NodeGroupInfo{}, err
		}
		cblogger.Debug(updateResult)
	}

	nodeGroupInfo, err := ClusterHandler.GetNodeGroup(clusterIID, irs.IID{SystemId: *nodeGroupName})
	if err != nil {
		cblogger.Error(err)
		return irs.NodeGroupInfo{}, err
	}
	return nodeGroupInfo, nil
}

func (ClusterHandler *AwsClusterHandler) RemoveNodeGroup(clusterIID irs.IID, nodeGroupIID irs.IID) (bool, error) {
	cblogger.Infof("Cluster SystemId : [%s] / NodeGroup SystemId : [%s]", clusterIID.SystemId, nodeGroupIID.SystemId)
	input := &eks.DeleteNodegroupInput{
		ClusterName:   aws.String(clusterIID.SystemId),   //required
		NodegroupName: aws.String(nodeGroupIID.SystemId), // required
	}

	result, err := ClusterHandler.Client.DeleteNodegroup(input)
	if err != nil {
		cblogger.Error(err)
		return false, err
	}

	cblogger.Debug(result)

	return true, nil
}

// ------ Upgrade K8S
func (ClusterHandler *AwsClusterHandler) UpgradeCluster(clusterIID irs.IID, newVersion string) (irs.ClusterInfo, error) {
	cblogger.Infof("Cluster SystemId : [%s] / Request New Version : [%s]", clusterIID.SystemId, newVersion)

	// -- version 만 update인 경우
	input := &eks.UpdateClusterVersionInput{
		Name:    aws.String(clusterIID.SystemId),
		Version: aws.String(newVersion),
	}

	result, err := ClusterHandler.Client.UpdateClusterVersion(input)
	if err != nil {
		cblogger.Error(err)
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			case eks.ErrCodeInvalidParameterException:
				cblogger.Error(eks.ErrCodeInvalidParameterException, aerr.Error())
			case eks.ErrCodeClientException:
				cblogger.Error(eks.ErrCodeClientException, aerr.Error())
			case eks.ErrCodeResourceNotFoundException:
				cblogger.Error(eks.ErrCodeResourceNotFoundException, aerr.Error())
			case eks.ErrCodeServerException:
				cblogger.Error(eks.ErrCodeServerException, aerr.Error())
			case eks.ErrCodeInvalidRequestException:
				cblogger.Error(eks.ErrCodeInvalidRequestException, aerr.Error())
			default:
				cblogger.Error(aerr.Error())
			}
		} else {
			// Print the error, cast err to awserr.Error to get the Code and
			// Message from an error.
			cblogger.Error(err.Error())
		}
	}
	cblogger.Debug(result)
	// getClusterInfo
	return irs.ClusterInfo{}, nil

}

func (ClusterHandler *AwsClusterHandler) getRole(role irs.IID) (*iam.GetRoleOutput, error) {
	input := &iam.GetRoleInput{
		RoleName: aws.String(role.SystemId),
	}

	result, err := ClusterHandler.Iam.GetRole(input)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			case iam.ErrCodeNoSuchEntityException:
				cblogger.Error(iam.ErrCodeNoSuchEntityException, aerr.Error())
			case iam.ErrCodeServiceFailureException:
				cblogger.Error(iam.ErrCodeServiceFailureException, aerr.Error())
			default:
				cblogger.Error(aerr.Error())
			}
		} else {
			// Print the error, cast err to awserr.Error to get the Code and
			// Message from an error.
			cblogger.Error(err.Error())
		}
		return nil, err
	}

	return result, nil
}

/*
EKS의 NodeGroup정보를 Spider의 NodeGroup으로 변경
*/
func (NodeGroupHandler *AwsClusterHandler) convertNodeGroup(nodeGroupOutput *eks.DescribeNodegroupOutput) (irs.NodeGroupInfo, error) {
	nodeGroupInfo := irs.NodeGroupInfo{}
	PrintToJson(nodeGroupOutput)

	nodeGroup := nodeGroupOutput.Nodegroup
	//nodeRole := nodeGroup.NodeRole
	//version := nodeGroup.Version
	//releaseVersion := nodeGroup.ReleaseVersion

	//subnetList := nodeGroup.Subnets
	//nodeGroupStatus := nodeGroup.Status
	nodeGroupInfo.Status = irs.NodeGroupStatus(*nodeGroup.Status)
	instanceTypeList := nodeGroup.InstanceTypes // spec

	//nodes := nodeGroup.Health.Issues[0].ResourceIds // 문제 있는 node들만 있는것이 아닌지..
	rootDiskSize := nodeGroup.DiskSize
	//nodeGroup.Taints// 미사용
	nodeGroupTagList := nodeGroup.Tags
	scalingConfig := nodeGroup.ScalingConfig
	//nodeGroup.RemoteAccess
	nodeGroupName := nodeGroup.NodegroupName

	//nodeGroup.LaunchTemplate //미사용
	//clusterName := nodeGroup.ClusterName
	//capacityType := nodeGroup.CapacityType // "ON_DEMAND"
	nodeGroupInfo.ImageIID.NameId = *nodeGroup.AmiType // AL2_x86_64"
	//createTime := nodeGroup.CreatedAt
	//health := nodeGroup.Health // Code, Message, ResourceIds	// ,"Health":{"Issues":[{"Code":"NodeCreationFailure","Message":"Unhealthy nodes in the kubernetes cluster",
	//labelList := nodeGroup.Labels
	//nodeGroupArn := nodeGroup.NodegroupArn
	//nodeGroupResources := nodeGroup.Resources
	//nodeGroupResources.AutoScalingGroups// 미사용
	//nodeGroupResources.RemoteAccessSecurityGroup// 미사용

	/*
		nodes := []irs.IID{}
		for _, issue := range nodeGroup.Health.Issues {
			resourceIds := issue.ResourceIds
			for _, resourceId := range resourceIds {
				nodes = append(nodes, irs.IID{SystemId: *resourceId})
			}
		}
		nodeGroupInfo.NodeList = nodes
	*/

	//=============
	//VM 노드 조회
	//=============
	// 오토스케일링 그룹 목록에서 VM 목록 정보 추출
	//"Resources":{"AutoScalingGroups":[{"Name":"eks-cb-eks-nodegroup-test-fec135d9-c812-8862-e3b0-7b773ce70d2e"}],

	if !reflect.ValueOf(nodeGroup.Resources).IsNil() {
		if !reflect.ValueOf(nodeGroup.Resources.AutoScalingGroups).IsNil() {
			autoscalingGroupName := *nodeGroup.Resources.AutoScalingGroups[0].Name //"eks-cb-eks-node-test02a-aws-9cc2876a-d3cb-2c25-55a8-9a19c431e716"
			cblogger.Debugf("autoscalingGroupName : [%s]", autoscalingGroupName)

			if autoscalingGroupName != "" {
				nodeList, errNodeList := NodeGroupHandler.GetAutoScalingGroups(autoscalingGroupName)
				if errNodeList != nil {
					return irs.NodeGroupInfo{}, errNodeList
				}

				nodeGroupInfo.Nodes = nodeList
			}
		}
	}

	nodeGroupInfo.DesiredNodeSize = int(*scalingConfig.DesiredSize)
	nodeGroupInfo.MinNodeSize = int(*scalingConfig.MinSize)
	nodeGroupInfo.MaxNodeSize = int(*scalingConfig.MaxSize)

	if nodeGroupTagList == nil {
		nodeGroupTagList[NODEGROUP_TAG] = nodeGroupName // 값이없으면 nodeGroupName이랑 같은값으로 set.
	}
	nodeGroupTag := ""
	for key, val := range nodeGroupTagList {
		if strings.EqualFold("key", NODEGROUP_TAG) {
			nodeGroupTag = *val
			break
		}
		cblogger.Debug(key, *val)
	}
	//printToJson(nodeGroupTagList)
	cblogger.Debug("nodeGroupName=", *nodeGroupName)
	cblogger.Debug("tag=", nodeGroupTagList[NODEGROUP_TAG])
	nodeGroupInfo.IId = irs.IID{
		NameId:   nodeGroupTag, // TAG에 이름
		SystemId: *nodeGroupName,
	}
	nodeGroupInfo.VMSpecName = *instanceTypeList[0]
	//nodeGroupInfo.ImageIID

	if !reflect.ValueOf(nodeGroup.RemoteAccess).IsNil() {
		if !reflect.ValueOf(nodeGroup.RemoteAccess.Ec2SshKey).IsNil() {
			nodeGroupInfo.KeyPairIID = irs.IID{
				SystemId: *nodeGroup.RemoteAccess.Ec2SshKey,
			}
		}
	}

	//nodeGroupInfo.RootDiskSize = strconv.FormatInt(*nodeGroup.DiskSize, 10)
	nodeGroupInfo.RootDiskSize = strconv.FormatInt(*rootDiskSize, 10)

	// TODO : node 목록 NodegroupArn 으로 조회해야하나??
	//nodeList := []irs.IID{}
	//if nodeList != nil {
	//	for _, nodeId := range nodes {
	//		nodeList = append(nodeList, irs.IID{NameId: "", SystemId: *nodeId})
	//	}
	//}
	//nodeGroupInfo.NodeList = nodeList
	//cblogger.Info("NodeGroup")
	//	{"Nodegroup":
	//		{"AmiType":"AL2_x86_64"
	//		,"CapacityType":"ON_DEMAND"
	//		,"ClusterName":"cb-eks-cluster"
	//		,"CreatedAt":"2022-08-05T01:51:49.673Z"
	//		,"DiskSize":20
	//		,"Health":{
	//					"Issues":[
	//							{"Code":"NodeCreationFailure"
	//							,"Message":"Unhealthy nodes in the kubernetes cluster"
	//							,"ResourceIds":["i-06ee95583f3f7de5c","i-0a283a92dcce27aa8"]}]},
	//		"InstanceTypes":["t3.medium"],
	//		"Labels":{},
	//		"LaunchTemplate":null,
	//		"ModifiedAt":"2022-08-05T02:15:14.308Z",
	//		"NodeRole":"arn:aws:iam::050864702683:role/cb-eks-nodegroup-role",
	//		"NodegroupArn":"arn:aws:eks:ap-northeast-2:050864702683:nodegroup/cb-eks-cluster/cb-eks-nodegroup-test/fec135d9-c812-8862-e3b0-7b773ce70d2e","NodegroupName":"cb-eks-nodegro
	//up-test",
	//		"ReleaseVersion":"1.22.9-20220725",
	//		"RemoteAccess":{"Ec2SshKey":"cb-webtool","SourceSecurityGroups":["sg-04607666"]},
	//		"Resources":{"AutoScalingGroups":[{"Name":"eks-cb-eks-nodegroup-test-fec135d9-c812-8862-e3b0-7b773ce70d2e"}],
	//		"RemoteAccessSecurityGroup":null},
	//		"ScalingConfig":{"DesiredSize":2,"MaxSize":2,"MinSize":2},
	//		"Status":"CREATE_FAILED",
	//		"Subnets":["subnet-262d6d7a","subnet-d0ee6fab","subnet-875a62cb","subnet-e08f5b8b"],
	//		"Tags":{},
	//		"Taints":null,
	//		"UpdateConfig":{"MaxUnavailable":1,"MaxUnavailablePercentage":null},
	//		"Version":"1.22"}}

	//nodeGroupArn
	// arn format
	//arn:partition:service:region:account-id:resource-id
	//arn:partition:service:region:account-id:resource-type/resource-id
	//arn:partition:service:region:account-id:resource-type:resource-id

	PrintToJson(nodeGroupInfo)
	//return irs.NodeGroupInfo{}, awserr.New(CUSTOM_ERR_CODE_BAD_REQUEST, "추출 오류", nil)
	return nodeGroupInfo, nil
}
