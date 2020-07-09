package service

import (
	"context"

	gc "github.com/cloud-barista/cb-spider/api-runtime/grpc-runtime/common"
	"github.com/cloud-barista/cb-spider/api-runtime/grpc-runtime/logger"
	pb "github.com/cloud-barista/cb-spider/api-runtime/grpc-runtime/stub/cbspider"

	rim "github.com/cloud-barista/cb-spider/cloud-info-manager/region-info-manager"
)

// ===== [ Constants and Variables ] =====

// ===== [ Types ] =====

// ===== [ Implementations ] =====

// CreateRegion - Region 생성
func (s *CIMService) CreateRegion(ctx context.Context, req *pb.RegionInfoRequest) (*pb.RegionInfoResponse, error) {
	logger := logger.NewLogger()

	logger.Debug("calling CIMService.CreateRegion()")

	// GRPC 메시지에서 CIM 객체로 복사
	var cimObj rim.RegionInfo
	err := gc.CopySrcToDest(&req.Item, &cimObj)
	if err != nil {
		return nil, gc.ConvGrpcStatusErr(err, "", "CIMService.CreateRegion()")
	}

	regionInfo, err := rim.RegisterRegionInfo(cimObj)
	if err != nil {
		return nil, gc.ConvGrpcStatusErr(err, "", "CIMService.CreateRegion()")
	}

	// CIM 객체에서 GRPC 메시지로 복사
	var grpcObj pb.RegionInfo
	err = gc.CopySrcToDest(&regionInfo, &grpcObj)
	if err != nil {
		return nil, gc.ConvGrpcStatusErr(err, "", "CIMService.CreateRegion()")
	}

	resp := &pb.RegionInfoResponse{Item: &grpcObj}
	return resp, nil
}

// ListRegion - Region 목록
func (s *CIMService) ListRegion(ctx context.Context, req *pb.Empty) (*pb.ListRegionInfoResponse, error) {
	logger := logger.NewLogger()

	logger.Debug("calling CIMService.ListRegion()")

	infoList, err := rim.ListRegion()
	if err != nil {
		return nil, gc.ConvGrpcStatusErr(err, "", "CIMService.ListRegion()")
	}

	// CIM 객체에서 GRPC 메시지로 복사
	var grpcObj []*pb.RegionInfo
	err = gc.CopySrcToDest(&infoList, &grpcObj)
	if err != nil {
		return nil, gc.ConvGrpcStatusErr(err, "", "CIMService.ListRegion()")
	}

	resp := &pb.ListRegionInfoResponse{Items: grpcObj}
	return resp, nil
}

// GetRegion - Region 조회
func (s *CIMService) GetRegion(ctx context.Context, req *pb.RegionQryRequest) (*pb.RegionInfoResponse, error) {
	logger := logger.NewLogger()

	logger.Debug("calling CIMService.GetRegion()")

	regionInfo, err := rim.GetRegion(req.RegionName)
	if err != nil {
		return nil, gc.ConvGrpcStatusErr(err, "", "CIMService.GetRegion()")
	}

	// CIM 객체에서 GRPC 메시지로 복사
	var grpcObj pb.RegionInfo
	err = gc.CopySrcToDest(&regionInfo, &grpcObj)
	if err != nil {
		return nil, gc.ConvGrpcStatusErr(err, "", "CIMService.GetRegion()")
	}

	resp := &pb.RegionInfoResponse{Item: &grpcObj}
	return resp, nil
}

// DeleteRegion - Region 삭제
func (s *CIMService) DeleteRegion(ctx context.Context, req *pb.RegionQryRequest) (*pb.BooleanResponse, error) {
	logger := logger.NewLogger()

	logger.Debug("calling CIMService.DeleteRegion()")

	result, err := rim.UnRegisterRegion(req.RegionName)
	if err != nil {
		return nil, gc.ConvGrpcStatusErr(err, "", "CIMService.DeleteRegion()")
	}

	resp := &pb.BooleanResponse{Result: result}
	return resp, nil
}

// ===== [ Private Functions ] =====

// ===== [ Public Functions ] =====
