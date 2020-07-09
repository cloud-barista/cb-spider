package service

import (
	"context"

	gc "github.com/cloud-barista/cb-spider/api-runtime/grpc-runtime/common"
	"github.com/cloud-barista/cb-spider/api-runtime/grpc-runtime/logger"
	pb "github.com/cloud-barista/cb-spider/api-runtime/grpc-runtime/stub/cbspider"

	cmrt "github.com/cloud-barista/cb-spider/api-runtime/common-runtime"
)

// ===== [ Constants and Variables ] =====

// ===== [ Types ] =====

// ===== [ Implementations ] =====

// ListVMSpec - VM Spec 목록
func (s *CCMService) ListVMSpec(ctx context.Context, req *pb.VMSpecAllQryRequest) (*pb.ListVMSpecInfoResponse, error) {
	logger := logger.NewLogger()

	logger.Debug("calling CCMService.ListVMSpec()")

	// Call common-runtime API
	result, err := cmrt.ListVMSpec(req.ConnectionName)
	if err != nil {
		return nil, gc.ConvGrpcStatusErr(err, "", "CCMService.ListVMSpec()")
	}

	// CCM 객체에서 GRPC 메시지로 복사
	var grpcObj []*pb.VMSpecInfo
	err = gc.CopySrcToDest(&result, &grpcObj)
	if err != nil {
		return nil, gc.ConvGrpcStatusErr(err, "", "CCMService.ListVMSpec()")
	}

	resp := &pb.ListVMSpecInfoResponse{Items: grpcObj}
	return resp, nil
}

// GetVMSpec - VM Spec 조회
func (s *CCMService) GetVMSpec(ctx context.Context, req *pb.VMSpecQryRequest) (*pb.VMSpecInfoResponse, error) {
	logger := logger.NewLogger()

	logger.Debug("calling CCMService.GetVMSpec()")

	// Call common-runtime API
	result, err := cmrt.GetVMSpec(req.ConnectionName, req.Name)
	if err != nil {
		return nil, gc.ConvGrpcStatusErr(err, "", "CCMService.GetVMSpec()")
	}

	// CCM 객체에서 GRPC 메시지로 복사
	var grpcObj pb.VMSpecInfo
	err = gc.CopySrcToDest(result, &grpcObj)
	if err != nil {
		return nil, gc.ConvGrpcStatusErr(err, "", "CCMService.GetVMSpec()")
	}

	resp := &pb.VMSpecInfoResponse{Item: &grpcObj}
	return resp, nil
}

// ListOrgVMSpec - 클라우드의 원래 VM Spec 목록
func (s *CCMService) ListOrgVMSpec(ctx context.Context, req *pb.VMSpecAllQryRequest) (*pb.StringResponse, error) {
	logger := logger.NewLogger()

	logger.Debug("calling CCMService.ListOrgVMSpec()")

	// Call common-runtime API
	result, err := cmrt.ListOrgVMSpec(req.ConnectionName)
	if err != nil {
		return nil, gc.ConvGrpcStatusErr(err, "", "CCMService.ListOrgVMSpec()")
	}

	resp := &pb.StringResponse{Result: result}

	return resp, nil
}

// GetOrgVMSpec - 클라우드의 원래 VM Spec 조회
func (s *CCMService) GetOrgVMSpec(ctx context.Context, req *pb.VMSpecQryRequest) (*pb.StringResponse, error) {
	logger := logger.NewLogger()

	logger.Debug("calling CCMService.GetOrgVMSpec()")

	// Call common-runtime API
	result, err := cmrt.GetOrgVMSpec(req.ConnectionName, req.Name)
	if err != nil {
		return nil, gc.ConvGrpcStatusErr(err, "", "CCMService.GetOrgVMSpec()")
	}

	resp := &pb.StringResponse{Result: result}

	return resp, nil
}

// ===== [ Private Functions ] =====

// ===== [ Public Functions ] =====
