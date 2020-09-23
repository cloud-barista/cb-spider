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

package calllogformatter

import (

	"fmt"

	"strconv"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
)

const (
	// Default log format will output [INFO]: 2006-01-02T15:04:05Z07:00 - Log message
	defaultLogFormat       = " %time% (%weekday%) %func% - %msg%\n"
	defaultTimestampFormat = time.RFC3339
)

// Formatter implements logrus.Formatter interface.
type Formatter struct {
	TimestampFormat string
	LogFormat string
}

// Format building log message.
func (f *Formatter) Format(entry *logrus.Entry) ([]byte, error) {
	output := f.LogFormat
	if output == "" {
		output = defaultLogFormat
	}

	timestampFormat := f.TimestampFormat
	if timestampFormat == "" {
		timestampFormat = defaultTimestampFormat
	}

	output = strings.Replace(output, "%time%", entry.Time.Format(timestampFormat), 1)
	output = strings.Replace(output, "%weekday%", entry.Time.Weekday().String(), 1)


        if entry.HasCaller() {
                funcVal := fmt.Sprintf("%s():%d", entry.Caller.Function, entry.Caller.Line)
		
		output = strings.Replace(output, "%func%", funcVal, 1)
	} else {
		output = strings.Replace(output, "%func%", "", 1)
	}

	output = strings.Replace(output, "%msg%", entry.Message, 1)


	for k, val := range entry.Data {
		switch v := val.(type) {
		case string:
			output = strings.Replace(output, "%"+k+"%", v, 1)
		case int:
			s := strconv.Itoa(v)
			output = strings.Replace(output, "%"+k+"%", s, 1)
		case bool:
			s := strconv.FormatBool(v)
			output = strings.Replace(output, "%"+k+"%", s, 1)
		}
	}

	return []byte(output), nil
}

func shortFilePathName(filePath string) string {
	strArray := strings.Split(filePath, "/")

	return strArray[len(strArray)-1]
}
