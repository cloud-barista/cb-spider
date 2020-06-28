package service

import (
	"context"

	gc "github.com/cloud-barista/cb-spider/api-runtime/grpc-runtime/common"
	"github.com/cloud-barista/cb-spider/api-runtime/grpc-runtime/logger"
	pb "github.com/cloud-barista/cb-spider/api-runtime/grpc-runtime/stub/cbspider"

	ccim "github.com/cloud-barista/cb-spider/cloud-info-manager/connection-config-info-manager"
)

// ===== [ Constants and Variables ] =====

// ===== [ Types ] =====

// ===== [ Implementations ] =====

// CreateConnectionConfig - Connection Config 생성
func (s *CIMService) CreateConnectionConfig(ctx context.Context, req *pb.ConnectionConfigInfoRequest) (*pb.ConnectionConfigInfoResponse, error) {
	logger := logger.NewLogger()

	logger.Debug("calling CIMService.CreateConnectionConfig()")

	// GRPC 메시지에서 CIM 객체로 복사
	var cimObj ccim.ConnectionConfigInfo
	err := gc.CopySrcToDest(&req.Item, &cimObj)
	if err != nil {
		return nil, gc.ConvGrpcStatusErr(err, "", "CIMService.CreateConnectionConfig()")
	}

	connInfo, err := ccim.CreateConnectionConfigInfo(cimObj)
	if err != nil {
		return nil, gc.ConvGrpcStatusErr(err, "", "CIMService.CreateConnectionConfig()")
	}

	// CIM 객체에서 GRPC 메시지로 복사
	var grpcObj pb.ConnectionConfigInfo
	err = gc.CopySrcToDest(&connInfo, &grpcObj)
	if err != nil {
		return nil, gc.ConvGrpcStatusErr(err, "", "CIMService.CreateConnectionConfig()")
	}

	resp := &pb.ConnectionConfigInfoResponse{Item: &grpcObj}
	return resp, nil
}

// ListConnectionConfig - Connection Config 목록
func (s *CIMService) ListConnectionConfig(ctx context.Context, req *pb.Empty) (*pb.ListConnectionConfigInfoResponse, error) {
	logger := logger.NewLogger()

	logger.Debug("calling CIMService.ListConnectionConfig()")

	infoList, err := ccim.ListConnectionConfig()
	if err != nil {
		return nil, gc.ConvGrpcStatusErr(err, "", "CIMService.ListConnectionConfig()")
	}

	// CIM 객체에서 GRPC 메시지로 복사
	var grpcObj []*pb.ConnectionConfigInfo
	err = gc.CopySrcToDest(&infoList, &grpcObj)
	if err != nil {
		return nil, gc.ConvGrpcStatusErr(err, "", "CIMService.ListConnectionConfig()")
	}

	resp := &pb.ListConnectionConfigInfoResponse{Items: grpcObj}
	return resp, nil
}

// GetConnectionConfig - Connection Config 조회
func (s *CIMService) GetConnectionConfig(ctx context.Context, req *pb.ConnectionConfigQryRequest) (*pb.ConnectionConfigInfoResponse, error) {
	logger := logger.NewLogger()

	logger.Debug("calling CIMService.GetConnectionConfig()")

	connInfo, err := ccim.GetConnectionConfig(req.ConfigName)
	if err != nil {
		return nil, gc.ConvGrpcStatusErr(err, "", "CIMService.GetConnectionConfig()")
	}

	// CIM 객체에서 GRPC 메시지로 복사
	var grpcObj pb.ConnectionConfigInfo
	err = gc.CopySrcToDest(&connInfo, &grpcObj)
	if err != nil {
		return nil, gc.ConvGrpcStatusErr(err, "", "CIMService.GetConnectionConfig()")
	}

	resp := &pb.ConnectionConfigInfoResponse{Item: &grpcObj}
	return resp, nil
}

// DeleteConnectionConfig - Connection Config 삭제
func (s *CIMService) DeleteConnectionConfig(ctx context.Context, req *pb.ConnectionConfigQryRequest) (*pb.BooleanResponse, error) {
	logger := logger.NewLogger()

	logger.Debug("calling CIMService.DeleteConnectionConfig()")

	result, err := ccim.DeleteConnectionConfig(req.ConfigName)
	if err != nil {
		return nil, gc.ConvGrpcStatusErr(err, "", "CIMService.DeleteConnectionConfig()")
	}

	resp := &pb.BooleanResponse{Result: result}
	return resp, nil
}

// ===== [ Private Functions ] =====

// ===== [ Public Functions ] =====
