// MeerKat Runtime of CB-Spider.
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// by CB-Spider Team, 2020.11.

package momkat

import (
	"fmt"
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

// Loop:
// 	1. start timer
//	2. called by others => reset timer
//	3. fired and be a MomKat
//	  (1) get childkat list
//	  (2) get the fist command
// 	  (3) if MomKat Command, run it!
//	  (4) check childkat liveness and set
//	  (5) request the Command to all ChildKat
//	  (6) clear the fist command after all completions.
func CheckChildKatAndSet(myServerID string) {

	for true { 
		common.StartTimer()
		if common.ResetFlag == true {
			continue
		}

		// role of MomKat
		SetImMomKat(myServerID)

		// (1) childKatList := getChildKatServerList()
		childKatStatusInfoList := getChildKatStatusInfoList2(myServerID)

		// (2) get the first command
		cmd := getFirstCommand()

		wg := new(sync.WaitGroup)

		// (3) if MomKat Command, run it!
		if (cmd!=nil) && (cmd.CMDTYPE==common.MOMKAT) {
			wg.Add(1)
			go func() {
				RunMomKatCommandAndSetResult(myServerID, cmd)
				wg.Done()
			}()
		}
		
		// only all childkats except this momkat
		for _, childKatStatusInfo:= range childKatStatusInfoList {
			// sould clone info object because childKatStatusInfo is a point of childKatStatusInfoList's children
			statusInfo := common.StatusInfo{childKatStatusInfo.RowNumber, childKatStatusInfo.CheckBit, 
				childKatStatusInfo.ServerID, childKatStatusInfo.Status, childKatStatusInfo.Time, childKatStatusInfo.Count}
			wg.Add(1)
			go func() {
				// (4) check childkat liveness and set
				GetAndSetStatus(statusInfo)

				// (5) request the Command to all ChildKat
				if (cmd!=nil) && (cmd.CMDTYPE==common.ALL) {
					RunCommandAndSetResult(statusInfo, cmd)
				}

				wg.Done()
			}()
		}
		wg.Wait()
		// (6)  clear the fist command and pupup command after all completions.
		if cmd != nil {
			popupCommand()
		}

	} // end of for true
}

// 1. retrive first command from Sheets
// 2. make a Command objject and return it
func getFirstCommand() *common.Command {
        cblogger := cblog.GetLogger("CB-SPIDER")

        srv, err := common.GetTableHandler()
        if err != nil {
                cblogger.Fatalf("Unable to retrieve Sheets client: %v", err)
        }

	values, err := th.ReadRange(srv, &th.CellRange{Sheet:common.CommandSheetName, X:common.CommandIDX, Y:common.CommandTableY, X2:common.CommandCMDX})
	if err != nil {
                cblogger.Errorf("could not read Range: %v", err)
                return nil
        }

	if len(values) <= 0 {
		return nil
	}

	return &common.Command{CMDID:values[0], CMDTYPE:values[1], CMD:values[2], Time:common.GetCurrentTime()}
}

func popupCommand(){
        cblogger := cblog.GetLogger("CB-SPIDER")

	srv, err := common.GetTableHandler()
        if err != nil {
                cblogger.Fatalf("Unable to retrieve Sheets client: %v", err)
        }

	values, err := th.ReadRange2(srv, &th.CellRange2{Sheet:common.CommandSheetName, X:common.CommandIDX, Y:common.CommandTableY, 
			X2:common.CommandCMDX, Y2:strSum(common.CommandTableY, common.MaxCommands)})
        if err != nil {
                cblogger.Errorf("could not read Range: %v", err)
        }

	// popup command, values type: [][]string
	switch len(values) {
	case 0:
		return 
	case 1: 
		
		th.WriteRange(srv, &th.CellRange{Sheet:common.CommandSheetName, X:common.CommandIDX, Y:common.CommandTableY,
                        X2:common.CommandCMDX}, []string{"", "", ""})
	default :
		for i, _ := range values {
			if i < (len(values)-1) {
				values[i] = values[i+1]
			}
		}
		values[len(values)-1] = []string{"", "", ""}
		th.WriteRange2(srv, &th.CellRange2{Sheet:common.CommandSheetName, X:common.CommandIDX, Y:common.CommandTableY,
                        X2:common.CommandCMDX, Y2:strSum(common.CommandTableY, common.MaxCommands)}, values)
	} // end of switch

}

func SetImMomKat(myServerID string) {
	cblogger := cblog.GetLogger("CB-SPIDER")
        srv, err := common.GetTableHandler()
        if err != nil {
                cblogger.Fatalf("Unable to retrieve Sheets client: %v", err)
        }

	strY := strDecrement(common.StatusTableY)  // minus 1
	strY = strDecrement(strY)   // minus 1
        err = th.WriteCell(srv, &th.Cell{Sheet:common.StatusSheetName, X:common.StatusSpiderIDX, Y:strY}, myServerID)
        if err != nil {
                cblogger.Fatalf("Unable to write data into sheet: %v", err)
        }
}

func GetAndSetStatus(statusInfo common.StatusInfo) {
	cblogger := cblog.GetLogger("CB-SPIDER")
	serverIP := (strings.Split(statusInfo.ServerID, "-"))[0]
	client, ctx, err := getStatusClient(serverIP)
	if err != nil {
		cblogger.Errorf("could not get Client: %v", err)
	}

	_, err = client.GetChildStatus(ctx, &common.Empty{})
	if err != nil {
		//cblogger.Errorf("could not Fetch Resource Status Information: %v", err)
		cblogger.Infof("%s: could not Fetch Resource Status Information: %v", serverIP, err)

		statusInfo.Count = strIncrement(statusInfo.Count)

		switch statusInfo.Status {
		case "L": 
			statusInfo.Status = "N"
			statusInfo.Count = "1"
		case "N": 
			if statusInfo.Count >= common.TransCount {
				statusInfo.Status = "Z"
				statusInfo.Count = "1"
			}
		case "Z": 
			if statusInfo.Count >= common.TransCount {
				statusInfo.Status = "D"
				statusInfo.Count = "1"
			}
		case "D": 
			if statusInfo.Count >= common.TransCount {
				// delete this spider in the list
				common.ClearCheckBit(statusInfo.RowNumber)
			}
		} // end of switch
	} else { // end of if
		if statusInfo.Status == "L" {
			statusInfo.Count = strIncrement(statusInfo.Count)
		} else {
			statusInfo.Status = "L"
			statusInfo.Count = "1"
		}
	}

	statusInfo.Time = common.GetCurrentTime()

	cblogger.Info("[" + statusInfo.ServerID + "] " + statusInfo.Status + "-" + statusInfo.Time)

	err = common.WriteStatusInfo(&statusInfo)
	if err != nil {
		cblogger.Errorf("could not write Cell: %v", err)
	}
}

func RunMomKatCommandAndSetResult(myServerID string, cmd *common.Command) {
	cblogger := cblog.GetLogger("CB-SPIDER")

        cmdResult, err := RunCommand(myServerID, cmd)
        if err != nil {
                //cblogger.Errorf("could not Run Command: %v", err)
                cblogger.Infof("%s: could not Run Command(%#v) - %v", myServerID, cmd, err)
        }

        // @todo Now, refined the time because time difference
        cmdResult.Time = common.GetCurrentTime()

        cblogger.Info("[" + cmdResult.ServerID + "] " + cmdResult.CMD + "-" + cmdResult.Result + "-" + cmdResult.Time)

        cmdResultInfo := common.CommandResultInfo{RowNumber:common.CommandTableY, ServerID:cmdResult.ServerID, ResultNow:cmdResult.Result, Time:cmdResult.Time}
        err = common.WriteCommandResult(&cmdResultInfo)
        if err != nil {
                cblogger.Errorf("could not write Cell: %v", err)
        }
}

func RunCommandAndSetResult(statusInfo common.StatusInfo, cmd *common.Command) {
        cblogger := cblog.GetLogger("CB-SPIDER")
        serverIP := (strings.Split(statusInfo.ServerID, "-"))[0]
        client, ctx, err := getRunCommandClient(serverIP)
        if err != nil {
                cblogger.Errorf("could not get Client: %v", err)
        }

	cmdResult, err := client.RunCommand(ctx, cmd)
        if err != nil {
                //cblogger.Errorf("could not Run Command: %v", err)
                cblogger.Infof("%s: could not Run Command(%#v) - %v", serverIP, cmd, err)
        }

	// @todo Now, refined the time because time difference
        cmdResult.Time = common.GetCurrentTime()

        cblogger.Info("[" + cmdResult.ServerID + "] " + cmdResult.CMD + "-" + cmdResult.Result + "-" + cmdResult.Time)

	cmdResultInfo := common.CommandResultInfo{RowNumber:common.CommandTableY, ServerID:cmdResult.ServerID, ResultNow:cmdResult.Result, Time:cmdResult.Time}
        err = common.WriteCommandResult(&cmdResultInfo)
        if err != nil {
                cblogger.Errorf("could not write Cell: %v", err)
        }
}

func strIncrement(strCount string) string {
	intCount, _ := strconv.Atoi(strCount)
	strCount = strconv.Itoa(intCount+1)
	return strCount
}

func strDecrement(strCount string) string {
        intCount, _ := strconv.Atoi(strCount)
        strCount = strconv.Itoa(intCount-1)
        return strCount
}

func strSum(strCount1 string, strCount2 string) string {
        intCount1, _ := strconv.Atoi(strCount1)
        intCount2, _ := strconv.Atoi(strCount2)
	strCount := strconv.Itoa(intCount1+intCount2)
        return strCount
}

func getStatusClient(serverPort string) (common.ChildStatusClient, context.Context, error)  {
        cblogger := cblog.GetLogger("CB-SPIDER")

        // Set up a connection to the server.
        conn, err := grpc.Dial(serverPort, grpc.WithInsecure())
        if err != nil {
                cblogger.Errorf("did not connect: %v", err)
        }

        client := common.NewChildStatusClient(conn)
        ctx, _ := context.WithTimeout(context.Background(), common.ChildKatCallTimeout*time.Millisecond)

        return client, ctx, nil
}

func getRunCommandClient(serverPort string) (common.RunCommandClient, context.Context, error)  {
        cblogger := cblog.GetLogger("CB-SPIDER")

        // Set up a connection to the server.
        conn, err := grpc.Dial(serverPort, grpc.WithInsecure())
        if err != nil {
                cblogger.Errorf("did not connect: %v", err)
        }

        client := common.NewRunCommandClient(conn)
        ctx, _ := context.WithTimeout(context.Background(), common.ChildKatCallTimeout*time.Millisecond)

        return client, ctx, nil
}

// 1. check all Check Bits
// 2. make the list of live children
func getChildKatStatusInfoList2(myServerID string) []common.StatusInfo {
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

	values, err := th.ReadRange2(srv, &th.CellRange2{Sheet:common.StatusSheetName, X:common.StatusRowLockX, Y:common.StatusTableY, X2:common.StatusCountX, Y2:strSum(common.StatusTableY, common.MaxSpiders)})
	if err != nil {
		cblogger.Errorf("could not read Range: %v", err)
		return nil
	}

	for count, row := range values {
		// skip self check.
		if row[1] == myServerID {
			continue
		}
		if row[0] == "1"  {
			thisY := intY + count 
			strY := strconv.Itoa(thisY)
			statusInfo := common.StatusInfo{strY, row[0], row[1], row[2], row[3], row[4]}

			// add this server into the effective childKat list
			childKatStatusInfoList = append(childKatStatusInfoList, statusInfo)
		}
        }

        return childKatStatusInfoList
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

func RunCommand(myServerID string, cmd *common.Command) (*common.CommandResult, error) {
        if cmd.CMDTYPE ==  "ALL" {
                return nil, fmt.Errorf("[%s] I'm a MomKat, I received ALL Command(%s)", myServerID, cmd.CMDID)
        }
        strResult := runCommand(cmd.CMD)
        time := common.GetCurrentTime()
        return &common.CommandResult{ServerID: myServerID, CMD: cmd.CMD, Result:strResult, Time: time}, nil
}

func runCommand(cmd string) string {
        // @todo run command
	return "MOMKAT:" + cmd + " - run command return sample msg"
}

