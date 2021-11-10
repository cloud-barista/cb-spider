package cloudos

import (
	"gopkg.in/yaml.v3"
	"io/ioutil"
	"os"
	"strings"
	"fmt"
	"github.com/fsnotify/fsnotify"
)

type CloudOSMetaInfo struct {
     Region     []string
     Credential []string
     RootDiskType []string
}

// struct for unmarshal
type YamlMetaInfo struct {
     Region     string
     Credential string
     RootDiskType string
}

// global variable to prevent file opereations
var metaInfo map[string]CloudOSMetaInfo

func GetCloudOSMetaInfo(cloudOS string) (CloudOSMetaInfo, error) {
	if metaInfo != nil {
		return metaInfo[cloudOS], nil
	}

	confFileName, err := getConfigFileName()
	if err != nil {
		cblog.Error(err)
		return CloudOSMetaInfo{}, err
	}

	readMetaYaml(confFileName)

	go setFSNotify(confFileName)

	return metaInfo[cloudOS], nil
}

func getConfigFileName() (string, error){
        // Set Environment Value of Project Root Path
        rootPath := os.Getenv("CBSPIDER_ROOT")
        if rootPath == "" {
                errmsg := "$CBSPIDER_ROOT is not set!!"
                cblog.Error(errmsg)
                return "", fmt.Errorf(errmsg)
        }
	return rootPath + "/cloud-driver-libs/cloudos_meta.yaml", nil
}

func readMetaYaml(confFileName string) error {
	if metaInfo == nil {
		metaInfo = make(map[string]CloudOSMetaInfo)
	} else { // clear map
		for k := range metaInfo {
			delete(metaInfo, k)
		}
	}

        data, err := ioutil.ReadFile(confFileName)
        if err != nil {
                cblog.Error(err)
                return err
        }

	yamlMetaInfo := make(map[string]YamlMetaInfo)
        err = yaml.Unmarshal(data, &yamlMetaInfo)
        if err != nil {
                cblog.Error(err)
                return err
        }
	convertAndSetMetaInfo(yamlMetaInfo)
	return nil
}

// map[string]YamlMetaInfo => map[string]CloudOSMetaInfo
func convertAndSetMetaInfo(yamlMetaInfo map[string]YamlMetaInfo)  {
	for k, v := range yamlMetaInfo {
		cloudOSMetaInfo := CloudOSMetaInfo {
			Region: splitAndTrim(v.Region),
			Credential: splitAndTrim(v.Credential),
			RootDiskType: splitAndTrim(v.RootDiskType),
		}
		metaInfo[k] = cloudOSMetaInfo
	}
}

// ex) in = "pd-standard / pd-balanced / pd-ssd / pd-extreme"
// ex) return = {"pd-standard", "pd-balanced", "pd-ssd", "pd-extreme"}
func splitAndTrim(in string) []string {
	ins := strings.Split(in, "/")
	for i, v := range ins {
		ins[i] = strings.TrimSpace(v)
	}
	return ins
}

// ref) https://github.com/fsnotify/fsnotify
// Thanks, fsnotify
func setFSNotify(confFileName string) {

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		cblog.Fatal(err)
	}
        defer watcher.Close()

	done := make(chan bool)
	go func() {
		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					return
				}
				if event.Op&fsnotify.Write == fsnotify.Write {
					cblog.Info("modified file:" + event.Name)
					err := readMetaYaml(confFileName)
					if err != nil {
						cblog.Fatal(err)
					}
				}
			case err, ok := <-watcher.Errors:
				if !ok {
					return
				}
				cblog.Error(err)
			}
		}
	}()

	err = watcher.Add(confFileName)
	if err != nil {
		cblog.Fatal(err)
	}

        <-done
}
