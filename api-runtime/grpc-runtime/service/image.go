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

// CreateImage - Image 생성
func (s *CCMService) CreateImage(ctx context.Context, req *pb.ImageCreateRequest) (*pb.ImageInfoResponse, error) {
	logger := logger.NewLogger()

	logger.Debug("calling CCMService.CreateImage()")

	// Grpc RegInfo => Driver ReqInfo
	reqInfo := cres.ImageReqInfo{
		IId: cres.IID{NameId: req.Item.Name, SystemId: ""},
	}

	// Call common-runtime API
	result, err := cmrt.CreateImage(req.ConnectionName, rsImage, reqInfo)
	if err != nil {
		return nil, gc.ConvGrpcStatusErr(err, "", "CCMService.CreateImage()")
	}

	// CCM 객체에서 GRPC 메시지로 복사
	var grpcObj pb.ImageInfo
	err = gc.CopySrcToDest(result, &grpcObj)
	if err != nil {
		return nil, gc.ConvGrpcStatusErr(err, "", "CCMService.CreateImage()")
	}

	resp := &pb.ImageInfoResponse{Item: &grpcObj}
	return resp, nil
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

	// Call common-runtime API
	result, err := cmrt.DeleteImage(req.ConnectionName, rsImage, req.Name)
	if err != nil {
		return nil, gc.ConvGrpcStatusErr(err, "", "CCMService.DeleteImage()")
	}

	resp := &pb.BooleanResponse{Result: result}
	return resp, nil
}

// ===== [ Private Functions ] =====

// ===== [ Public Functions ] =====
