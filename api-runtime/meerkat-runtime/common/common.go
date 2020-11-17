// MeerKat Runtime of CB-Spider.
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// by CB-Spider Team, 2020.11.

package common

import (
	"time"
	"strconv"
	cblog "github.com/cloud-barista/cb-log"

        "google.golang.org/api/sheets/v4"
        th "github.com/cloud-barista/cb-spider/api-runtime/meerkat-runtime/table-handler"
)

type StatusInfo struct{
        RowNumber string
        CheckBit string
        ServerID string
        Status string
        Time string
        Count string
}


var tableHandler *sheets.Service

func GetCurrentTime() string {
        currentTime := time.Now()
        return currentTime.Format("2006.01.02 15:04:05 Mon")
}

func GetMaxSpiders() int {
        cblogger := cblog.GetLogger("CB-SPIDER")

        result, err := strconv.Atoi(MaxSpiders)
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

func WriteStatusInfo(statusInfo *StatusInfo) error {
        cblogger := cblog.GetLogger("CB-SPIDER")
        srv, err := GetTableHandler()
        if err != nil {
                cblogger.Fatalf("Unable to retrieve Sheets client: %v", err)
        }

        err = th.WriteRange(srv, &th.CellRange{Sheet:StatusSheetName, X:StatusSpiderIDX, Y:statusInfo.RowNumber, X2:StatusCountX},
                []string{statusInfo.ServerID, statusInfo.Status, statusInfo.Time, statusInfo.Count})
        return err
}

func GetTableHandler() (*sheets.Service, error) {
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

