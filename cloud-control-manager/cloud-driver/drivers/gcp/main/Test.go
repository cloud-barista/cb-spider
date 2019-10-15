// Proof of Concepts of CB-Spider.
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// This is a Cloud Driver Example for PoC Test.
//
// by hyokyung.kim@innogrid.co.kr, 2019.07.

package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"

	idrv "../../../interfaces"
	irs "../../../interfaces/resources"
	gcpdrv "../../gcp"
	yaml "gopkg.in/yaml.v3"
)

// Test VM Handler Functions (Get VM Info, VM Status)
// func getVMInfo() {
// 	vmHandler, err := setVMHandler()
// 	if err != nil {
// 		panic(err)
// 	}
// 	config := readConfigFile()

// 	// Get VM List
// 	vmList := vmHandler.ListVM()
// 	for i, vm := range vmList {
// 		fmt.Println("[", i, "] ")
// 		spew.Dump(vm)
// 	}

// 	vmId := config.GCP.GroupName + ":" + config.GCP.VMName

// 	// Get VM Info
// 	vmInfo := vmHandler.GetVM(vmId)
// 	spew.Dump(vmInfo)

// 	// Get VM Status List
// 	vmStatusList := vmHandler.ListVMStatus()
// 	for i, vmStatus := range vmStatusList {
// 		fmt.Println("[", i, "] ", *vmStatus)
// 	}

// 	// Get VM Status
// 	vmStatus := vmHandler.GetVMStatus(vmId)
// 	fmt.Println(vmStatus)
// }

// // Test VM Lifecycle Management (Suspend/Resume/Reboot/Terminate)
// func handleVM() {
// 	vmHandler, err := setVMHandler()
// 	if err != nil {
// 		panic(err)
// 	}
// 	config := readConfigFile()

// 	fmt.Println("VM LifeCycle Management")
// 	fmt.Println("1. Suspend VM")
// 	fmt.Println("2. Resume VM")
// 	fmt.Println("3. Reboot VM")
// 	fmt.Println("4. Terminate VM")

// 	for {
// 		var commandNum int
// 		inputCnt, err := fmt.Scan(&commandNum)
// 		if err != nil {
// 			panic(err)
// 		}

// 		vmId := config.GCP.GroupName + ":" + config.GCP.VMName

// 		if inputCnt == 1 {
// 			switch commandNum {
// 			case 1:
// 				fmt.Println("Start Suspend VM ...")
// 				vmHandler.SuspendVM(vmId)
// 				fmt.Println("Finish Suspend VM")
// 			case 2:
// 				fmt.Println("Start Resume  VM ...")
// 				vmHandler.ResumeVM(vmId)
// 				fmt.Println("Finish Resume VM")
// 			case 3:
// 				fmt.Println("Start Reboot  VM ...")
// 				vmHandler.RebootVM(vmId)
// 				fmt.Println("Finish Reboot VM")
// 			case 4:
// 				fmt.Println("Start Terminate  VM ...")
// 				vmHandler.TerminateVM(vmId)
// 				fmt.Println("Finish Terminate VM")
// 			}
// 		}
// 	}
// }

// // Test VM Deployment
// func createVM() {
// 	fmt.Println("Start Create VM ...")
// 	vmHandler, err := setVMHandler()
// 	if err != nil {
// 		panic(err)
// 	}
// 	config := readConfigFile()

// 	vmName := config.GCP.GroupName + ":" + config.GCP.VMName
// 	imageId := config.GCP.Image.Publisher + ":" + config.GCP.Image.Offer + ":" + config.GCP.Image.Sku + ":" + config.GCP.Image.Version
// 	vmReqInfo := irs.VMReqInfo{
// 		Name: vmName,
// 		ImageInfo: irs.ImageInfo{
// 			Id: imageId,
// 		},
// 		SpecID: config.GCP.VMSize,
// 		VNetworkInfo: irs.VNetworkInfo{
// 			Id: config.GCP.Network.ID,
// 		},
// 		LoginInfo: irs.LoginInfo{
// 			AdminUsername: config.GCP.Os.AdminUsername,
// 			AdminPassword: config.GCP.Os.AdminPassword,
// 		},
// 	}

// 	vm, err := vmHandler.StartVM(vmReqInfo)
// 	if err != nil {
// 		panic(err)
// 	}
// 	spew.Dump(vm)
// 	fmt.Println("Finish Create VM")
// }

