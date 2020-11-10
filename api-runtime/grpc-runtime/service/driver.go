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

	dim "github.com/cloud-barista/cb-spider/cloud-info-manager/driver-info-manager"
)

// ===== [ Constants and Variables ] =====

// ===== [ Types ] =====

// ===== [ Implementations ] =====

// CreateCloudDriver - Cloud Driver 생성
func (s *CIMService) CreateCloudDriver(ctx context.Context, req *pb.CloudDriverInfoRequest) (*pb.CloudDriverInfoResponse, error) {
	logger := logger.NewLogger()

	logger.Debug("calling CIMService.CreateCloudDriver()")

	// GRPC 메시지에서 CIM 객체로 복사
	var cimObj dim.CloudDriverInfo
	err := gc.CopySrcToDest(&req.Item, &cimObj)
	if err != nil {
		return nil, gc.ConvGrpcStatusErr(err, "", "CIMService.CreateCloudDriver()")
	}

	drvInfo, err := dim.RegisterCloudDriverInfo(cimObj)
	if err != nil {
		return nil, gc.ConvGrpcStatusErr(err, "", "CIMService.CreateCloudDriver()")
	}

	// CIM 객체에서 GRPC 메시지로 복사
	var grpcObj pb.CloudDriverInfo
	err = gc.CopySrcToDest(&drvInfo, &grpcObj)
	if err != nil {
		return nil, gc.ConvGrpcStatusErr(err, "", "CIMService.CreateCloudDriver()")
	}

	resp := &pb.CloudDriverInfoResponse{Item: &grpcObj}
	return resp, nil
}

// ListCloudDriver -Cloud Driver 목록
func (s *CIMService) ListCloudDriver(ctx context.Context, req *pb.Empty) (*pb.ListCloudDriverInfoResponse, error) {
	logger := logger.NewLogger()

	logger.Debug("calling CIMService.ListCloudDriver()")

	infoList, err := dim.ListCloudDriver()
	if err != nil {
		return nil, gc.ConvGrpcStatusErr(err, "", "CIMService.ListCloudDriver()")
	}

	// CIM 객체에서 GRPC 메시지로 복사
	var grpcObj []*pb.CloudDriverInfo
	err = gc.CopySrcToDest(&infoList, &grpcObj)
	if err != nil {
		return nil, gc.ConvGrpcStatusErr(err, "", "CIMService.ListCloudDriver()")
	}

	resp := &pb.ListCloudDriverInfoResponse{Items: grpcObj}
	return resp, nil
}

// GetCloudDriver - Cloud Driver 조회
func (s *CIMService) GetCloudDriver(ctx context.Context, req *pb.CloudDriverQryRequest) (*pb.CloudDriverInfoResponse, error) {
	logger := logger.NewLogger()

	logger.Debug("calling CIMService.GetCloudDriver()")

	drvInfo, err := dim.GetCloudDriver(req.DriverName)
	if err != nil {
		return nil, gc.ConvGrpcStatusErr(err, "", "CIMService.GetCloudDriver()")
	}

	// CIM 객체에서 GRPC 메시지로 복사
	var grpcObj pb.CloudDriverInfo
	err = gc.CopySrcToDest(&drvInfo, &grpcObj)
	if err != nil {
		return nil, gc.ConvGrpcStatusErr(err, "", "CIMService.GetCloudDriver()")
	}

	resp := &pb.CloudDriverInfoResponse{Item: &grpcObj}
	return resp, nil
}

// DeleteCloudDriver - Cloud Driver 삭제
func (s *CIMService) DeleteCloudDriver(ctx context.Context, req *pb.CloudDriverQryRequest) (*pb.BooleanResponse, error) {
	logger := logger.NewLogger()

	logger.Debug("calling CIMService.DeleteCloudDriver()")

	result, err := dim.UnRegisterCloudDriver(req.DriverName)
	if err != nil {
		return nil, gc.ConvGrpcStatusErr(err, "", "CIMService.DeleteCloudDriver()")
	}

	resp := &pb.BooleanResponse{Result: result}
	return resp, nil
}

// ===== [ Private Functions ] =====

// ===== [ Public Functions ] =====
