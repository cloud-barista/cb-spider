package cloudos

import (
        "github.com/sirupsen/logrus"
        "github.com/cloud-barista/cb-store/config"

	"gopkg.in/yaml.v3"
	"io/ioutil"
	"os"
	_ "fmt"
	"strings"
	"sort"
)

var cblog *logrus.Logger

func init() {
        cblog = config.Cblogger
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
	for n, cloudos := range cloudosList.Name{
		cloudosList.Name[n] = strings.ToUpper(cloudos)
	}

	sort.Strings(cloudosList.Name)
	cblog.Info(cloudosList)

	return cloudosList.Name
}

