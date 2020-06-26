package service

import (
	"context"
	"strings"

	gc "github.com/cloud-barista/cb-spider/api-runtime/grpc-runtime/common"
	"github.com/cloud-barista/cb-spider/api-runtime/grpc-runtime/logger"
	pb "github.com/cloud-barista/cb-spider/api-runtime/grpc-runtime/stub/cbspider"

	sshrun "github.com/cloud-barista/cb-spider/cloud-control-manager/vm-ssh"
)

// ===== [ Constants and Variables ] =====

// ===== [ Types ] =====

// ===== [ Implementations ] =====

// SSHRun - SSH 실행
func (s *SSHService) SSHRun(ctx context.Context, req *pb.SSHRunRequest) (*pb.StringResponse, error) {
	logger := logger.NewLogger()

	logger.Debug("calling SSHService.SSHRun()")

	strPrivateKey := strings.Join(req.PrivateKey[:], "\n")

	sshInfo := sshrun.SSHInfo{
		UserName:   req.UserName,
		PrivateKey: []byte(strPrivateKey),
		ServerPort: req.ServerPort,
	}

	var result string
	var err error
	if result, err = sshrun.SSHRun(sshInfo, req.Command); err != nil {
		return nil, gc.NewGrpcStatusErr("Error while running cmd: "+req.Command+"]"+err.Error(), "", "SSHService.SSHRun()")
	}

	resp := &pb.StringResponse{Result: result}
	return resp, nil
}

// ===== [ Private Functions ] =====

// ===== [ Public Functions ] =====
