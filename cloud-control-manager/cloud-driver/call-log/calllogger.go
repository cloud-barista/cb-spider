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
	"fmt"
	"os"
	"reflect"
	"strings"
	"time"

	"github.com/chyeh/pubip"
	calllogformatter "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/call-log/formatter"
	"github.com/sirupsen/logrus"
	"github.com/snowzach/rotatefilehook"
)

type CLOUD_OS string
type RES_TYPE string

const (
	//=========== CloudOS (ref: cb-spider/cloud-driver-libs/cloudos.yaml)
	AWS       CLOUD_OS = "AWS"
	AZURE     CLOUD_OS = "AZURE"
	GCP       CLOUD_OS = "GCP"
	ALIBABA   CLOUD_OS = "ALIBABA"
	TENCENT   CLOUD_OS = "TENCENT"
	IBM       CLOUD_OS = "IBM"
	OPENSTACK CLOUD_OS = "OPENSTACK"
	CLOUDIT   CLOUD_OS = "CLOUDIT"
	NCP       CLOUD_OS = "NCP"
	NCPVPC    CLOUD_OS = "NCPVPC"
	KTCLOUD   CLOUD_OS = "KTCLOUD"
	NHNCLOUD  CLOUD_OS = "NHNCLOUD"
	DOCKER    CLOUD_OS = "DOCKER"
	MOCK      CLOUD_OS = "MOCK"
	CLOUDTWIN CLOUD_OS = "CLOUDTWIN"

	//=========== ResourceType
	VMIMAGE       RES_TYPE = "VMIMAGE"
	VMSPEC        RES_TYPE = "VMSPEC"
	VPCSUBNET     RES_TYPE = "VPC/SUBNET"
	SECURITYGROUP RES_TYPE = "SECURITYGROUP"
	VMKEYPAIR     RES_TYPE = "VMKEYPAIR"
	DISK          RES_TYPE = "DISK"
	VM            RES_TYPE = "VM"
	MYIMAGE       RES_TYPE = "MYIMAGE"
	NLB           RES_TYPE = "NETWORKLOADBALANCER"

	//=========== PMKS: Provider-Managed K8S
	CLUSTER       RES_TYPE = "CLUSTER"
)

type CALLLogger struct {
	loggerName string
	logrus     *logrus.Logger
}

// global var.
var (
	HostIPorName  string
	callLogger    *CALLLogger
	callFormatter *calllogformatter.Formatter
	calllogConfig CALLLOGCONFIG
)

func init() {
	HostIPorName = getHostIPorName()
}

func getHostIPorName() string {
	ip, err := pubip.Get()
	if err != nil {
		logrus.Error(err)
		hostName, err := os.Hostname()
		if err != nil {
			logrus.Error(err)
		}
		return hostName
	}

	return ip.String()
}

func GetLogger(loggerName string) *logrus.Logger {
	if callLogger != nil {
		return callLogger.logrus
	}
	callLogger = new(CALLLogger)
	callLogger.loggerName = loggerName
	callLogger.logrus = &logrus.Logger{
		Level:     logrus.InfoLevel,
		Out:       os.Stderr,
		Hooks:     make(logrus.LevelHooks),
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
		time.Sleep(time.Second * 2)
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
		Formatter:  getFormatter(loggerName),
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

func checkLevel(lvl string) error {
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
		LogFormat:       "[" + loggerName + "].[" + HostIPorName + "] %time% (%weekday%) %func% - %msg%\n",
	}
	return callFormatter
}

//=========================
type CLOUDLOGSCHEMA struct {
	CloudOS      CLOUD_OS // ex) AWS | AZURE | ALIBABA | GCP | OPENSTACK | CLOUDTWIN | CLOUDIT | DOCKER | NCP | MOCK | IBM
	RegionZone   string   // ex) us-east1/us-east1-c
	ResourceType RES_TYPE // ex) VMIMAGE | VMSPEC | VPCSUBNET | SECURITYGROUP | VMKEYPAIR | VM
	ResourceName string   // ex) vpc-01
	CloudOSAPI   string   // ex) CreateKeyPair()
	ElapsedTime  string   // ex) 2.0201 (sec)
	ErrorMSG     string   // if success, ""
}

/* TBD or Do not support.
type VMLOGSCHEMA struct {
}
*/

func Start() time.Time {
	return time.Now()
}

func Elapsed(start time.Time) string {
	return fmt.Sprintf("%.4f", time.Since(start).Seconds())
}

func String(logInfo interface{}) string {
	t := reflect.TypeOf(logInfo)
	v := reflect.ValueOf(logInfo)

	msg := ""
	for idx := 0; idx < t.NumField(); idx++ {
		typeOne := t.Field(idx)
		one := v.Field(idx)
		if idx < (t.NumField() - 1) {
			msg += fmt.Sprintf("\"%s\" : \"%s\", ", typeOne.Name, one)
		} else {
			msg += fmt.Sprintf("\"%s\" : \"%s\"", typeOne.Name, one)
		}
	}

	return msg
}
