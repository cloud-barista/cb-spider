package service

import (
	"context"

	gc "github.com/cloud-barista/cb-spider/api-runtime/grpc-runtime/common"
	"github.com/cloud-barista/cb-spider/api-runtime/grpc-runtime/logger"
	pb "github.com/cloud-barista/cb-spider/api-runtime/grpc-runtime/stub/cbspider"

	cim "github.com/cloud-barista/cb-spider/cloud-info-manager/credential-info-manager"
)

// ===== [ Constants and Variables ] =====

// ===== [ Types ] =====

// ===== [ Implementations ] =====

// CreateCredential - Credential 생성
func (s *CIMService) CreateCredential(ctx context.Context, req *pb.CredentialInfoRequest) (*pb.CredentialInfoResponse, error) {
	logger := logger.NewLogger()

	logger.Debug("calling CIMService.CreateCredential()")

	// GRPC 메시지에서 CIM 객체로 복사
	var cimObj cim.CredentialInfo
	err := gc.CopySrcToDest(&req.Item, &cimObj)
	if err != nil {
		return nil, gc.ConvGrpcStatusErr(err, "", "CIMService.CreateCredential()")
	}

	crdInfo, err := cim.RegisterCredentialInfo(cimObj)
	if err != nil {
		return nil, gc.ConvGrpcStatusErr(err, "", "CIMService.CreateCredential()")
	}

	// CIM 객체에서 GRPC 메시지로 복사
	var grpcObj pb.CredentialInfo
	err = gc.CopySrcToDest(&crdInfo, &grpcObj)
	if err != nil {
		return nil, gc.ConvGrpcStatusErr(err, "", "CIMService.CreateCredential()")
	}

	resp := &pb.CredentialInfoResponse{Item: &grpcObj}
	return resp, nil
}

// ListCredential - Credential 목록
func (s *CIMService) ListCredential(ctx context.Context, req *pb.Empty) (*pb.ListCredentialInfoResponse, error) {
	logger := logger.NewLogger()

	logger.Debug("calling CIMService.ListCredential()")

	infoList, err := cim.ListCredential()
	if err != nil {
		return nil, gc.ConvGrpcStatusErr(err, "", "CIMService.ListCredential()")
	}

	// CIM 객체에서 GRPC 메시지로 복사
	var grpcObj []*pb.CredentialInfo
	err = gc.CopySrcToDest(&infoList, &grpcObj)
	if err != nil {
		return nil, gc.ConvGrpcStatusErr(err, "", "CIMService.ListCredential()")
	}

	resp := &pb.ListCredentialInfoResponse{Items: grpcObj}
	return resp, nil
}

// GetCredential - Credential 조회
func (s *CIMService) GetCredential(ctx context.Context, req *pb.CredentialQryRequest) (*pb.CredentialInfoResponse, error) {
	logger := logger.NewLogger()

	logger.Debug("calling CIMService.GetCredential()")

	crdInfo, err := cim.GetCredential(req.CredentialName)
	if err != nil {
		return nil, gc.ConvGrpcStatusErr(err, "", "CIMService.GetCredential()")
	}

	// CIM 객체에서 GRPC 메시지로 복사
	var grpcObj pb.CredentialInfo
	err = gc.CopySrcToDest(&crdInfo, &grpcObj)
	if err != nil {
		return nil, gc.ConvGrpcStatusErr(err, "", "CIMService.GetCredential()")
	}

	resp := &pb.CredentialInfoResponse{Item: &grpcObj}
	return resp, nil
}

// DeleteCredential - Credential 삭제
func (s *CIMService) DeleteCredential(ctx context.Context, req *pb.CredentialQryRequest) (*pb.BooleanResponse, error) {
	logger := logger.NewLogger()

	logger.Debug("calling CIMService.DeleteCredential()")

	result, err := cim.UnRegisterCredential(req.CredentialName)
	if err != nil {
		return nil, gc.ConvGrpcStatusErr(err, "", "CIMService.DeleteCredential()")
	}

	resp := &pb.BooleanResponse{Result: result}
	return resp, nil
}

// ===== [ Private Functions ] =====

// ===== [ Public Functions ] =====
