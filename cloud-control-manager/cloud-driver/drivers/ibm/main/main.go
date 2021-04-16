package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/softlayer/softlayer-go/services"
	"github.com/softlayer/softlayer-go/session"
)

func main() {
	// SoftLayer API username and key
	//username := "dev.secloudit@innogrid.com"
	username := os.Getenv("USER_NAME")
	apikey := os.Getenv("API_KEY")

	// Create session
	sess := session.New(username, apikey)

	// Get SoftLayer_Account service.
	service := services.GetAccountService(sess)

	// Declare mask that will be used to get specific data
	mask := "id,name,parentId,userRecordId,summary,note,status[name]," +
		"storageRepository[datacenter];imageTypeKeyName"

	// Retrieve image templates from account.
	images, err := service.Mask(mask).GetPrivateBlockDeviceTemplateGroups()
	if err != nil {
		fmt.Printf("\n Unable to retrieve image templates:\n - %s\n", err)
		return
	}

	// Following creates a JSON object which is based on data of the captured image.
	for _, image := range images {
		jsonFormat, JsonErr := json.Marshal(image)
		if JsonErr != nil {
			fmt.Println(JsonErr)
			return
		}
		fmt.Println(string(jsonFormat))
	}
}
