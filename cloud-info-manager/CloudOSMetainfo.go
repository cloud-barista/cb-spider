package cloudos

import (
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"sync"

	"github.com/fsnotify/fsnotify"
	"gopkg.in/yaml.v3"
)

// CloudOSMetaInfo represents metadata information for a specific Cloud OS.
// @Description Metadata information related to a Cloud OS, including supported regions, credentials, and disk configurations.
type CloudOSMetaInfo struct {
	Region               []string `json:"Region" validate:"required"`               // A list of supported regions by the Cloud OS.
	Credential           []string `json:"Credential" validate:"required"`           // A list of required credential keys.
	CredentialCSP        []string `json:"CredentialCSP" validate:"required"`        // A list of credentials required by the cloud service provider (CSP).
	RootDiskType         []string `json:"RootDiskType" validate:"required"`         // Supported root disk types (e.g., gp2, standard).
	RootDiskSize         []string `json:"RootDiskSize" validate:"required"`         // Supported root disk sizes (in GB).
	DiskType             []string `json:"DiskType" validate:"required"`             // Supported additional disk types.
	DiskSize             []string `json:"DiskSize" validate:"required"`             // Supported additional disk sizes (in GB).
	IdMaxLength          []string `json:"IdMaxLength" validate:"required"`          // Maximum allowed length for IDs in the cloud provider.
	DefaultRegionToQuery []string `json:"DefaultRegionToQuery" validate:"required"` // Default region to use if none is specified for a query.
}

// struct for unmarshal
type YamlMetaInfo struct {
	Region               string
	Credential           string
	CredentialCSP        string
	RootDiskType         string
	RootDiskSize         string
	DiskType             string
	DiskSize             string
	IdMaxLength          string
	DefaultRegionToQuery string
}

// global variable to prevent file opereations
var metaInfo map[string]CloudOSMetaInfo

// Lock for metaInfo
var rwMutex sync.RWMutex

func GetCloudOSMetaInfo(cloudOS string) (CloudOSMetaInfo, error) {

	cloudOS = strings.ToUpper(cloudOS)

	rwMutex.Lock()
	if metaInfo != nil {
		rwMutex.Unlock()
		return metaInfo[cloudOS], nil
	}
	rwMutex.Unlock()

	confFileName, err := getConfigFileName()
	if err != nil {
		cblog.Error(err)
		return CloudOSMetaInfo{}, err
	}

	readMetaYaml(confFileName)

	go setFSNotify(confFileName)

	rwMutex.Lock()
	mInfo := metaInfo[cloudOS]
	ret := CloudOSMetaInfo{
		Region:               cloneSlice(mInfo.Region),
		Credential:           cloneSlice(mInfo.Credential),
		CredentialCSP:        cloneSlice(mInfo.CredentialCSP),
		RootDiskType:         cloneSlice(mInfo.RootDiskType),
		RootDiskSize:         cloneSlice(mInfo.RootDiskSize),
		DiskType:             cloneSlice(mInfo.DiskType),
		DiskSize:             cloneSlice(mInfo.DiskSize),
		IdMaxLength:          cloneSlice(mInfo.IdMaxLength),
		DefaultRegionToQuery: cloneSlice(mInfo.DefaultRegionToQuery),
	}
	rwMutex.Unlock()
	return ret, nil
}

func getConfigFileName() (string, error) {
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
	rwMutex.Lock()

	if metaInfo == nil {
		metaInfo = make(map[string]CloudOSMetaInfo)
	} else { // clear map
		for k := range metaInfo {
			delete(metaInfo, k)
		}
	}
	rwMutex.Unlock()

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

	rwMutex.Lock()
	convertAndSetMetaInfo(yamlMetaInfo)
	rwMutex.Unlock()
	return nil
}

// map[string]YamlMetaInfo => map[string]CloudOSMetaInfo
func convertAndSetMetaInfo(yamlMetaInfo map[string]YamlMetaInfo) {

	for k, v := range yamlMetaInfo {
		cloudOSMetaInfo := CloudOSMetaInfo{
			Region:               splitAndTrim(v.Region),
			Credential:           splitAndTrim(v.Credential),
			CredentialCSP:        splitAndTrim(v.CredentialCSP),
			RootDiskType:         splitAndTrim(v.RootDiskType),
			RootDiskSize:         splitAndTrim(v.RootDiskSize),
			DiskType:             splitAndTrim(v.DiskType),
			DiskSize:             splitAndTrim(v.DiskSize),
			IdMaxLength:          splitAndTrim(v.IdMaxLength),
			DefaultRegionToQuery: splitAndTrim(v.DefaultRegionToQuery),
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
