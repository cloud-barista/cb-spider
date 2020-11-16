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
	"strings"
	"google.golang.org/grpc"
        "google.golang.org/grpc/reflection"
	"google.golang.org/api/sheets/v4"

	cblog "github.com/cloud-barista/cb-log"
	cr "github.com/cloud-barista/cb-spider/api-runtime/common-runtime"
        common "github.com/cloud-barista/cb-spider/api-runtime/meerkat-runtime/common"
	th "github.com/cloud-barista/cb-spider/api-runtime/meerkat-runtime/table-handler"
	kv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
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

	go checkAndSet()

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

// @todo get LCK
	strY := findFreeRow()
	setCheckBit(strY)
// @todo relese LCK

	cblogger.Info("[" + status.ServerID + "] " + status.Status + "-" + status.Time)
	err = writeStatus(strY, status)
	if err != nil {
		cblogger.Errorf("could not write Cell: %v", err)
	}
	return strY
}

func findFreeRow() string {
        cblogger := cblog.GetLogger("CB-SPIDER")

	srv, err := getTableHandler()
        if err != nil {
                cblogger.Fatalf("Unable to retrieve Sheets client: %v", err)
        }

	max := getMaxSpiders()
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
	srv, err := getTableHandler()
        if err != nil {
                cblogger.Fatalf("Unable to retrieve Sheets client: %v", err)
        }

        err = th.WriteCell(srv, &th.Cell{Sheet:common.StatusSheetName, X:common.StatusRowLockX, Y:strY},  "1")
        return err
}

func clearCheckBit(strY string) error {
	cblogger := cblog.GetLogger("CB-SPIDER")

	srv, err := getTableHandler()
        if err != nil {
                cblogger.Fatalf("Unable to retrieve Sheets client: %v", err)
        }

        err = th.WriteCell(srv, &th.Cell{Sheet:common.StatusSheetName, X:common.StatusRowLockX, Y:strY},  "")
        return err
}

func getMaxSpiders() int {
	cblogger := cblog.GetLogger("CB-SPIDER")

	result, err := strconv.Atoi(common.MaxSpiders)
        if err != nil {
                cblogger.Error(err)
                return -1
        }

	return result

/* Now, do not use this method because of Sheets Quota Limits.
        srv, err := th.GetHandler()
        if err != nil {
                cblogger.Fatalf("Unable to retrieve Sheets client: %v", err)
        }

        value, err := th.ReadCell(srv, &th.Cell{Sheet:common.StatusSheetName, X:"c", Y:"2"})
        if err != nil {
                cblogger.Errorf("could not read Cell: %v", err)
        }

	result, err := strconv.Atoi(value)
	if err != nil {
                cblogger.Error(err)
		return -1
	}
	return result
*/
}

func checkAndSet() {
        cblogger := cblog.GetLogger("CB-SPIDER")

	// to wait this server listening
	time.Sleep(time.Millisecond*20)


	//childKatList := getChildKatServerList()
	childKatList := getChildKatServerList2()


	for _, kv_childKat:= range childKatList {
		client, ctx, err := getClient(kv_childKat.Value)
		if err != nil {
			cblogger.Errorf("could not get Client: %v", err)
		}

		status, err := client.GetChildStatus(ctx, &common.Empty{})
		if err != nil {
			//cblogger.Errorf("could not Fetch Resource Status Information: %v", err)
			cblogger.Infof("%s: could not Fetch Resource Status Information: %v", kv_childKat.Value, err)

			strStatus := "N"
			time := GetCurrentTime()
			status = &common.Status{ServerID: kv_childKat.Value, Status: strStatus, Time: time}
		}

		cblogger.Info("[" + status.ServerID + "] " + status.Status + "-" + status.Time)
		err = writeStatus(kv_childKat.Key, status)
		if err != nil {
			cblogger.Errorf("could not write Cell: %v", err)
		}
	}
}

