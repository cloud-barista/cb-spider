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

// CreateKey - KeyPair 생성
func (s *CCMService) CreateKey(ctx context.Context, req *pb.KeyPairCreateRequest) (*pb.KeyPairInfoResponse, error) {
	logger := logger.NewLogger()

	logger.Debug("calling CCMService.CreateKey()")

	// Grpc RegInfo => Driver ReqInfo
	reqInfo := cres.KeyPairReqInfo{
		IId: cres.IID{NameId: req.Item.Name, SystemId: ""},
	}

	// Call common-runtime API
	result, err := cmrt.CreateKey(req.ConnectionName, rsKey, reqInfo)
	if err != nil {
		return nil, gc.ConvGrpcStatusErr(err, "", "CCMService.CreateKey()")
	}

	// CCM 객체에서 GRPC 메시지로 복사
	var grpcObj pb.KeyPairInfo
	err = gc.CopySrcToDest(result, &grpcObj)
	if err != nil {
		return nil, gc.ConvGrpcStatusErr(err, "", "CCMService.CreateKey()")
	}

	resp := &pb.KeyPairInfoResponse{Item: &grpcObj}
	return resp, nil
}

// ListKey - KeyPair 목록
func (s *CCMService) ListKey(ctx context.Context, req *pb.KeyPairAllQryRequest) (*pb.ListKeyPairInfoResponse, error) {
	logger := logger.NewLogger()

	logger.Debug("calling CCMService.ListKey()")

	// Call common-runtime API
	result, err := cmrt.ListKey(req.ConnectionName, rsKey)
	if err != nil {
		return nil, gc.ConvGrpcStatusErr(err, "", "CCMService.ListKey()")
	}

	// CCM 객체에서 GRPC 메시지로 복사
	var grpcObj []*pb.KeyPairInfo
	err = gc.CopySrcToDest(&result, &grpcObj)
	if err != nil {
		return nil, gc.ConvGrpcStatusErr(err, "", "CCMService.ListKey()")
	}

	resp := &pb.ListKeyPairInfoResponse{Items: grpcObj}
	return resp, nil
}

// GetKey - KeyPair 조회
func (s *CCMService) GetKey(ctx context.Context, req *pb.KeyPairQryRequest) (*pb.KeyPairInfoResponse, error) {
	logger := logger.NewLogger()

	logger.Debug("calling CCMService.GetKey()")

	// Call common-runtime API
	result, err := cmrt.GetKey(req.ConnectionName, rsKey, req.Name)
	if err != nil {
		return nil, gc.ConvGrpcStatusErr(err, "", "CCMService.GetKey()")
	}

	// CCM 객체에서 GRPC 메시지로 복사
	var grpcObj pb.KeyPairInfo
	err = gc.CopySrcToDest(result, &grpcObj)
	if err != nil {
		return nil, gc.ConvGrpcStatusErr(err, "", "CCMService.GetKey()")
	}

	resp := &pb.KeyPairInfoResponse{Item: &grpcObj}
	return resp, nil
}

// DeleteKey - KeyPair 삭제
func (s *CCMService) DeleteKey(ctx context.Context, req *pb.KeyPairQryRequest) (*pb.BooleanResponse, error) {
	logger := logger.NewLogger()

	logger.Debug("calling CCMService.DeleteKey()")

	// Call common-runtime API
	result, _, err := cmrt.DeleteResource(req.ConnectionName, rsKey, req.Name, req.Force)
	if err != nil {
		return nil, gc.ConvGrpcStatusErr(err, "", "CCMService.DeleteKey()")
	}

	resp := &pb.BooleanResponse{Result: result}
	return resp, nil
}

// ListAllKey - 관리 Key 목록
func (s *CCMService) ListAllKey(ctx context.Context, req *pb.KeyPairAllQryRequest) (*pb.AllResourceInfoResponse, error) {
	logger := logger.NewLogger()

	logger.Debug("calling CCMService.ListAllKey()")

	// Call common-runtime API
	allResourceList, err := cmrt.ListAllResource(req.ConnectionName, rsKey)
	if err != nil {
		return nil, gc.ConvGrpcStatusErr(err, "", "CCMService.ListAllKey()")
	}

	// CCM 객체에서 GRPC 메시지로 복사
	var grpcObj pb.AllResourceInfoResponse
	err = gc.CopySrcToDest(&allResourceList, &grpcObj)
	if err != nil {
		return nil, gc.ConvGrpcStatusErr(err, "", "CCMService.ListAllKey()")
	}

	return &grpcObj, nil
}

// DeleteCSPKey - CSP Key 삭제
func (s *CCMService) DeleteCSPKey(ctx context.Context, req *pb.CSPKeyPairQryRequest) (*pb.BooleanResponse, error) {
	logger := logger.NewLogger()

	logger.Debug("calling CCMService.DeleteCSPKey()")

	// Call common-runtime API
	result, _, err := cmrt.DeleteCSPResource(req.ConnectionName, rsKey, req.Id)
	if err != nil {
		return nil, gc.ConvGrpcStatusErr(err, "", "CCMService.DeleteCSPKey()")
	}

	resp := &pb.BooleanResponse{Result: result}
	return resp, nil
}

// ===== [ Private Functions ] =====

// ===== [ Public Functions ] =====
