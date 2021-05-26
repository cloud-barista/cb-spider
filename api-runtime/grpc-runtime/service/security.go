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
	"strings"

	cm "github.com/cloud-barista/cb-spider/api-runtime/common-runtime"
	gc "github.com/cloud-barista/cb-spider/api-runtime/grpc-runtime/common"
	"github.com/cloud-barista/cb-spider/api-runtime/grpc-runtime/logger"
	pb "github.com/cloud-barista/cb-spider/api-runtime/grpc-runtime/stub/cbspider"

	cmrt "github.com/cloud-barista/cb-spider/api-runtime/common-runtime"
	cres "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
)

// ===== [ Constants and Variables ] =====

// ===== [ Types ] =====

// ===== [ Implementations ] =====

// CreateSecurity - Security 생성
func (s *CCMService) CreateSecurity(ctx context.Context, req *pb.SecurityCreateRequest) (*pb.SecurityInfoResponse, error) {
	logger := logger.NewLogger()

	logger.Debug("calling CCMService.CreateSecurity()")

	// check the input Name to include the SecurityGroup Delimiter
	if strings.HasPrefix(req.Item.Name, cm.SG_DELIMITER) {
		return nil, gc.NewGrpcStatusErr(cm.SG_DELIMITER+" cannot be used in SecurityGroup name!!", "", "CCMService.CreateSecurity()")
	}

	// GRPC 메시지에서 CCM 객체로 복사
	var reqInfo cres.SecurityReqInfo
	err := gc.CopySrcToDest(&req.Item, &reqInfo)
	if err != nil {
		return nil, gc.ConvGrpcStatusErr(err, "", "CCMService.CreateSecurity()")
	}
	// SG NameID format => {VPC NameID} + cm.SG_DELIMITER + {SG NameID}
	// transform: SG NameID => {VPC NameID} + cm.SG_DELIMITER + {SG NameID}
	//reqInfo.IId = cres.IID{NameId: req.Item.VpcName + cm.SG_DELIMITER + req.Item.Name, SystemId: ""}
	reqInfo.IId = cres.IID{NameId: req.Item.VpcName + cm.SG_DELIMITER + req.Item.Name, SystemId: req.Item.Name} // for NCP: fixed NameID => SystemID, Driver: (1)search systemID with fixed NameID (2)replace fixed NameID into SysemID
	reqInfo.VpcIID = cres.IID{NameId: req.Item.VpcName, SystemId: ""}

	// Call common-runtime API
	result, err := cmrt.CreateSecurity(req.ConnectionName, rsSG, reqInfo)
	if err != nil {
		return nil, gc.ConvGrpcStatusErr(err, "", "CCMService.CreateSecurity()")
	}

	// CCM 객체에서 GRPC 메시지로 복사
	var grpcObj pb.SecurityInfo
	err = gc.CopySrcToDest(result, &grpcObj)
	if err != nil {
		return nil, gc.ConvGrpcStatusErr(err, "", "CCMService.CreateSecurity()")
	}

	resp := &pb.SecurityInfoResponse{Item: &grpcObj}
	return resp, nil
}

// ListSecurity - Security 목록
func (s *CCMService) ListSecurity(ctx context.Context, req *pb.SecurityAllQryRequest) (*pb.ListSecurityInfoResponse, error) {
	logger := logger.NewLogger()

	logger.Debug("calling CCMService.ListSecurity()")

	// Call common-runtime API
	result, err := cmrt.ListSecurity(req.ConnectionName, rsSG)
	if err != nil {
		return nil, gc.ConvGrpcStatusErr(err, "", "CCMService.ListSecurity()")
	}

	// CCM 객체에서 GRPC 메시지로 복사
	var grpcObj []*pb.SecurityInfo
	err = gc.CopySrcToDest(&result, &grpcObj)
	if err != nil {
		return nil, gc.ConvGrpcStatusErr(err, "", "CCMService.ListSecurity()")
	}

	resp := &pb.ListSecurityInfoResponse{Items: grpcObj}
	return resp, nil
}

// GetSecurity - Security 조회
func (s *CCMService) GetSecurity(ctx context.Context, req *pb.SecurityQryRequest) (*pb.SecurityInfoResponse, error) {
	logger := logger.NewLogger()

	logger.Debug("calling CCMService.GetSecurity()")

	// Call common-runtime API
	result, err := cmrt.GetSecurity(req.ConnectionName, rsSG, req.Name)
	if err != nil {
		return nil, gc.ConvGrpcStatusErr(err, "", "CCMService.GetSecurity()")
	}

	// CCM 객체에서 GRPC 메시지로 복사
	var grpcObj pb.SecurityInfo
	err = gc.CopySrcToDest(result, &grpcObj)
	if err != nil {
		return nil, gc.ConvGrpcStatusErr(err, "", "CCMService.GetSecurity()")
	}

	resp := &pb.SecurityInfoResponse{Item: &grpcObj}
	return resp, nil
}

// DeleteSecurity - Security 삭제
func (s *CCMService) DeleteSecurity(ctx context.Context, req *pb.SecurityQryRequest) (*pb.BooleanResponse, error) {
	logger := logger.NewLogger()

	logger.Debug("calling CCMService.DeleteSecurity()")

	// Call common-runtime API
	result, _, err := cmrt.DeleteResource(req.ConnectionName, rsSG, req.Name, req.Force)
	if err != nil {
		return nil, gc.ConvGrpcStatusErr(err, "", "CCMService.DeleteSecurity()")
	}

	resp := &pb.BooleanResponse{Result: result}
	return resp, nil
}

// ListAllSecurity - 관리 Security 목록
func (s *CCMService) ListAllSecurity(ctx context.Context, req *pb.SecurityAllQryRequest) (*pb.AllResourceInfoResponse, error) {
	logger := logger.NewLogger()

	logger.Debug("calling CCMService.ListAllSecurity()")

	// Call common-runtime API
	allResourceList, err := cmrt.ListAllResource(req.ConnectionName, rsSG)
	if err != nil {
		return nil, gc.ConvGrpcStatusErr(err, "", "CCMService.ListAllSecurity()")
	}

	// CCM 객체에서 GRPC 메시지로 복사
	var grpcObj pb.AllResourceInfoResponse
	err = gc.CopySrcToDest(&allResourceList, &grpcObj)
	if err != nil {
		return nil, gc.ConvGrpcStatusErr(err, "", "CCMService.ListAllSecurity()")
	}

	return &grpcObj, nil
}

// DeleteCSPSecurity - CSP Security 삭제
func (s *CCMService) DeleteCSPSecurity(ctx context.Context, req *pb.CSPSecurityQryRequest) (*pb.BooleanResponse, error) {
	logger := logger.NewLogger()

	logger.Debug("calling CCMService.DeleteCSPSecurity()")

	// Call common-runtime API
	result, _, err := cmrt.DeleteCSPResource(req.ConnectionName, rsSG, req.Id)
	if err != nil {
		return nil, gc.ConvGrpcStatusErr(err, "", "CCMService.DeleteCSPSecurity()")
	}

	resp := &pb.BooleanResponse{Result: result}
	return resp, nil
}

// ===== [ Private Functions ] =====

// ===== [ Public Functions ] =====