// func testImageHandler() {
// 	imageHandler, err := setImageHandler()
// 	if err != nil {
// 		panic(err)
// 	}
// 	config := readConfigFile()

// 	fmt.Println("Test ImageHandler")
// 	fmt.Println("1. ListImage()")
// 	fmt.Println("2. GetImage()")
// 	fmt.Println("3. CreateImage()")
// 	fmt.Println("4. DeleteImage()")
// 	fmt.Println("5. Exit Program")

// Loop:
// 	for {
// 		var commandNum int
// 		inputCnt, err := fmt.Scan(&commandNum)
// 		if err != nil {
// 			panic(err)
// 		}

// 		imageId := config.GCP.ImageInfo.GroupName + ":" + config.GCP.ImageInfo.Name

// 		if inputCnt == 1 {
// 			switch commandNum {
// 			case 1:
// 				fmt.Println("Start ListImage() ...")
// 				imageHandler.ListImage()
// 				fmt.Println("Finish ListImage()")
// 			case 2:
// 				fmt.Println("Start GetImage() ...")
// 				imageHandler.GetImage(imageId)
// 				fmt.Println("Finish GetImage()")
// 			case 3:
// 				fmt.Println("Start CreateImage() ...")
// 				reqInfo := irs.ImageReqInfo{Id: imageId}
// 				_, err := imageHandler.CreateImage(reqInfo)
// 				if err != nil {
// 					panic(err)
// 				}
// 				fmt.Println("Finish CreateImage()")
// 			case 4:
// 				fmt.Println("Start DeleteImage() ...")
// 				imageHandler.DeleteImage(imageId)
// 				fmt.Println("Finish DeleteImage()")
// 			case 5:
// 				fmt.Println("Exit Program")
// 				break Loop
// 			}
// 		}
// 	}
// }

// func testPublicIPHandler() {
// 	publicIPHandler, err := setPublicIPHandler()
// 	if err != nil {
// 		panic(err)
// 	}
// 	config := readConfigFile()

// 	fmt.Println("Test PublicIPHandler")
// 	fmt.Println("1. ListPublicIP()")
// 	fmt.Println("2. GetPublicIP()")
// 	fmt.Println("3. CreatePublicIP()")
// 	fmt.Println("4. DeletePublicIP()")
// 	fmt.Println("5. Exit Program")

// Loop:
// 	for {
// 		var commandNum int
// 		inputCnt, err := fmt.Scan(&commandNum)
// 		if err != nil {
// 			panic(err)
// 		}

// 		publicIPId := config.GCP.PublicIP.GroupName + ":" + config.GCP.PublicIP.Name

// 		if inputCnt == 1 {
// 			switch commandNum {
// 			case 1:
// 				fmt.Println("Start ListPublicIP() ...")
// 				publicIPHandler.ListPublicIP()
// 				fmt.Println("Finish ListPublicIP()")
// 			case 2:
// 				fmt.Println("Start GetPublicIP() ...")
// 				publicIPHandler.GetPublicIP(publicIPId)
// 				fmt.Println("Finish GetPublicIP()")
// 			case 3:
// 				fmt.Println("Start CreatePublicIP() ...")
// 				reqInfo := irs.PublicIPReqInfo{Id: publicIPId}
// 				_, err := publicIPHandler.CreatePublicIP(reqInfo)
// 				if err != nil {
// 					panic(err)
// 				}
// 				fmt.Println("Finish CreatePublicIP()")
// 			case 4:
// 				fmt.Println("Start DeletePublicIP() ...")
// 				publicIPHandler.DeletePublicIP(publicIPId)
// 				fmt.Println("Finish DeletePublicIP()")
// 			case 5:
// 				fmt.Println("Exit Program")
// 				break Loop
// 			}
// 		}
// 	}
// }

// func testSecurityHandler() {
// 	securityHandler, err := setSecurityHandler()
// 	if err != nil {
// 		panic(err)
// 	}
// 	config := readConfigFile()

// 	fmt.Println("Test SecurityHandler")
// 	fmt.Println("1. ListSecurity()")
// 	fmt.Println("2. GetSecurity()")
// 	fmt.Println("3. CreateSecurity()")
// 	fmt.Println("4. DeleteSecurity()")
// 	fmt.Println("5. Exit Program")

