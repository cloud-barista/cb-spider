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
