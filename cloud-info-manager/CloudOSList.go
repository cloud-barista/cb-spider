package cloudos

import (
	cblogger "github.com/cloud-barista/cb-log"
	"github.com/sirupsen/logrus"

	_ "fmt"
	"io/ioutil"
	"os"
	_ "sort"
	"strings"

	"gopkg.in/yaml.v3"
)

var cblog *logrus.Logger

func init() {
	cblog = cblogger.GetLogger("CLOUD-BARISTA")
}

type CloudOSList struct {
	Name []string `yaml:"cloudos"`
}

func readYaml() CloudOSList {
	// Set Environment Value of Project Root Path
	rootPath := os.Getenv("CBSPIDER_ROOT")
	if rootPath == "" {
		cblog.Error("$CBSPIDER_ROOT is not set!!")
		os.Exit(1)
	}
	data, err := ioutil.ReadFile(rootPath + "/cloud-driver-libs/cloudos.yaml")
	if err != nil {
		cblog.Error(err)
		panic(err)
	}

	var coList CloudOSList
	err = yaml.Unmarshal(data, &coList)
	if err != nil {
		cblog.Error(err)
		panic(err)
	}

	return coList
}

func ListCloudOS() []string {

	// read YAML file
	cloudosList := readYaml()

	// to Upper
	for n, cloudos := range cloudosList.Name {
		cloudosList.Name[n] = strings.ToUpper(cloudos)
	}

	//sort.Strings(cloudosList.Name)
	cblog.Info(cloudosList)

	return cloudosList.Name
}
