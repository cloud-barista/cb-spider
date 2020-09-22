package main

import (
	"fmt"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/cloud-barista/cb-spider/cloud-control-manager/call-log"

	"testing"
)

var calllogger *logrus.Logger

type DBCONN struct {
	name string
}

func init() {
	// calllog is a global variable.
	calllogger = calllog.GetLogger("HISCALL")
}

func TestCallLog(t *testing.T) {

	for i:=0;i<=0;i++ {
		calllogger.Info("start.........")

		err := createUser("newUser")
		calllogger.Info("msg for debugging msg!!")
		if err != nil {
			t.Error(err)
		}

		calllogger.Info("end.........")

		time.Sleep(time.Second*1)
		fmt.Print("\n")
	}
}

func createUser(newUser string) error {
	calllogger.Info("start creating user.")

	var db *DBCONN
	db = new(DBCONN)
	if db == nil {
		calllogger.Error("DBMS Session is closed!!")
	}
	
	isExist, err := checkUser(newUser)
	calllogger.Info("msg for info msg!!")
	if isExist {
		return err
	}

	calllogger.Info("finish creating user.")
	return nil
}

func checkUser(user string) (bool, error) {
	return false, fmt.Errorf("%s: already existed User!!", user)
}

