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
	"errors"

	gc "github.com/cloud-barista/cb-spider/api-runtime/grpc-runtime/common"
	"github.com/cloud-barista/cb-spider/api-runtime/grpc-runtime/logger"
	pb "github.com/cloud-barista/cb-spider/api-runtime/grpc-runtime/stub/cbspider"

	cmrt "github.com/cloud-barista/cb-spider/api-runtime/common-runtime"
)

// ===== [ Constants and Variables ] =====

// ===== [ Types ] =====

// ===== [ Implementations ] =====

// CreateImage - Image 생성
func (s *CCMService) CreateImage(ctx context.Context, req *pb.ImageCreateRequest) (*pb.ImageInfoResponse, error) {
	logger := logger.NewLogger()

	logger.Debug("calling CCMService.CreateImage()")

	return nil, gc.ConvGrpcStatusErr(errors.New("Unsupported API"), "", "CCMService.CreateImage()")
}

// ListImage - Image 목록
func (s *CCMService) ListImage(ctx context.Context, req *pb.ImageAllQryRequest) (*pb.ListImageInfoResponse, error) {
	logger := logger.NewLogger()

	logger.Debug("calling CCMService.ListImage()")

	// Call common-runtime API
	result, err := cmrt.ListImage(req.ConnectionName, rsImage)
	if err != nil {
		return nil, gc.ConvGrpcStatusErr(err, "", "CCMService.ListImage()")
	}

	// CCM 객체에서 GRPC 메시지로 복사
	var grpcObj []*pb.ImageInfo
	err = gc.CopySrcToDest(&result, &grpcObj)
	if err != nil {
		return nil, gc.ConvGrpcStatusErr(err, "", "CCMService.ListImage()")
	}

	resp := &pb.ListImageInfoResponse{Items: grpcObj}
	return resp, nil
}

// GetImage - Image 조회
func (s *CCMService) GetImage(ctx context.Context, req *pb.ImageQryRequest) (*pb.ImageInfoResponse, error) {
	logger := logger.NewLogger()

	logger.Debug("calling CCMService.GetImage()")

	// Call common-runtime API
	result, err := cmrt.GetImage(req.ConnectionName, rsImage, req.Name)
	if err != nil {
		return nil, gc.ConvGrpcStatusErr(err, "", "CCMService.GetImage()")
	}

	// CCM 객체에서 GRPC 메시지로 복사
	var grpcObj pb.ImageInfo
	err = gc.CopySrcToDest(result, &grpcObj)
	if err != nil {
		return nil, gc.ConvGrpcStatusErr(err, "", "CCMService.GetImage()")
	}

	resp := &pb.ImageInfoResponse{Item: &grpcObj}
	return resp, nil
}

// DeleteImage - Image 삭제
func (s *CCMService) DeleteImage(ctx context.Context, req *pb.ImageQryRequest) (*pb.BooleanResponse, error) {
	logger := logger.NewLogger()

	logger.Debug("calling CCMService.DeleteImage()")

	return nil, gc.ConvGrpcStatusErr(errors.New("Unsupported API"), "", "CCMService.DeleteImage()")
}

// ===== [ Private Functions ] =====

// ===== [ Public Functions ] =====