// Loop:
// 	for {
// 		var commandNum int
// 		inputCnt, err := fmt.Scan(&commandNum)
// 		if err != nil {
// 			panic(err)
// 		}

// 		securityId := config.GCP.Security.GroupName + ":" + config.GCP.Security.Name

// 		if inputCnt == 1 {
// 			switch commandNum {
// 			case 1:
// 				fmt.Println("Start ListSecurity() ...")
// 				securityHandler.ListSecurity()
// 				fmt.Println("Finish ListSecurity()")
// 			case 2:
// 				fmt.Println("Start GetSecurity() ...")
// 				securityHandler.GetSecurity(securityId)
// 				fmt.Println("Finish GetSecurity()")
// 			case 3:
// 				fmt.Println("Start CreateSecurity() ...")
// 				reqInfo := irs.SecurityReqInfo{Id: securityId}
// 				_, err := securityHandler.CreateSecurity(reqInfo)
// 				if err != nil {
// 					panic(err)
// 				}
// 				fmt.Println("Finish CreateSecurity()")
// 			case 4:
// 				fmt.Println("Start DeleteSecurity() ...")
// 				securityHandler.DeleteSecurity(securityId)
// 				fmt.Println("Finish DeleteSecurity()")
// 			case 5:
// 				fmt.Println("Exit Program")
// 				break Loop
// 			}
// 		}
// 	}
// }

// func testVNetworkHandler() {
// 	vNetHandler, err := setVNetHandler()
// 	if err != nil {
// 		panic(err)
// 	}
// 	config := readConfigFile()

// 	fmt.Println("Test VNetworkHandler")
// 	fmt.Println("1. ListVNetwork()")
// 	fmt.Println("2. GetVNetwork()")
// 	fmt.Println("3. CreateVNetwork()")
// 	fmt.Println("4. DeleteVNetwork()")
// 	fmt.Println("5. Exit Program")

// Loop:
// 	for {
// 		var commandNum int
// 		inputCnt, err := fmt.Scan(&commandNum)
// 		if err != nil {
// 			panic(err)
// 		}

// 		networkId := config.GCP.VNetwork.GroupName + ":" + config.GCP.VNetwork.Name

// 		if inputCnt == 1 {
// 			switch commandNum {
// 			case 1:
// 				fmt.Println("Start ListVNetwork() ...")
// 				vNetHandler.ListVNetwork()
// 				fmt.Println("Finish ListVNetwork()")
// 			case 2:
// 				fmt.Println("Start GetVNetwork() ...")
// 				vNetHandler.GetVNetwork(networkId)
// 				fmt.Println("Finish GetVNetwork()")
// 			case 3:
// 				fmt.Println("Start CreateVNetwork() ...")
// 				reqInfo := irs.VNetworkReqInfo{Id: networkId}
// 				_, err := vNetHandler.CreateVNetwork(reqInfo)
// 				if err != nil {
// 					panic(err)
// 				}
// 				fmt.Println("Finish CreateVNetwork()")
// 			case 4:
// 				fmt.Println("Start DeleteVNetwork() ...")
// 				vNetHandler.DeleteVNetwork(networkId)
// 				fmt.Println("Finish DeleteVNetwork()")
// 			case 5:
// 				fmt.Println("Exit Program")
// 				break Loop
// 			}
// 		}
// 	}
// }

// func testVNicHandler() {
// 	vNicHandler, err := setVNicHandler()
// 	if err != nil {
// 		panic(err)
// 	}
// 	config := readConfigFile()

// 	fmt.Println("Test VNicHandler")
// 	fmt.Println("1. ListVNic()")
// 	fmt.Println("2. GetVNic()")
// 	fmt.Println("3. CreateVNic()")
// 	fmt.Println("4. DeleteVNic()")
// 	fmt.Println("5. Exit Program")

// Loop:
// 	for {
// 		var commandNum int
// 		inputCnt, err := fmt.Scan(&commandNum)
// 		if err != nil {
// 			panic(err)
// 		}

// 		vNicId := config.GCP.VNic.GroupName + ":" + config.GCP.VNic.Name

