// Cloud Driver Interface of CB-Spider.
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// This is Mini Driver.
//
// by CB-Spider Team, 2021.11.

package resources

import (
	"time"
	"errors"
	"encoding/json"

	cblog "github.com/cloud-barista/cb-log"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
	"github.com/cloud-barista/cb-spider/interface/api"


	"github.com/go-redis/redis"
	"fmt"
)


type MiniImageHandler struct {
	MiniAddr string
	AuthToken string
	ConnectionName string
}

func init() {
}


func (imageHandler *MiniImageHandler) ListImage() ([]*irs.ImageInfo, error) {
	cblogger := cblog.GetLogger("CB-SPIDER")
	cblogger.Info("Mini Driver: called ListImage()!")

	miniAddr := imageHandler.MiniAddr

	// 1. Create CloudResourceHandler
        crh := api.NewCloudResourceHandler()

        // 2. Setup env.
        err := crh.SetServerAddr(miniAddr)
        if err != nil {
                cblogger.Error(err)
        }
        err = crh.SetTimeout(1800 * time.Second)
        if err != nil {
                cblogger.Error(err)
        }

	// 3. Open New Session
        err = crh.Open()
        if err != nil {
                cblogger.Error(err)
		return nil, err
        }
        // 4. Close (with defer)
        defer crh.Close()

        // 5. get ImageList
	//connName := "aws-ohio-config"
	//result, err := crh.ListImageInfo(connName)
	result, err := crh.ListImageByParam(imageHandler.ConnectionName)
        if err != nil {
                cblogger.Error(err)
		return nil, err
        }
	var jsonResult struct {
		Result []*irs.ImageInfo `json:"image"`
	}
	json.Unmarshal([]byte(result), &jsonResult)

	return jsonResult.Result, nil
}

func (imageHandler *MiniImageHandler) ListImage2() ([]*irs.ImageInfo, error) {
        cblogger := cblog.GetLogger("CB-SPIDER")
        cblogger.Info("Mini Driver: called ListImage()!")

	client := redis.NewClient(&redis.Options{
                Addr: "localhost:6379",
                Password: "",
                DB: 0,
	})

	result, err := client.Get("imageinfo:aws:ohio").Result()
	if err != nil {
                cblogger.Error(err)
		return nil, err
	}

	var jsonResult struct {
                Result []*irs.ImageInfo `json:"image"`
        }
        json.Unmarshal([]byte(result), &jsonResult)

	return jsonResult.Result, nil
}

func (imageHandler *MiniImageHandler) GetImage(imageIID irs.IID) (irs.ImageInfo, error) {
	cblogger := cblog.GetLogger("CB-SPIDER")
	cblogger.Info("Mini Driver: called GetImage()!")

	imgInfoList, err := imageHandler.ListImage()
	if err != nil {
		cblogger.Error(err)
		return irs.ImageInfo{}, err
	}

	for _, info := range imgInfoList {
		if (*info).IId.NameId == imageIID.NameId {
			return *info, nil
		}
	}

	return irs.ImageInfo{}, errors.New(imageIID.NameId + " image does not exist!!")
}


//---

func (imageHandler *MiniImageHandler) CreateImage(imageReqInfo irs.ImageReqInfo) (irs.ImageInfo, error) {
	cblogger := cblog.GetLogger("CB-SPIDER")
	cblogger.Info("Mini Driver: called CreateImage()!")

	return irs.ImageInfo{}, errors.New("not implemented")
}

func (imageHandler *MiniImageHandler) DeleteImage(imageIID irs.IID) (bool, error) {
	cblogger := cblog.GetLogger("CB-SPIDER")
	cblogger.Info("Mini Driver: called DeleteImage()!")

	return false, errors.New("not implemented")
}

func (imageHandler *MiniImageHandler) CheckWindowsImage(imageIID irs.IID) (bool, error) {
	return false, fmt.Errorf("Does not support CheckWindowsImage() yet!!")
}

