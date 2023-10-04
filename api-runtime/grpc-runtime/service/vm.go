// gRPC Runtime of CB-Spider.
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// by CB-Spider Team, 2020.09.

package service

import (
	"context"

	gc "github.com/cloud-barista/cb-spider/api-runtime/grpc-runtime/common"
	"github.com/cloud-barista/cb-spider/api-runtime/grpc-runtime/logger"
	pb "github.com/cloud-barista/cb-spider/api-runtime/grpc-runtime/stub/cbspider"

	cmrt "github.com/cloud-barista/cb-spider/api-runtime/common-runtime"
	cres "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
)

// ===== [ Constants and Variables ] =====

// ===== [ Types ] =====

// ===== [ Implementations ] =====

// StartVM - VM 시작
func (s *CCMService) StartVM(ctx context.Context, req *pb.VMCreateRequest) (*pb.VMInfoResponse, error) {
	logger := logger.NewLogger()

	logger.Debug("calling CCMService.StartVM()")

	// Rest RegInfo => Driver ReqInfo
	// (1) create SecurityGroup IID List
	sgIIDList := []cres.IID{}
	for _, sgName := range req.Item.SecurityGroupNames {
		// SG NameID format => {VPC NameID} + cm.SG_DELIMITER + {SG NameID}
		// transform: SG NameID => {VPC NameID}-{SG NameID}
		// sgIID := cres.IID{NameId: req.Item.VpcName + cm.SG_DELIMITER + sgName, SystemId: ""}
		sgIID := cres.IID{sgName, ""}
		sgIIDList = append(sgIIDList, sgIID)
	}
	// (2) create VMReqInfo with SecurityGroup IID List
	reqInfo := cres.VMReqInfo{
		IId:               cres.IID{NameId: req.Item.Name, SystemId: ""},
		ImageIID:          cres.IID{NameId: req.Item.ImageName, SystemId: ""},
		VpcIID:            cres.IID{NameId: req.Item.VpcName, SystemId: ""},
		SubnetIID:         cres.IID{NameId: req.Item.SubnetName, SystemId: ""},
		SecurityGroupIIDs: sgIIDList,

		VMSpecName: req.Item.VmSpecName,
		KeyPairIID: cres.IID{NameId: req.Item.KeyPairName, SystemId: ""},

		RootDiskType: req.Item.RootDiskType,
		RootDiskSize: req.Item.RootDiskSize,

		VMUserId:     req.Item.VmUserId,
		VMUserPasswd: req.Item.VmUserPasswd,
	}

	// Call common-runtime API
	result, err := cmrt.StartVM(req.ConnectionName, rsVM, reqInfo)
	if err != nil {
		return nil, gc.ConvGrpcStatusErr(err, "", "CCMService.StartVM()")
	}

	// CCM 객체에서 GRPC 메시지로 복사
	var grpcObj pb.VMInfo
	err = gc.CopySrcToDest(result, &grpcObj)
	if err != nil {
		return nil, gc.ConvGrpcStatusErr(err, "", "CCMService.StartVM()")
	}

	resp := &pb.VMInfoResponse{Item: &grpcObj}
	return resp, nil
}

// ControlVM - VM 제어
func (s *CCMService) ControlVM(ctx context.Context, req *pb.VMActionRequest) (*pb.StatusResponse, error) {
	logger := logger.NewLogger()

	logger.Debug("calling CCMService.ControlVM()")

	// Call common-runtime API
	result, err := cmrt.ControlVM(req.ConnectionName, rsVM, req.Name, req.Action)
	if err != nil {
		return nil, gc.ConvGrpcStatusErr(err, "", "CCMService.ControlVM()")
	}

	resp := &pb.StatusResponse{Status: string(result)}
	return resp, nil
}

// ListVM - VM 목록
func (s *CCMService) ListVM(ctx context.Context, req *pb.VMAllQryRequest) (*pb.ListVMInfoResponse, error) {
	logger := logger.NewLogger()

	logger.Debug("calling CCMService.ListVM()")

	// Call common-runtime API
	result, err := cmrt.ListVM(req.ConnectionName, rsVM)
	if err != nil {
		return nil, gc.ConvGrpcStatusErr(err, "", "CCMService.ListVM()")
	}

	// CCM 객체에서 GRPC 메시지로 복사
	var grpcObj []*pb.VMInfo
	err = gc.CopySrcToDest(&result, &grpcObj)
	if err != nil {
		return nil, gc.ConvGrpcStatusErr(err, "", "CCMService.ListVM()")
	}

	resp := &pb.ListVMInfoResponse{Items: grpcObj}
	return resp, nil
}

// GetVM - VM 조회
func (s *CCMService) GetVM(ctx context.Context, req *pb.VMQryRequest) (*pb.VMInfoResponse, error) {
	logger := logger.NewLogger()

	logger.Debug("calling CCMService.GetVM()")

	// Call common-runtime API
	result, err := cmrt.GetVM(req.ConnectionName, rsVM, req.Name)
	if err != nil {
		return nil, gc.ConvGrpcStatusErr(err, "", "CCMService.GetVM()")
	}

	// CCM 객체에서 GRPC 메시지로 복사
	var grpcObj pb.VMInfo
	err = gc.CopySrcToDest(result, &grpcObj)
	if err != nil {
		return nil, gc.ConvGrpcStatusErr(err, "", "CCMService.GetVM()")
	}

	resp := &pb.VMInfoResponse{Item: &grpcObj}
	return resp, nil
}