// 		if inputCnt == 1 {
// 			switch commandNum {
// 			case 1:
// 				fmt.Println("Start ListVNic() ...")
// 				vNicHandler.ListVNic()
// 				fmt.Println("Finish ListVNic()")
// 			case 2:
// 				fmt.Println("Start GetVNic() ...")
// 				vNicHandler.GetVNic(vNicId)
// 				fmt.Println("Finish GetVNic()")
// 			case 3:
// 				fmt.Println("Start CreateVNic() ...")
// 				reqInfo := irs.VNicReqInfo{Id: vNicId}
// 				_, err := vNicHandler.CreateVNic(reqInfo)
// 				if err != nil {
// 					panic(err)
// 				}
// 				fmt.Println("Finish CreateVNic()")
// 			case 4:
// 				fmt.Println("Start DeleteVNic() ...")
// 				vNicHandler.DeleteVNic(vNicId)
// 				fmt.Println("Finish DeleteVNic()")
// 			case 5:
// 				fmt.Println("Exit Program")
// 				break Loop
// 			}
// 		}
// 	}
// }

func setVMHandler() (irs.VMHandler, error) {
	var cloudDriver idrv.CloudDriver
	cloudDriver = new(gcpdrv.GCPDriver)
	cloudDriver.GetDriverVersion()
	credentialFilePath := os.Getenv("GOOGLE_APPLICATION_CREDENTIALS")
	config, _ := readFileConfig(credentialFilePath)
	region := "asia-northeast1"

	connectionInfo := idrv.ConnectionInfo{
		CredentialInfo: idrv.CredentialInfo{
			ClientSecret: credentialFilePath,
			ProjectID:    config.ProjectID,
		},
		RegionInfo: idrv.RegionInfo{
			Region: region,
		},
	}

	cloudConnection, err := cloudDriver.ConnectCloud(connectionInfo)
	if err != nil {
		return nil, err
	}
	vmHandler, err := cloudConnection.CreateVMHandler()
	if err != nil {
		return nil, err
	}
	return vmHandler, nil
}

// func setImageHandler() (irs.ImageHandler, error) {
// 	var cloudDriver idrv.CloudDriver
// 	cloudDriver = new(gcpdrv.GCPDriver)

// 	config := readConfigFile()
// 	connectionInfo := idrv.ConnectionInfo{
// 		CredentialInfo: idrv.CredentialInfo{
// 			ClientId:       config.GCP.ClientId,
// 			ClientSecret:   config.GCP.ClientSecret,
// 			TenantId:       config.GCP.TenantId,
// 			SubscriptionId: config.GCP.SubscriptionID,
// 		},
// 		RegionInfo: idrv.RegionInfo{
// 			Region:        config.GCP.Location,
// 			ResourceGroup: config.GCP.GroupName,
// 		},
// 	}

// 	cloudConnection, err := cloudDriver.ConnectCloud(connectionInfo)
// 	if err != nil {
// 		return nil, err
// 	}
// 	imageHandler, err := cloudConnection.CreateImageHandler()
// 	if err != nil {
// 		return nil, err
// 	}
// 	return imageHandler, nil
// }

// func setPublicIPHandler() (irs.PublicIPHandler, error) {
// 	var cloudDriver idrv.CloudDriver
// 	cloudDriver = new(gcpdrv.GCPDriver)

// 	config := readConfigFile()
// 	connectionInfo := idrv.ConnectionInfo{
// 		CredentialInfo: idrv.CredentialInfo{
// 			ClientId:       config.GCP.ClientId,
// 			ClientSecret:   config.GCP.ClientSecret,
// 			TenantId:       config.GCP.TenantId,
// 			SubscriptionId: config.GCP.SubscriptionID,
// 		},
// 		RegionInfo: idrv.RegionInfo{
// 			Region:        config.GCP.Location,
// 			ResourceGroup: config.GCP.GroupName,
// 		},
// 	}

// 	cloudConnection, err := cloudDriver.ConnectCloud(connectionInfo)
// 	if err != nil {
// 		return nil, err
// 	}
// 	publicIPHandler, err := cloudConnection.CreatePublicIPHandler()
// 	if err != nil {
// 		return nil, err
// 	}
// 	return publicIPHandler, nil
// }

// func setSecurityHandler() (irs.SecurityHandler, error) {
// 	var cloudDriver idrv.CloudDriver
// 	cloudDriver = new(gcpdrv.GCPDriver)