// 1. check all Check Bits
// 2. make the list of live children
func getChildKatServerList() []kv.KeyValue {
	cblogger := cblog.GetLogger("CB-SPIDER")

	srv, err := getTableHandler()
        if err != nil {
                cblogger.Fatalf("Unable to retrieve Sheets client: %v", err)
        }

        max := getMaxSpiders()
        if max == -1 {
                return nil;
        }

	childKatList := []kv.KeyValue{}
        for i:=0;i<max;i++ {
                intY, _ := strconv.Atoi(common.StatusTableY)
                intY += i
                strY := strconv.Itoa(intY)
                value, err := th.ReadCell(srv, &th.Cell{Sheet:common.StatusSheetName, X:common.StatusRowLockX, Y:strY})
                if err != nil {
                        cblogger.Errorf("could not read Cell: %v", err)
                        break;
                }
                if value == "1" {
			serverID, err := th.ReadCell(srv, &th.Cell{Sheet:common.StatusSheetName, X:common.StatusSpiderIDX, Y:strY})
			if err != nil {
				cblogger.Errorf("could not read Cell: %v", err)
				break;
			}

			serverIP := (strings.Split(serverID, "-"))[0]
			childKat := kv.KeyValue{strY, serverIP}
				

                        // add this server into the effective childKat list
			childKatList = append(childKatList, childKat)
                }
        }

        return childKatList
}

// 1. check all Check Bits
// 2. make the list of live children
func getChildKatServerList2() []kv.KeyValue {
        cblogger := cblog.GetLogger("CB-SPIDER")

	srv, err := getTableHandler()
        if err != nil {
                cblogger.Fatalf("Unable to retrieve Sheets client: %v", err)
        }

        max := getMaxSpiders()
        if max == -1 {
                return nil
        }

        childKatList := []kv.KeyValue{}
	intY, _ := strconv.Atoi(common.StatusTableY)

	values, err := th.ReadRange2(srv, &th.CellRange2{Sheet:common.StatusSheetName, X:common.StatusRowLockX, Y:common.StatusTableY, X2:common.StatusCountX, Y2:common.MaxSpiders})
	if err != nil {
		cblogger.Errorf("could not read Range: %v", err)
		return nil
	}

	for count, row := range values {
		if row[0] == "1"  {
			intY += count 
			strY := strconv.Itoa(intY)
			serverIP := (strings.Split(row[1], "-"))[0]
			childKat := kv.KeyValue{strY, serverIP}

			// add this server into the effective childKat list
			childKatList = append(childKatList, childKat)
		}
        }

        return childKatList
}

func writeStatus(strY string, status *common.Status) error {
        cblogger := cblog.GetLogger("CB-SPIDER")
	srv, err := getTableHandler()
        if err != nil {
                cblogger.Fatalf("Unable to retrieve Sheets client: %v", err)
        }

	err = th.WriteRange(srv, &th.CellRange{Sheet:common.StatusSheetName, X:common.StatusSpiderIDX, Y:strY, X2:common.StatusTimeX},	[]string{status.ServerID, status.Status, status.Time})
	return err
}

func getClient(serverPort string) (common.ChildStatusClient, context.Context, error)  {
        cblogger := cblog.GetLogger("CB-SPIDER")

	// Set up a connection to the server.
        conn, err := grpc.Dial(serverPort, grpc.WithInsecure())
        if err != nil {
                cblogger.Errorf("did not connect: %v", err)
        }

        client := common.NewChildStatusClient(conn)
        ctx, _ := context.WithTimeout(context.Background(), 1000*time.Millisecond)

	return client, ctx, nil
}

var tableHandler *sheets.Service
func getTableHandler() (*sheets.Service, error) {
        cblogger := cblog.GetLogger("CB-SPIDER")

	if tableHandler != nil {
		return tableHandler, nil
	}

	var err error
	tableHandler, err = th.GetHandler()
	if err != nil {
                cblogger.Errorf("disconnected handler: %v", err)
		return nil, err
	}

	return tableHandler, nil
}