// ListVMStatus - VM 상태 목록
func (s *CCMService) ListVMStatus(ctx context.Context, req *pb.VMAllQryRequest) (*pb.ListVMStatusInfoResponse, error) {
	logger := logger.NewLogger()

	logger.Debug("calling CCMService.ListVMStatus()")

	// Call common-runtime API
	result, err := cmrt.ListVMStatus(req.ConnectionName, rsVM)
	if err != nil {
		return nil, gc.ConvGrpcStatusErr(err, "", "CCMService.ListVMStatus()")
	}

	// CCM 객체에서 GRPC 메시지로 복사
	var grpcObj []*pb.VMStatusInfo
	err = gc.CopySrcToDest(&result, &grpcObj)
	if err != nil {
		return nil, gc.ConvGrpcStatusErr(err, "", "CCMService.ListVMStatus()")
	}

	resp := &pb.ListVMStatusInfoResponse{Items: grpcObj}
	return resp, nil
}

// GetVMStatus - VM 상태 조회
func (s *CCMService) GetVMStatus(ctx context.Context, req *pb.VMQryRequest) (*pb.StatusResponse, error) {
	logger := logger.NewLogger()

	logger.Debug("calling CCMService.GetVMStatus()")

	// Call common-runtime API
	result, err := cmrt.GetVMStatus(req.ConnectionName, rsVM, req.Name)
	if err != nil {
		return nil, gc.ConvGrpcStatusErr(err, "", "CCMService.GetVMStatus()")
	}

	resp := &pb.StatusResponse{Status: string(result)}
	return resp, nil
}

// TerminateVM - VM 삭제
func (s *CCMService) TerminateVM(ctx context.Context, req *pb.VMQryRequest) (*pb.StatusResponse, error) {
	logger := logger.NewLogger()

	logger.Debug("calling CCMService.TerminateVM()")

	// Call common-runtime API
	_, result, err := cmrt.DeleteVM(req.ConnectionName, rsVM, req.Name, req.Force)
	if err != nil {
		return nil, gc.ConvGrpcStatusErr(err, "", "CCMService.TerminateVM()")
	}

	resp := &pb.StatusResponse{Status: string(result)}
	return resp, nil
}

// ListAllVM - 관리 VM 목록
func (s *CCMService) ListAllVM(ctx context.Context, req *pb.VMAllQryRequest) (*pb.AllResourceInfoResponse, error) {
	logger := logger.NewLogger()

	logger.Debug("calling CCMService.ListAllVM()")

	// Call common-runtime API
	allResourceList, err := cmrt.ListAllResource(req.ConnectionName, rsVM)
	if err != nil {
		return nil, gc.ConvGrpcStatusErr(err, "", "CCMService.ListAllVM()")
	}

	// CCM 객체에서 GRPC 메시지로 복사
	var grpcObj pb.AllResourceInfoResponse
	err = gc.CopySrcToDest(&allResourceList, &grpcObj)
	if err != nil {
		return nil, gc.ConvGrpcStatusErr(err, "", "CCMService.ListAllVM()")
	}

	return &grpcObj, nil
}

// TerminateCSPVM - CSP VM 삭제
func (s *CCMService) TerminateCSPVM(ctx context.Context, req *pb.CSPVMQryRequest) (*pb.StatusResponse, error) {
	logger := logger.NewLogger()

	logger.Debug("calling CCMService.TerminateCSPVM()")

	// Call common-runtime API
	_, result, err := cmrt.DeleteCSPResource(req.ConnectionName, rsVM, req.Id)
	if err != nil {
		return nil, gc.ConvGrpcStatusErr(err, "", "CCMService.TerminateCSPVM()")
	}

	resp := &pb.StatusResponse{Status: string(result)}
	return resp, nil
}

// RegisterVM - VM 등록
func (s *CCMService) RegisterVM(ctx context.Context, req *pb.VMRegisterRequest) (*pb.VMInfoResponse, error) {
	logger := logger.NewLogger()

	logger.Debug("calling CCMService.RegisterVM()")

	userIId := cres.IID{req.Item.Name, req.Item.CspId}

	// Call common-runtime API
	result, err := cmrt.RegisterVM(req.ConnectionName, userIId)
	if err != nil {
		return nil, gc.ConvGrpcStatusErr(err, "", "CCMService.RegisterVM()")
	}

	// CCM 객체에서 GRPC 메시지로 복사
	var grpcObj pb.VMInfo
	err = gc.CopySrcToDest(result, &grpcObj)
	if err != nil {
		return nil, gc.ConvGrpcStatusErr(err, "", "CCMService.RegisterVM()")
	}

	resp := &pb.VMInfoResponse{Item: &grpcObj}
	return resp, nil
}

// UnregisterVM - VM 제거
func (s *CCMService) UnregisterVM(ctx context.Context, req *pb.VMUnregiserQryRequest) (*pb.BooleanResponse, error) {
	logger := logger.NewLogger()

	logger.Debug("calling CCMService.UnregisterVM()")

	// Call common-runtime API
	result, err := cmrt.UnregisterResource(req.ConnectionName, rsVM, req.Name)
	if err != nil {
		return nil, gc.ConvGrpcStatusErr(err, "", "CCMService.UnregisterVM()")
	}

	resp := &pb.BooleanResponse{Result: result}
	return resp, nil
}

// ===== [ Private Functions ] =====

// ===== [ Public Functions ] =====
