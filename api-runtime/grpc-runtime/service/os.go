package service

import (
	"context"

	"github.com/cloud-barista/cb-spider/api-runtime/grpc-runtime/logger"
	pb "github.com/cloud-barista/cb-spider/api-runtime/grpc-runtime/stub/cbspider"

	im "github.com/cloud-barista/cb-spider/cloud-info-manager"
)

// ===== [ Constants and Variables ] =====

// ===== [ Types ] =====

// ===== [ Implementations ] =====

// ListCloudOS - Cloud OS 조회
func (s *CIMService) ListCloudOS(ctx context.Context, req *pb.Empty) (*pb.ListCloudOSInfoResponse, error) {
	logger := logger.NewLogger()

	logger.Debug("calling CIMService.ListCloudOS()")

	infoList := im.ListCloudOS()

	resp := &pb.ListCloudOSInfoResponse{Items: infoList}
	return resp, nil
}

// ===== [ Private Functions ] =====

// ===== [ Public Functions ] =====
