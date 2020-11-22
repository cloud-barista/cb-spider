// MeerKat Runtime of CB-Spider.
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// by CB-Spider Team, 2020.11.

package common

import (
	"fmt"
	"strings"
	"time"
//	"sync"
	"os"
)


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

func RunCommand(cmd string, serverID string) string {
        cmd = strings.TrimSpace(cmd)
        // ex) "$print abc def" => "$print" / "abc def"
        cmd_args := strings.SplitN(cmd, " ", 2)
        var strResult string = ""

	//wg := new(sync.WaitGroup)

        switch cmd_args[0] {
        // has no args
        case whoareu:
                strResult = "I'm " + serverID  + "__^..^__"

        // has 1 arg
        case print:
                if len(cmd_args) == 1 {
                        strResult = ""
                }else {
                        strResult = cmd_args[1]
                }
        case kill:
                if len(cmd_args) == 1 {
			strResult = "ERROR: " + cmd_args[0] + " need to have a argument!"
                }else {
			if cmd_args[1] == "you" {
				//wg.Add(1)
				go func() {
					time.Sleep(time.Millisecond*5000)
					os.Exit(0)
				}()
			
			}
                        strResult = "exited!"
                }

        default:
                strResult = cmd +" - is not a defined Command!"
        }

        fmt.Println(strResult)
        return strResult
}

