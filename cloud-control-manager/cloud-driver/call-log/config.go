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
//	https://godoc.org/gopkg.in/yaml.v3
//
// by CB-Spider Team, 2020.09.


package calllog

import (
    "os"
    "strings"
    "io/ioutil"
    "log"

    "gopkg.in/yaml.v3"
)

type CALLLOGCONFIG struct {
        CALLLOG struct {
                LOOPCHECK bool
                LOGLEVEL string
                LOGFILE bool
        }

        LOGFILEINFO struct {
                FILENAME string
                MAXSIZE int
                MAXBACKUPS int
                MAXAGE int
        }
}

func load(filePath string) ([]byte, error) {
        data, err := ioutil.ReadFile(filePath)
        return data, err
}

func GetConfigInfos() CALLLOGCONFIG {
        calllogRootPath := os.Getenv("CBSPIDER_ROOT")
        if calllogRootPath == "" {
                log.Fatalf("$CBSPIDER_ROOT is not set!!")
                os.Exit(1)
        }
        data, err := load(calllogRootPath + "/conf/calllog_conf.yaml")

        if err != nil {
                log.Fatalf("error: %v", err)
        }

        configInfos := CALLLOGCONFIG{}
        err = yaml.Unmarshal([]byte(data), &configInfos)
        if err != nil {
                log.Fatalf("error: %v", err)
        }

	configInfos.LOGFILEINFO.FILENAME = ReplaceEnvPath(configInfos.LOGFILEINFO.FILENAME)
	return configInfos
}

// $ABC/def ==> /abc/def
func ReplaceEnvPath(str string) string {
        if strings.Index(str, "$") == -1 {
                return str
        }

        // ex) input "$CBSTORE_ROOT/meta_db/dat"
        strList := strings.Split(str, "/")
        for n, one := range strList {
                if strings.Index(one, "$") != -1 {
                        callstoreRootPath := os.Getenv(strings.Trim(one, "$"))
                        if callstoreRootPath == "" {
                                log.Fatal(one  +" is not set!")
                        }
                        strList[n] = callstoreRootPath
                }
        }

        var resultStr string
        for _, one := range strList {
                resultStr = resultStr + one + "/"
        }
        // ex) "/root/go/src/github.com/cloud-barista/cb-spider/meta_db/dat/"
        resultStr = strings.TrimRight(resultStr, "/")
        resultStr = strings.ReplaceAll(resultStr, "//", "/")
        return resultStr
}


func GetConfigString(configInfos *CALLLOGCONFIG) string {
        d, err := yaml.Marshal(configInfos)
        if err != nil {
                log.Fatalf("error: %v", err)
        }
	return string(d)
}
