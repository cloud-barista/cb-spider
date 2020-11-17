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
	"context"
	"os"
	"os/signal"
	"syscall"
	"fmt"
	"time"
	"strconv"

	"google.golang.org/grpc"
        "google.golang.org/grpc/reflection"

	cblog "github.com/cloud-barista/cb-log"
	cr "github.com/cloud-barista/cb-spider/api-runtime/common-runtime"
	"github.com/cloud-barista/cb-spider/api-runtime/meerkat-runtime/momkat"
        common "github.com/cloud-barista/cb-spider/api-runtime/meerkat-runtime/common"
	th "github.com/cloud-barista/cb-spider/api-runtime/meerkat-runtime/table-handler"
)

var myServerID string
const ( port = ":4096")

type server struct{}


func init() {
        myServerID = cr.HostIPorName + port + "-" +  cr.MiddleStartTime
}

func (s *server) GetChildStatus(ctx context.Context, in *common.Empty) (*common.Status, error) {
        return getStatus()
}

func getStatus() (*common.Status, error) {
        status := "L"
	time := GetCurrentTime()
	return &common.Status{ServerID: myServerID, Status: status, Time: time}, nil
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
        common.RegisterChildStatusServer(s, &server{})
        // Register reflection service on gRPC server.
        reflection.Register(s)

	spiderBanner(cr.HostIPorName + port)

	// register this server status into SpiderHub's registry
	strY := itsMe()

	// for Ctrl+C signal
	setupSigHandler(strY)

	defer clearCheckBit(strY)

	go func() {
		// to wait this server listening
		time.Sleep(time.Millisecond*10)
		momkat.CheckChildKatAndSet()
	}()

        if err := s.Serve(lis); err != nil {
                cblogger.Errorf("failed to serve: %v", err)
        }
}

func setupSigHandler(strY string) {
        cblogger := cblog.GetLogger("CB-SPIDER")

	c := make(chan os.Signal)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		cblogger.Info("\r- Ctrl+C pressed in Terminal")
		clearCheckBit(strY)
		os.Exit(0)
	}()
}

func spiderBanner(server string) {
	gRPCServer := "Meer-Kat: grpc://" +  server
        fmt.Printf("     - %s\n", gRPCServer)
}

// 1. get this server status info
// 2. find the first free row
// 3. set the check bit with '1'
// 4. write this sever inial status info into spiderhub registry
// 5. return the number of Y
func itsMe() string {
        cblogger := cblog.GetLogger("CB-SPIDER")

	status, err := getStatus()
	if err != nil {
		cblogger.Fatalf("could not Fetch Resource Status Information: %v", err)
	}

// @todo get LCK (maybe distributed LCK like zookeeper)
	strY := findFreeRow()
	setCheckBit(strY)
// @todo relese LCK

	cblogger.Info("[" + status.ServerID + "] " + status.Status + "-" + status.Time)
	statusInfo := common.StatusInfo{RowNumber:strY, ServerID:status.ServerID, Status:status.Status, Time:status.Time, Count:"1"}
	err = common.WriteStatusInfo(&statusInfo)
	if err != nil {
		cblogger.Errorf("could not write Cell: %v", err)
	}
	return strY
}

func findFreeRow() string {
        cblogger := cblog.GetLogger("CB-SPIDER")

	srv, err := common.GetTableHandler()
        if err != nil {
                cblogger.Fatalf("Unable to retrieve Sheets client: %v", err)
        }

	max := common.GetMaxSpiders()
	if max == -1 {
		return "";
	}
	for i:=0;i<max;i++ {
		intY, _ := strconv.Atoi(common.StatusTableY)
		intY += i
		strY := strconv.Itoa(intY) 
		value, err := th.ReadCell(srv, &th.Cell{Sheet:common.StatusSheetName, X:common.StatusRowLockX, Y:strY})
		if err != nil {
			cblogger.Errorf("could not read Cell: %v", err)
			break;
		}
		if value != "1" {
			return strY
		}
	}
	cblogger.Error("no free space in the Status Table")
	return "" 
}

func setCheckBit(strY string) error {
	cblogger := cblog.GetLogger("CB-SPIDER")
	srv, err := common.GetTableHandler()
        if err != nil {
                cblogger.Fatalf("Unable to retrieve Sheets client: %v", err)
        }

        err = th.WriteCell(srv, &th.Cell{Sheet:common.StatusSheetName, X:common.StatusRowLockX, Y:strY},  "1")
        return err
}

func clearCheckBit(strY string) error {
	cblogger := cblog.GetLogger("CB-SPIDER")

	srv, err := common.GetTableHandler()
        if err != nil {
                cblogger.Fatalf("Unable to retrieve Sheets client: %v", err)
        }

        err = th.WriteCell(srv, &th.Cell{Sheet:common.StatusSheetName, X:common.StatusRowLockX, Y:strY},  "")
        return err
}

