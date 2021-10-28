package cloudos

import (
	"gopkg.in/yaml.v3"
	"io/ioutil"
	"os"
	"fmt"
)

type CloudOSMetaInfo struct {
     Region     string
     Credential string
}

func GetCloudOSMetaInfo(cloudOS string) (CloudOSMetaInfo, error) {
	// Set Environment Value of Project Root Path
	rootPath := os.Getenv("CBSPIDER_ROOT")
        if rootPath == "" {
		errmsg := "$CBSPIDER_ROOT is not set!!"
                cblog.Error(errmsg)
		return CloudOSMetaInfo{}, fmt.Errorf(errmsg)
        }
	data, err := ioutil.ReadFile(rootPath + "/cloud-driver-libs/cloudos_meta.yaml")
	if err != nil {
		cblog.Error(err)
		return CloudOSMetaInfo{}, err
	}

	metaInfo := make(map[string]CloudOSMetaInfo)
	err = yaml.Unmarshal(data, &metaInfo)
	if err != nil {
		cblog.Error(err)
		return CloudOSMetaInfo{}, err
	}

	return metaInfo[cloudOS], nil
}
