// gRPC Runtime of CB-Spider.
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// by CB-Spider Team, 2020.09.

package grpcruntime

import (
	"errors"
	"fmt"
	"net"
	"os"

	cr "github.com/cloud-barista/cb-spider/api-runtime/common-runtime"
	gc "github.com/cloud-barista/cb-spider/api-runtime/grpc-runtime/common"
	"github.com/cloud-barista/cb-spider/api-runtime/grpc-runtime/config"
	"github.com/cloud-barista/cb-spider/api-runtime/grpc-runtime/logger"
	grpc_service "github.com/cloud-barista/cb-spider/api-runtime/grpc-runtime/service"
	pb "github.com/cloud-barista/cb-spider/api-runtime/grpc-runtime/stub/cbspider"

	"google.golang.org/grpc/reflection"
)

// RunServer - GRPC 서버 실행
func RunServer() {
	logger := logger.NewLogger()

        cbspiderRoot := os.Getenv("CBSPIDER_ROOT")
        if cbspiderRoot == "" {
                logger.Error("$CBSPIDER_ROOT is not set!!")
                os.Exit(1)
        }
	configPath := cbspiderRoot + "/conf/grpc_conf.yaml"
	gConf, err := configLoad(configPath)
	if err != nil {
		logger.Error("failed to load config : ", err)
		return
	}

	spidersrv := gConf.GSL.SpiderSrv

	conn, err := net.Listen("tcp", spidersrv.Addr)
	if err != nil {
		logger.Error("failed to listen: ", err)
		return
	}

	cbserver, closer, err := gc.NewCBServer(spidersrv)
	if err != nil {
		logger.Error("failed to create grpc server: ", err)
		return
	}

	if closer != nil {
		defer closer.Close()
	}

	gs := cbserver.Server
	pb.RegisterCIMServer(gs, &grpc_service.CIMService{})
	pb.RegisterCCMServer(gs, &grpc_service.CCMService{})
	pb.RegisterSSHServer(gs, &grpc_service.SSHService{})

	if spidersrv.Reflection == "enable" {
		if spidersrv.Interceptors.AuthJWT != nil {
			fmt.Printf("\n\n*** you can run reflection when jwt auth interceptor is not used ***\n\n")
		} else {
			reflection.Register(gs)
		}
	}

	//fmt.Printf("\n\n => grpc server started on %s\n\n", spidersrv.Addr)
	spiderBanner(cr.HostIPorName + spidersrv.Addr)

	if err := gs.Serve(conn); err != nil {
		logger.Error("failed to serve: ", err)
	}
}

func spiderBanner(server string) {
	gRPCServer := "Go   API: grpc://" +  server
        fmt.Printf("     - %s\n", gRPCServer)
}

func configLoad(cf string) (config.GrpcConfig, error) {
	logger := logger.NewLogger()

	// Viper 를 사용하는 설정 파서 생성
	parser := config.MakeParser()

	var (
		gConf config.GrpcConfig
		err   error
	)

	if cf == "" {
		logger.Error("Please, provide the path to your configuration file")
		return gConf, errors.New("configuration file are not specified")
	}

	logger.Debug("Parsing configuration file: ", cf)
	if gConf, err = parser.GrpcParse(cf); err != nil {
		logger.Error("ERROR - Parsing the configuration file.\n", err.Error())
		return gConf, err
	}

	// Command line 에 지정된 옵션을 설정에 적용 (우선권)

	// SPIDER 필수 입력 항목 체크
	spidersrv := gConf.GSL.SpiderSrv

	if spidersrv == nil {
		return gConf, errors.New("spidersrv field are not specified")
	}

	if spidersrv.Addr == "" {
		return gConf, errors.New("spidersrv.addr field are not specified")
	}

	if spidersrv.TLS != nil {
		if spidersrv.TLS.TLSCert == "" {
			return gConf, errors.New("spidersrv.tls.tls_cert field are not specified")
		}
		if spidersrv.TLS.TLSKey == "" {
			return gConf, errors.New("spidersrv.tls.tls_key field are not specified")
		}
	}

	if spidersrv.Interceptors != nil {
		if spidersrv.Interceptors.AuthJWT != nil {
			if spidersrv.Interceptors.AuthJWT.JWTKey == "" {
				return gConf, errors.New("spidersrv.interceptors.auth_jwt.jwt_key field are not specified")
			}
		}
		if spidersrv.Interceptors.PrometheusMetrics != nil {
			if spidersrv.Interceptors.PrometheusMetrics.ListenPort == 0 {
				return gConf, errors.New("spidersrv.interceptors.prometheus_metrics.listen_port field are not specified")
			}
		}
		if spidersrv.Interceptors.Opentracing != nil {
			if spidersrv.Interceptors.Opentracing.Jaeger != nil {
				if spidersrv.Interceptors.Opentracing.Jaeger.Endpoint == "" {
					return gConf, errors.New("spidersrv.interceptors.opentracing.jaeger.endpoint field are not specified")
				}
			}
		}
	}

	return gConf, nil
}
