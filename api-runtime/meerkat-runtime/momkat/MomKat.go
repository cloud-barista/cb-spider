// MeerKat Runtime of CB-Spider.
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// by CB-Spider Team, 2020.11.

package momkat

import (
	"strconv"
	"strings"
	"sync"
	"time"

	"context"
        "google.golang.org/grpc"

	cblog "github.com/cloud-barista/cb-log"
        common "github.com/cloud-barista/cb-spider/api-runtime/meerkat-runtime/common"
	th "github.com/cloud-barista/cb-spider/api-runtime/meerkat-runtime/table-handler"
	kv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
)

func CheckChildKatAndSet() {
	//childKatList := getChildKatServerList()
	childKatStatusInfoList := getChildKatStatusInfoList2()

	wg := new(sync.WaitGroup)

	for _, childKatStatusInfo:= range childKatStatusInfoList {
		wg.Add(1)
		go func() {
			GetAndSetStatus(childKatStatusInfo)
			wg.Done()
		}()
	}
	wg.Wait()
}

func GetAndSetStatus(statusInfo common.StatusInfo) {
	cblogger := cblog.GetLogger("CB-SPIDER")
	serverIP := (strings.Split(statusInfo.ServerID, "-"))[0]
	client, ctx, err := getClient(serverIP)
	if err != nil {
		cblogger.Errorf("could not get Client: %v", err)
	}

	_, err = client.GetChildStatus(ctx, &common.Empty{})
	if err != nil {
		//cblogger.Errorf("could not Fetch Resource Status Information: %v", err)
		cblogger.Infof("%s: could not Fetch Resource Status Information: %v", serverIP, err)

		statusInfo.Status = "N"
	}
	statusInfo.Time = common.GetCurrentTime()

	cblogger.Info("[" + statusInfo.ServerID + "] " + statusInfo.Status + "-" + statusInfo.Time)
	err = common.WriteStatusInfo(&statusInfo)
	if err != nil {
		cblogger.Errorf("could not write Cell: %v", err)
	}
}

func getClient(serverPort string) (common.ChildStatusClient, context.Context, error)  {
        cblogger := cblog.GetLogger("CB-SPIDER")

        // Set up a connection to the server.
        conn, err := grpc.Dial(serverPort, grpc.WithInsecure())
        if err != nil {
                cblogger.Errorf("did not connect: %v", err)
        }

        client := common.NewChildStatusClient(conn)
        ctx, _ := context.WithTimeout(context.Background(), 50*time.Millisecond)

        return client, ctx, nil
}

// 1. check all Check Bits
// 2. make the list of live children
// deprecated because Goolge Sheeets access Quota limits
func getChildKatServerList() []kv.KeyValue {
	cblogger := cblog.GetLogger("CB-SPIDER")

	srv, err := common.GetTableHandler()
        if err != nil {
                cblogger.Fatalf("Unable to retrieve Sheets client: %v", err)
        }

        max := common.GetMaxSpiders()
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
func getChildKatStatusInfoList2() []common.StatusInfo {
        cblogger := cblog.GetLogger("CB-SPIDER")

	srv, err := common.GetTableHandler()
        if err != nil {
                cblogger.Fatalf("Unable to retrieve Sheets client: %v", err)
        }

        max := common.GetMaxSpiders()
        if max == -1 {
                return nil
        }

        childKatStatusInfoList := []common.StatusInfo{}
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
			statusInfo := common.StatusInfo{strY, row[0], row[1], row[2], row[3], "1"}

			// add this server into the effective childKat list
			childKatStatusInfoList = append(childKatStatusInfoList, statusInfo)
		}
        }

        return childKatStatusInfoList
}

