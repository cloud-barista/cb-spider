// MeerKat Runtime of CB-Spider.
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// by CB-Spider Team, 2020.11.

package meerkatruntime

import (
        "net"
        "os"
	"context"
	"fmt"
	"time"
	"google.golang.org/grpc"
        "google.golang.org/grpc/reflection"

	cblog "github.com/cloud-barista/cb-log"
	cr "github.com/cloud-barista/cb-spider/api-runtime/common-runtime"
        pb "github.com/cloud-barista/cb-spider/api-runtime/meerkat-runtime/common"
	th "github.com/cloud-barista/cb-spider/api-runtime/meerkat-runtime/table-handler"
)

const (
	port = ":4096"
)

type server struct{}

func (s *server) GetChildStatus(ctx context.Context, in *pb.Empty) (*pb.Status, error) {
        serverID, _ := os.Hostname()
        status := "L"
	time := GetCurrentTime()
        return &pb.Status{ServerID: serverID, Status: status, Time: time}, nil
}

func GetCurrentTime() string {
	currentTime := time.Now()
	return currentTime.Format("2006.01.02 15:04:05 Mon")
}


func RunServer() {
        cblogger := cblog.GetLogger("CB-SPIDER")
	
        lis, err := net.Listen("tcp", port)
        if err != nil {
                cblogger.Errorf("failed to listen: %v", err)
        }
        s := grpc.NewServer()
        pb.RegisterChildStatusServer(s, &server{})
        // Register reflection service on gRPC server.
        reflection.Register(s)

	spiderBanner(cr.HostIPorName + port)

	go check()

        if err := s.Serve(lis); err != nil {
                cblogger.Errorf("failed to serve: %v", err)
        }
}

func spiderBanner(server string) {
	gRPCServer := "Meer-Kat: grpc://" +  server
        fmt.Printf("     - %s\n", gRPCServer)
}

func check() {
        cblogger := cblog.GetLogger("CB-SPIDER")

	client, ctx, err := getClient("localhost:4096")
	if err != nil {
		cblogger.Fatalf("could not get Client: %v", err)
	}

	sum := 1
	for sum < 5 {
		sum += sum

		status, err := client.GetChildStatus(ctx, &pb.Empty{})
		if err != nil {
			cblogger.Fatalf("could not Fetch Resource Status Information: %v", err)
		}
		cblogger.Info("[" + status.ServerID + "] " + status.Status + "-" + status.Time)
		err = writeStatus(status)
		if err != nil {
			cblogger.Error("could not write Cell: %v", err)
		}
		time.Sleep(time.Millisecond*100)
	}
}

func writeStatus(status *pb.Status) error {
        cblogger := cblog.GetLogger("CB-SPIDER")
	srv, err := th.GetHandler()
        if err != nil {
                cblogger.Fatalf("Unable to retrieve Sheets client: %v", err)
        }

	err = th.WriteRange(srv, &th.CellRange{Sheet:"Status", X:"B", Y:"4", X2:"D"},	[]string{status.ServerID, status.Status, status.Time})
	return err
}

func getClient(serverPort string) (pb.ChildStatusClient, context.Context, error)  {
        cblogger := cblog.GetLogger("CB-SPIDER")

	// Set up a connection to the server.
        conn, err := grpc.Dial(serverPort, grpc.WithInsecure())
        if err != nil {
                cblogger.Fatalf("did not connect: %v", err)
        }

        client := pb.NewChildStatusClient(conn)
        ctx, _ := context.WithTimeout(context.Background(), 100*time.Hour)

	return client, ctx, nil
}

