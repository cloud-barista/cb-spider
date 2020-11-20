// MeerKat Runtime of CB-Spider.
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// by CB-Spider Team, 2020.11.

package childkat

import (
	"context"
	"fmt"
	"strings"
        common "github.com/cloud-barista/cb-spider/api-runtime/meerkat-runtime/common"
)

var MyServerID string
type Server struct{}


//////////////////////////////////////////////// StatusZone
func (s *Server) GetChildStatus(ctx context.Context, in *common.Empty) (*common.Status, error) {
        common.ResetTimer()
        return GetStatus()
}

func GetStatus() (*common.Status, error) {
        status := "L"
        time := common.GetCurrentTime()
        return &common.Status{ServerID: MyServerID, Status: status, Time: time}, nil
}



//////////////////////////////////////////////// CommandZone
func (s *Server) RunCommand(ctx context.Context, cmd *common.Command) (*common.CommandResult, error) {
	if cmd.CMDTYPE ==  "MOMKAT" {
		return nil, fmt.Errorf("[%s] I'm a ChildKat, I received MOMKAT Command(%s)", MyServerID, cmd.CMDID)
	}
	strResult := runCommand(cmd.CMD)
        time := common.GetCurrentTime()
	return &common.CommandResult{ServerID: MyServerID, CMD: cmd.CMD, Result:strResult, Time: time}, nil
}

// Definitions of Command Type
const (
	// has no args
        whoareu string = "$whoareu"

	// has 1 arg
        print string = "$print" // $print Hello
        kill string = "$kill"   // $kill you, $kill ChildKatID
        clear string = "$clear" // $clear cmdresult
        list string = "$list"   // $list vm
)

func runCommand(cmd string) string {
	cmd = strings.TrimSpace(cmd)
	// ex) "$print abc def" => "$print" / "abc def"
	cmd_args := strings.SplitN(cmd, " ", 2)
	var strResult string = ""

	switch cmd_args[0] {
	// has no args
	case whoareu: 
		strResult = "I'm " + MyServerID  + "__^..^__"

	// has 1 arg
	case print: 
		if len(cmd_args) == 1 {
			strResult = ""
		}else {
			strResult = cmd_args[1]
		}

	default: 
		strResult = cmd +" - is not a defined Command!"
	}

	fmt.Println(strResult)
	return strResult
}
