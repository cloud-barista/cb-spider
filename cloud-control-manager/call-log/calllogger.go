// Call-Log: calling logger of Cloud & VM in CB-Spider
//           Referred to cb-log
//
//      * Cloud-Barista: https://github.com/cloud-barista
//      * CB-Spider: https://github.com/cloud-barista/cb-spider
//      * cb-log: https://github.com/cloud-barista/cb-log
//
// load and set config file
//
// ref) https://github.com/go-yaml/yaml/tree/v3
//      https://godoc.org/gopkg.in/yaml.v3
//
// by CB-Spider Team, 2020.09.

package calllog


import (
	"os"
	"fmt"
	"time"
	"strings"
	"reflect"

        "github.com/sirupsen/logrus"
	"github.com/snowzach/rotatefilehook"
	"github.com/cloud-barista/cb-spider/cloud-control-manager/call-log/formatter"
)

type CALLLogger struct {
	loggerName string
	logrus *logrus.Logger
}

// global var.
var (
	callLogger *CALLLogger
	callFormatter *calllogformatter.Formatter
	calllogConfig CALLLOGCONFIG
)

func GetLog() *CALLLogger {
	return callLogger
}

func GetLogger(loggerName string) *logrus.Logger {
	if callLogger != nil {
		return callLogger.logrus
	}
	callLogger = new(CALLLogger)
	callLogger.loggerName = loggerName
	callLogger.logrus =  &logrus.Logger{
        Level: logrus.InfoLevel,
        Out:   os.Stderr,
        Hooks: make(logrus.LevelHooks),
        Formatter: getFormatter(loggerName),
	}

	// set config.
	setup(loggerName)
	return callLogger.logrus
}

func setup(loggerName string) {
	calllogConfig = GetConfigInfos()
	callLogger.logrus.SetReportCaller(true)

	if calllogConfig.CALLLOG.LOOPCHECK {
		SetLevel(calllogConfig.CALLLOG.LOGLEVEL)
		go levelSetupLoop(loggerName)
	} else {
		SetLevel(calllogConfig.CALLLOG.LOGLEVEL)
	}

	if calllogConfig.CALLLOG.LOGFILE {
		setRotateFileHook(loggerName, &calllogConfig)
	}
}

// Now, this method is busy wait. 
// @TODO must change this  with file watch&event.
// ref) https://github.com/fsnotify/fsnotify/blob/master/example_test.go
func levelSetupLoop(loggerName string) {
	for {
		calllogConfig = GetConfigInfos()
		SetLevel(calllogConfig.CALLLOG.LOGLEVEL)
		time.Sleep(time.Second*2)
	}
}

func setRotateFileHook(loggerName string, logConfig *CALLLOGCONFIG) {
	level, _ := logrus.ParseLevel(logConfig.CALLLOG.LOGLEVEL)

        rotateFileHook, err := rotatefilehook.NewRotateFileHook(rotatefilehook.RotateFileConfig{
                Filename:   logConfig.LOGFILEINFO.FILENAME,
                MaxSize:    logConfig.LOGFILEINFO.MAXSIZE, // megabytes
                MaxBackups: logConfig.LOGFILEINFO.MAXBACKUPS,
                MaxAge:     logConfig.LOGFILEINFO.MAXAGE, //days
                Level:      level,
                Formatter: getFormatter(loggerName),
        })

        if err != nil {
                logrus.Fatalf("Failed to initialize file rotate hook: %v", err)
        }
        callLogger.logrus.AddHook(rotateFileHook)
}

func SetLevel(strLevel string) {
	err := checkLevel(strLevel)
	if err != nil {
                logrus.Errorf("Failed to set log level: %v", err)
	}
	level, _ := logrus.ParseLevel(strLevel)
	callLogger.logrus.SetLevel(level)
}

func checkLevel(lvl string) (error) {
	switch strings.ToLower(lvl) {
	case "error":
		return nil
	case "info":
		return nil
	}
	return fmt.Errorf("not a valid calllog Level: %q", lvl)
}

func GetLevel() string {
	return callLogger.logrus.GetLevel().String()
}

func getFormatter(loggerName string) *calllogformatter.Formatter {

	if callFormatter != nil {
		return callFormatter
	}
	callFormatter = &calllogformatter.Formatter{
            TimestampFormat: "2006-01-02 15:04:05",
            LogFormat:       "[" + loggerName + "]." + "[%lvl%]: %time% %func% - %msg%\n",
        }	
	return callFormatter
}

type CLOUDLOGSCHEMA struct {
	CSPName string      // ex) AWS | AZURE | ALIBABA | GCP | OPENSTACK | CLOUDTWIN | CLOUDIT | DOCKER
	RegionZone string   // ex) us-east1/us-east1-c
	ResourceType string // ex) VPC/SUBNET | SECURITYGROUP | KEYPAIR | VM

	ResourceName string // ex) vpc-01
	ElapsedTime string  // ex) 10msec

	ErrorNumber string  // if success, ""
	ErrorMSG string     // if success, ""
}
/*
type VMLOGSCHEMA struct {
}
*/

func Info(logInfo interface{}) {
	t := reflect.TypeOf(logInfo)
	v := reflect.ValueOf(logInfo)

	msg := ""
	for idx:=0; idx < t.NumField(); idx++ {
                typeOne := t.Field(idx)
                one := v.Field(idx)
		if idx != t.NumField() {
			msg += fmt.Sprintf("\"%s\" : \"%s\", ", typeOne.Name, one)
		} else {
			msg += fmt.Sprintf("\"%s\" : \"%s\"", typeOne.Name, one)
		}
        }
	
	callLogger.logrus.Info(msg)	
}

func Error(msg string) {
}