// 	config := readConfigFile()
// 	connectionInfo := idrv.ConnectionInfo{
// 		CredentialInfo: idrv.CredentialInfo{
// 			ClientId:       config.GCP.ClientId,
// 			ClientSecret:   config.GCP.ClientSecret,
// 			TenantId:       config.GCP.TenantId,
// 			SubscriptionId: config.GCP.SubscriptionID,
// 		},
// 		RegionInfo: idrv.RegionInfo{
// 			Region:        config.GCP.Location,
// 			ResourceGroup: config.GCP.GroupName,
// 		},
// 	}

// 	cloudConnection, err := cloudDriver.ConnectCloud(connectionInfo)
// 	if err != nil {
// 		return nil, err
// 	}
// 	securityHandler, err := cloudConnection.CreateSecurityHandler()
// 	if err != nil {
// 		return nil, err
// 	}
// 	return securityHandler, nil
// }

// func setVNetHandler() (irs.VNetworkHandler, error) {
// 	var cloudDriver idrv.CloudDriver
// 	cloudDriver = new(gcpdrv.GCPDriver)

// 	config := readConfigFile()
// 	connectionInfo := idrv.ConnectionInfo{
// 		CredentialInfo: idrv.CredentialInfo{
// 			ClientId:       config.GCP.ClientId,
// 			ClientSecret:   config.GCP.ClientSecret,
// 			TenantId:       config.GCP.TenantId,
// 			SubscriptionId: config.GCP.SubscriptionID,
// 		},
// 		RegionInfo: idrv.RegionInfo{
// 			Region:        config.GCP.Location,
// 			ResourceGroup: config.GCP.GroupName,
// 		},
// 	}

// 	cloudConnection, err := cloudDriver.ConnectCloud(connectionInfo)
// 	if err != nil {
// 		return nil, err
// 	}
// 	vNetHandler, err := cloudConnection.CreateVNetworkHandler()
// 	if err != nil {
// 		return nil, err
// 	}
// 	return vNetHandler, nil
// }

// func setVNicHandler() (irs.VNicHandler, error) {
// 	var cloudDriver idrv.CloudDriver
// 	cloudDriver = new(gcpdrv.GCPDriver)

// 	config := readConfigFile()
// 	connectionInfo := idrv.ConnectionInfo{
// 		CredentialInfo: idrv.CredentialInfo{
// 			ClientId:       config.GCP.ClientId,
// 			ClientSecret:   config.GCP.ClientSecret,
// 			TenantId:       config.GCP.TenantId,
// 			SubscriptionId: config.GCP.SubscriptionID,
// 		},
// 		RegionInfo: idrv.RegionInfo{
// 			Region:        config.GCP.Location,
// 			ResourceGroup: config.GCP.GroupName,
// 		},
// 	}

// 	cloudConnection, err := cloudDriver.ConnectCloud(connectionInfo)
// 	if err != nil {
// 		return nil, err
// 	}
// 	vNicHandler, err := cloudConnection.CreateVNicHandler()
// 	if err != nil {
// 		return nil, err
// 	}
// 	return vNicHandler, nil
// }

func main() {
	// Test VM Handler
	vmHandler, err := setVMHandler()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(vmHandler)
	//getVMInfo()
	//handleVM()
	//createVM()

	// Teset Resource Handler
	//testImageHandler()
	//testPublicIPHandler()
	//testSecurityHandler()
	//testVNetworkHandler()
	//testVNicHandler()
}

type Config struct {
	Type         string `json:"type"`
	ProjectID    string `json:"project_id"`
	PrivateKeyID string `json:"private_key_id"`
	PrivateKey   string `json:"private_key"`
	ClientEmail  string `json:"client_email"`
	ClientID     string `json:"client_id"`
	AuthURI      string `json:"auth_uri"`
	TokenURI     string `json:"token_uri"`
	AuthProvider string `json:"auth_provider_x509_cert_url"`
}

func readConfigFile() Config {
	// Set Environment Value of Project Root Path
	rootPath := os.Getenv("CBSPIDER_PATH")
	data, err := ioutil.ReadFile(rootPath + "/config/config.yaml")
	if err != nil {
		panic(err)
	}

	var config Config
	err = yaml.Unmarshal(data, &config)
	if err != nil {
		panic(err)
	}

	return config
}
func readFileConfig(filepath string) (Config, error) {

	data, err := ioutil.ReadFile(filepath)
	if err != nil {
		panic(err)
	}

	var config Config
	json.Unmarshal(data, &config)
	fmt.Println("readFileConfig Json : ", config.ClientEmail)

	return config, err

}
