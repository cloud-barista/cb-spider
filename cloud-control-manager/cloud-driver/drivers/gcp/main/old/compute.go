// Copyright 2017 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/davecgh/go-spew/spew"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"

	compute "google.golang.org/api/compute/v1"
	"google.golang.org/api/googleapi"
)

func init() {
	scopes := strings.Join([]string{
		compute.DevstorageFullControlScope,
		compute.ComputeScope,
	}, " ")
	fmt.Println("init :", scopes)
	//computeMain()
	// registerDemo("compute", scopes, computeMain)
}

const ProjectID = "mcloud-barista-251102"

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

type InstanceInfo struct {
	zone         string
	region       string
	instnaceName string
}

type vmInstanceInfo struct {
	VMId string
}
type KeyValue struct {
	Key   string
	Value string
}

type SecurityReqInfo struct {
	Name          string
	Direction     string // GCP 는 하나에 한개의 Direction만 생성/조회 가능
	SecurityRules *[]SecurityRuleInfo
}

type SecurityRuleInfo struct {
	FromPort   string
	ToPort     string
	IPProtocol string
	Direction  string
}
type VNicInfo struct {
	Id               string
	Name             string
	PublicIP         string
	MacAddress       string
	OwnedVMID        string
	SecurityGroupIds []string
	Status           string

	KeyValueList []KeyValue
}
type SecurityInfo struct {
	Id            string
	Name          string
	Direction     string // GCP 는 하나에 한개의 Direction만 생성/조회 가능
	SecurityRules *[]SecurityRuleInfo

	KeyValueList []KeyValue
}
type PublicIPInfo struct {
	Name string // AWS
	Id   string
	// @todo

	Domain                  string // AWS
	PublicIp                string // AWS
	PublicIpv4Pool          string // AWS
	AllocationId            string // AWS:할당ID
	AssociationId           string // AWS:연결ID
	InstanceId              string // AWS:연결된 VM, GCP:연결된 VM name
	NetworkInterfaceId      string // AWS:연결된 Nic
	NetworkInterfaceOwnerId string // AWS
	PrivateIpAddress        string // AWS

	Region            string // GCP
	CreationTimestamp string // GCP
	Address           string // GCP
	NetworkTier       string // GCP : PREMIUM, STANDARD
	AddressType       string // GCP : External, INTERNAL, UNSPECIFIED_TYPE
	Status            string // GCP : IN_USE, RESERVED, RESERVING
	KeyValueList      []KeyValue
}

type CredentialInfo struct {
	// @todo TBD
	// key-value pairs
	ClientId         string // Azure Credential
	ClientSecret     string // Azure Credential
	TenantId         string // Azure Credential
	SubscriptionId   string // Azure Credential
	IdentityEndpoint string // OpenStack Credential
	Username         string // OpenStack Credential
	Password         string // OpenStack Credential
	DomainName       string // OpenStack Credential
	ProjectID        string // OpenStack Credential
	AuthToken        string // Cloudit Credential
	Client_Email     string // GCP
	Private_Key      string // GCP

}

func createInstance(service *compute.Service, conf Config, zone string, vmname string, diskname string) {

	projectID := conf.ProjectID

	prefix := "https://www.googleapis.com/compute/v1/projects/" + projectID
	imageURL := "https://www.googleapis.com/compute/v1/projects/debian-cloud/global/images/debian-7-wheezy-v20140606"
	zone = zone
	instanceName := vmname

	// Show the current images that are available.
	// res, err := service.Images.List(projectID).Do()
	// log.Printf("Got compute.Images.List, err: %#v, %v", res, err)

	instance := &compute.Instance{
		Name:        instanceName,
		Description: "compute sample instance",
		MachineType: prefix + "/zones/" + zone + "/machineTypes/n1-standard-1",
		Disks: []*compute.AttachedDisk{
			{
				AutoDelete: true,
				Boot:       true,
				Type:       "PERSISTENT",
				InitializeParams: &compute.AttachedDiskInitializeParams{
					DiskName:    diskname,
					SourceImage: imageURL,
				},
			},
		},
		NetworkInterfaces: []*compute.NetworkInterface{
			{
				AccessConfigs: []*compute.AccessConfig{
					{
						Type: "ONE_TO_ONE_NAT",
						Name: "External NAT",
					},
				},
				Network: prefix + "/global/networks/default",
			},
		},
		ServiceAccounts: []*compute.ServiceAccount{
			{
				Email: conf.ClientEmail,
				Scopes: []string{
					compute.DevstorageFullControlScope,
					compute.ComputeScope,
				},
			},
		},
	}

	op, err := service.Instances.Insert(projectID, zone, instance).Do()
	js, err := op.MarshalJSON()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("Insert vm to marshal Json : ", string(js))
	log.Printf("Got compute.Operation, err: %#v, %v", op, err)
	etag := op.Header.Get("Etag")
	log.Printf("Etag=%v", etag)

	inst, err := service.Instances.Get(projectID, zone, instanceName).IfNoneMatch(etag).Do()
	log.Printf("Got compute.Instance, err: %#v, %v", inst, err)
	if googleapi.IsNotModified(err) {
		log.Printf("Instance not modified since insert.")
	} else {
		log.Printf("Instance modified since insert.")
	}

}
func getPublicIPFromInstance(instance *compute.Instance) {
	fmt.Println("network Interface : ", instance.NetworkInterfaces[0].AccessConfigs[0].Name)
}
func getPublicIP(ctx context.Context, service *compute.Service, region string, publicNm string, conf Config) {
	info, err := service.Addresses.Get(conf.ProjectID, region, publicNm).Do()
	if err != nil {
		log.Fatal(err)
	}

	infoByte, err := info.MarshalJSON()
	var result map[string]interface{}

	fmt.Println("infoByte : ", string(infoByte))
	if err != nil {
		log.Fatal(err)
	}

	var publicInfo PublicIPInfo
	err = json.Unmarshal(infoByte, &publicInfo)
	//key value 담아서 넣기
	json.Unmarshal(infoByte, &result)
	var keyValueList []KeyValue
	for k, v := range result {
		fmt.Println("key : ", k)
		fmt.Println("value : ", v)
		keyValueList = append(keyValueList, KeyValue{k, v.(string)})
	}
	fmt.Println("KeyValueList : ", keyValueList)
	fmt.Println("publicInfo addressip : ", publicInfo.Address)

	//getkeyvaluelist test
	kl := GetKeyValueList(result)
	fmt.Println("GetKeyValueList : ", kl)
	getValue := GetKeyValue(kl, "address")
	fmt.Println("getValue :", getValue)
	if users := info.Users; users != nil {
		vmArr := strings.Split(users[0], "/")
		publicInfo.InstanceId = vmArr[len(vmArr)-1]
	}

	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("publicInfo : ", publicInfo)

}
func GetKeyValueList(i map[string]interface{}) []KeyValue {
	var keyValueList []KeyValue
	for k, v := range i {
		keyValueList = append(keyValueList, KeyValue{k, v.(string)})
		fmt.Println("getKeyValueList : ", keyValueList)
	}

	return keyValueList
}

func GetKeyValue(keyValusList []KeyValue, key string) interface{} {
	var getValue string
	for _, v := range keyValusList {
		fmt.Println(v.Key)
		if v.Key != "" && v.Key == key {
			getValue = v.Value
			return getValue
		}
	}
	return nil
}

func getInstance(ctx context.Context, service *compute.Service, zone string, instanceName string, conf Config) *compute.Instance {
	/// ctx := context.Background()
	inst, err := service.Instances.Get(conf.ProjectID, zone, instanceName).Context(ctx).Do()
	//log.Printf("Got compute.Instance, err: %#v, %v", inst, err)
	if err != nil {
		log.Fatal(err)
	}
	js, err := inst.MarshalJSON()
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("get Instance Marshal Json : ", string(js))
	fmt.Println("Instance status :", inst.Status)

	return inst
}

func stopVM(ctx context.Context, service *compute.Service, zone string, instanceName string, conf Config) (string, error) {
	// ctx := context.Background()

	inst, err := service.Instances.Stop(conf.ProjectID, zone, instanceName).Context(ctx).Do()
	if err != nil {
		log.Fatal(err)
	}
	js, err := inst.MarshalJSON()
	fmt.Println("Instance marshaljson :", string(js))
	log.Printf("Instances Stop, err: %#v, %v", inst, err)
	fmt.Println("Instance Status :", inst.Status)
	return inst.Status, err
}

func startVM(ctx context.Context, service *compute.Service, zone string, instanceName string, conf Config) (string, error) {

	getInst := getInstance(ctx, service, zone, instanceName, conf)

	if getInst.Status == "TERMINATED" {
		fmt.Println("터미네이터 이다.")
	}

	inst, err := service.Instances.Start(conf.ProjectID, zone, instanceName).Context(ctx).Do()
	js, err := inst.MarshalJSON()
	fmt.Println("Instance marshaljson :", string(js))
	log.Printf("StartVM, err: %#v, %v", inst, err)
	fmt.Println("Status :", inst.Status)
	fmt.Println("VM type : ", reflect.TypeOf(inst))
	return inst.Status, err
}

func deleteVM(ctx context.Context, service *compute.Service, zone string, instanceName string, conf Config) (string, error) {
	//ctx := context.Background()
	inst, err := service.Instances.Delete(conf.ProjectID, zone, instanceName).Context(ctx).Do()
	js, err := inst.MarshalJSON()
	fmt.Println("Instance marshaljson :", string(js))
	log.Printf("StartVM, err: %#v, %v", inst, err)
	fmt.Println("Status :", inst.Status)
	fmt.Println("VM type : ", reflect.TypeOf(inst))
	return inst.Status, err
}

func rebootVM(ctx context.Context, service *compute.Service, zone string, instanceName string, conf Config) (string, error) {
	//ctx := context.Background()
	st, err := stopVM(ctx, service, zone, instanceName, conf)
	if err != nil {
		log.Fatal(err)
	}

	return st, err
}

func ListPublicIP(ctx context.Context, service *compute.Service, conf Config, region string) (string, string) {
	list, err := service.Addresses.List(conf.ProjectID, region).Context(ctx).Do()
	listInfo, err := list.MarshalJSON()
	if err != nil {
		log.Fatal(err)
	}

	var result map[string]interface{}
	json.Unmarshal(listInfo, &result)

	fmt.Println("ListPublicIP Info : ", result)
	// fmt.Println("ListPublicIP[0] Name : ", list.Items[0].Name)
	// fmt.Println("ListPublicIP[0] Address : ", list.Items[0].Address)
	//log.Printf("getGlovalAddressList, err: %#v, %v", list, err)
	var publicInfoArr []*PublicIPInfo

	var rejson map[string]interface{}
	for _, item := range list.Items {
		// fmt.Println("index : ", index)
		fmt.Println("item : ", item)
		var publicIPInfos PublicIPInfo
		publicIPInfos.Name = item.Name
		publicIPInfos.Id = strconv.FormatUint(item.Id, 10)
		publicIPInfos.Region = item.Region
		publicIPInfos.CreationTimestamp = item.CreationTimestamp
		publicIPInfos.Address = item.Address
		publicIPInfos.NetworkTier = item.Network
		if user := item.Users; user != nil {
			publicIPInfos.InstanceId = user[0]
		}

		it := item
		fmt.Println("it :", it)
		// bts := json.Marshal(it)
		// json.Unmarshal(bts, &rejson)
		// publicIPInfos[index].InstanceId = item.Users[0]
		publicIPInfos.Status = item.Status
		publicInfoArr = append(publicInfoArr, &publicIPInfos)

	}

	fmt.Println("rejson : ", rejson)
	// for _, st := range publicIPInfos {

	// 	if st.Status == "RESERVED" {
	// 		fmt.Println(st.Status)
	// 	}
	// }
	fmt.Println("publicInfos Arr : ", publicInfoArr)
	name := list.Items[0].Name
	address := list.Items[0].Address
	return name, address
}
func getGlobalAddressList(ctx context.Context, service *compute.Service, config Config) {

	list, err := service.GlobalAddresses.List(config.ProjectID).Context(ctx).Do()
	log.Printf("getGlovalAddressList, err: %#v, %v", list, err)

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

func connect(filePath string) *compute.Service {
	gcpType := "service_account"
	clientEmail := "675581125193-compute@developer.gserviceaccount.com"
	privateKey := "-----BEGIN PRIVATE KEY-----\nMIIEvAIBADANBgkqhkiG9w0BAQEFAASCBKYwggSiAgEAAoIBAQCT+RlOV3L1si2q\npcjZj7jx+6MU24GjyDOjCL4Lo67scAP5QWHePzvqndIQzN1LqLeIJVtcYKLEzwER\nvM0wSzCnW768k9ek+Rpfu5znbas7wF9p36v9Z0qL3BaimkRxSb1kI6ENz7qbM/EY\nepS3do+G2+GeOKEA4JfAhGVQiit0EUEKQ1m7WB2izobjkgCZooaGt1suapG3VUzN\n8PYmn6mB0+Ls21TO/wTGZEhTkBzUN1S0+bc4Wum8M93mJ3n2VouGu6xXaqXgDdIu\nZZDqPdWGDvXzEBvk/4imtBt7+Chs4J9dvk48e0B4rvFx4H8HBWA6oFysZgwZkR51\nupf8WKxxAgMBAAECggEAMaVNnD8yzsQtFifxLy1NO8LVgFX1NOIikPyJ5pXQRnt+\nmc4Z69sDW0AADqrtREki6oa+FExH0Agzr6PMo/tWI5BgehyQKUV6V8w2ZF8jKDTu\nzjLBHY/eLvZ0kbF4bRn0dPiPPHcJgLD4nuHhq3wXw4NaOx98xTKVN340D8WLtrDg\nmrFnytZNyRqS/eHbNQtOWclMffbdA6hDJBBUA/J7bWnSkcjg+lccO+zs+yZgJ5wr\nKiN83dgXORimbjUWcdNnxoSC7HfgqmqziYg1s1MrdlVe9eL8fMi1Pz9jUnZwtk0D\nzJKDp8xbEsX4AXOHJK2KMsy3zJvUfgRH+J+L0ytPUQKBgQDI6UlaTTh4MRCgsVPr\n8yjiseCrAwXlxDyVX1Oy1ud5lkW1VvmxTs0KN8DScCj4jkXWKct2mh77v6Wg81MU\nxuUF8nC4bSYfLtahMmcaCN2Ccad4QaWGnEmKit6apOW+HbgU2i8pBvmcvv9JvJ59\n+p8jOKa/e9aVoY4zCNwxGXFYiwKBgQC8i+E9XdiSZ1ownHXz+GkdRoWsx0X+Tdkp\nS0XxEGaORh8DfXAo73O08eoBOjftS+aQfj/qEu31JQb0i07qZAAaZbf1WO8aG97G\n8JZLV5Aez0iNiSgNZJfKgjxlG+IVlpP0oWJXbpomIkWsjuONLYDX5f+jUlQeYb3U\n97O83wBycwKBgFr2hGeGHtMMI+MdZkmlxhUdRAMpUzo8JtHaXyLReevqxZTc1CAa\n9Wpy47JjZaljgOr98Ui5bt28X1kH0c3OX1LZ+X8GrAPiSPqiv1tiOCgfHRutXSwd\nBo7bYP3TOtFg0z9dqYyBw/Hb5+mSpI+VMQfZVmXLw9PrWV5x3H++bTsRAoGAKtOC\n99NnK+n53GzNhfr4tUOdfV9OELNSDkUgv96/zLU0ujA117Z8C6+fPWQh6+5/knZ6\nwgpGrpYYfFdgN3E7bMOKA1qOBNorwfhHyxk6jST8D9oFlPUyXTczzKuGsOyg8sHt\nenqO3PaP6OAT469gQqnlZQ2AOd5tpgAVfWMR0O0CgYBPJ6DSGlGlzCHpqpg2JGzO\nn3kXVxVvQA58cfZWz7hzmjyJr9B2bPFfeSidLJEBHjujW28663+9NVp7xwZRotir\nFw8k3/z97EKadjrvZB6m2CPS7NFFWDgDqSPz1YyNYxGyJynT5GIKpLFdqcMMc9Bk\nT9NsVtofa1Iu7Vos4vd+NA==\n-----END PRIVATE KEY-----\n"
	data := make(map[string]string)
	data["type"] = gcpType
	data["private_key"] = privateKey
	data["client_email"] = clientEmail

	res, err := json.Marshal(data)

	dt, err := ioutil.ReadFile(filePath)

	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(dt)
	// s := string(data)
	// d := []byte(s)
	conf, err := google.JWTConfigFromJSON(res, "https://www.googleapis.com/auth/compute")
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("Connection Success : ", reflect.TypeOf(res))

	client := conf.Client(oauth2.NoContext)

	computeService, err := compute.New(client)

	return computeService

}

func CreatePublicIP(ctx context.Context, service *compute.Service, name string, region string, conf Config) {
	address := &compute.Address{
		Name: name,
	}
	info, err := service.Addresses.Insert(conf.ProjectID, region, address).Context(ctx).Do()
	if err != nil {
		log.Fatal(err)
	}
	infoJson, err := info.MarshalJSON()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("createPublicIP Info : ", string(infoJson))
	time.Sleep(time.Second * 3)

	getPublicIP(ctx, service, region, name, conf)

}

func ListVM(ctx context.Context, service *compute.Service, zone string, conf Config) []byte {
	list, err := service.Instances.List(conf.ProjectID, zone).Do()
	if err != nil {
		log.Fatal(err)
	}

	listJson, err := list.MarshalJSON()
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("List Vm : ", string(listJson))

	return listJson
}

func ListImage(ctx context.Context, service *compute.Service, conf Config) []byte {
	projectID := conf.ProjectID
	list, err := service.Images.List(projectID).Do()
	log.Printf("Got compute.Images.List, err: %#v, %v", list, err)
	req := service.Images.List(projectID)
	if err := req.Pages(ctx, func(page *compute.ImageList) error {
		for i, image := range page.Items {
			// TODO: Change code below to process each `image` resource:
			fmt.Printf("get ImagetList : %#v\n", image, i)
		}
		return nil
	}); err != nil {
		log.Fatal(err)
	}

	if err != nil {
		log.Fatal(err)
	}

	imageListJson, err := list.MarshalJSON()
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("List Vm : ", string(imageListJson))

	return imageListJson
}

func createVnet(ctx context.Context, service *compute.Service, conf Config, name string) {
	network := &compute.Network{
		Name:                  name,
		AutoCreateSubnetworks: true,
	}

	res, err1 := service.Networks.Insert(conf.ProjectID, network).Context(ctx).Do()
	if err1 != nil {
		fmt.Println("create vnet error")
		log.Fatal(err1)
	}
	fmt.Println("result", res)
	time.Sleep(time.Second * 15)
	info, err2 := service.Networks.Get(conf.ProjectID, name).Context(ctx).Do()
	if err2 != nil {
		fmt.Println("Get Vnetwork error")
		log.Fatal(err2)
	}

	js, _ := info.MarshalJSON()
	fmt.Println("getVnet : ", string(js))

}

func getVnet(ctx context.Context, service *compute.Service, conf Config, name string) {
	info, err2 := service.Networks.Get(ProjectID, name).Context(ctx).Do()
	if err2 != nil {
		fmt.Println("Get Vnetwork error")
		log.Fatal(err2)
	}

	js, _ := info.MarshalJSON()
	fmt.Println("getVnet : ", string(js))
}

func getFireWall(service *compute.Service, name string) {

	security, err := service.Firewalls.Get(ProjectID, name).Do()
	if err != nil {
		log.Fatal(err)

	}

	// //전부 keyvalue 저장
	// var result map[string]interface{}
	// var keyValueList []KeyValue
	// security.Id = strconv.FormatUint(security.Id, 10)
	mjs, _ := security.MarshalJSON()
	fmt.Println(string(mjs))
	//json.Unmarshal(mjs, &result)

	// for k, v := range result {
	// 	keyValueList = append(keyValueList, KeyValue{
	// 		Key: k, Value: v.(string),
	// 	})
	// }
	// fmt.Println(result)
	// var securityRules irs.SecurityRuleInfogi
	// securityInfo := irs.SecurityInfo{
	// 	Id: strconv.FormatUint(security.Id,10),
	// 	Name: security.Name,
	// 	KeyValueList: keyValueList,

	// }

}

func createFireWall(securityReqInfo SecurityReqInfo, service *compute.Service) {
	ports := *securityReqInfo.SecurityRules
	fmt.Println("ports : ", ports)
	var firewallAllowed []*compute.FirewallAllowed

	// fmt.Println(reflect.TypeOf(t))
	// t = append(t, &compute.FirewallAllowed{
	// 	IPProtocol: "tcp",
	// })

	// fmt.Println(t)
	// for _, item := range ports {
	// 	var port string
	// 	fp := item.FromPort
	// 	tp := item.ToPort

	// 	if tp != "" && fp != "" {
	// 		port = fp + "-" + tp
	// 	}
	// 	if tp != "" && fp == "" {
	// 		port = tp
	// 	}
	// 	if tp == "" && fp != "" {
	// 		port = fp
	// 	}
	// 	// if tp == "" && fp == "" {
	// 	// 	port = ""
	// 	// }
	// 	fmt.Println(port)
	// 	t = append(t, &compute.FirewallAllowed{
	// 		IPProtocol: item.IPProtocol,
	// 		Ports:      []string{port},
	// 	})
	// }
	// fmt.Println(t[0])

	for _, item := range ports {
		var port string
		fp := item.FromPort
		tp := item.ToPort

		if tp != "" && fp != "" {
			port = fp + "-" + tp
		}
		if tp != "" && fp == "" {
			port = tp
		}
		if tp == "" && fp != "" {
			port = fp
		}

		firewallAllowed = append(firewallAllowed, &compute.FirewallAllowed{
			IPProtocol: item.IPProtocol,
			Ports: []string{
				port,
			},
		})
	}
	fireWall := &compute.Firewall{
		Allowed:   firewallAllowed,
		Direction: securityReqInfo.Direction, //INGRESS(inbound), EGRESS(outbound)
		SourceRanges: []string{
			"0.0.0.0/0",
		},
		Name: securityReqInfo.Name,
	}

	res, err := service.Firewalls.Insert(ProjectID, fireWall).Do()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("create result : ", res)
}
func getVNic(service *compute.Service) {
	res, err := service.Instances.Get(ProjectID, "asia-northeast1-b", "2578782397763975033").Do()
	if err != nil {
		log.Fatal(err)
	}
	networkInfo := mappingNetworkInfo(res)
	spew.Dump(networkInfo)
	fmt.Println("networkInfo : ", networkInfo)
}
func listVNic(service *compute.Service) {

	res, err := service.Instances.List(ProjectID, "asia-northeast1-b").Do()
	if err != nil {
		log.Fatal(err)
	}
	var vNicInfo []*VNicInfo
	for _, item := range res.Items {
		info := mappingNetworkInfo(item)
		vNicInfo = append(vNicInfo, &info)
	}
	spew.Dump(vNicInfo)
	fmt.Println("networkInfo : ", vNicInfo)

}
func mappingNetworkInfo(res *compute.Instance) VNicInfo {
	netWorkInfo := VNicInfo{
		Id:        strconv.FormatUint(res.Id, 10),
		Name:      res.NetworkInterfaces[0].Name,
		PublicIP:  res.NetworkInterfaces[0].AccessConfigs[0].NatIP,
		OwnedVMID: strconv.FormatUint(res.Id, 10),
		Status:    res.Status, //nic 상태를 알 수 있는 것이 없어서 Instance의 상태값을 가져다 넣어줌
		KeyValueList: []KeyValue{
			{"Network", res.NetworkInterfaces[0].Network},
			{"NetworkIP", res.NetworkInterfaces[0].NetworkIP},
			{"PublicIPName", res.NetworkInterfaces[0].AccessConfigs[0].Name},
			{"NetworkTier", res.NetworkInterfaces[0].AccessConfigs[0].NetworkTier},
			{"Network", res.NetworkInterfaces[0].Network},
		},
	}

	return netWorkInfo

}
func main() {
	credentialFilePath := os.Getenv("GOOGLE_APPLICATION_CREDENTIALS")
	//config, _ := readFileConfig(credentialFilePath)

	// securityReq := SecurityReqInfo{
	// 	Name:      "firewalltest",
	// 	Direction: "INGRESS",
	// 	SecurityRules: &[]SecurityRuleInfo{
	// 		{FromPort: "22", ToPort: "25", IPProtocol: "tcp"},
	// 		{FromPort: "65234", ToPort: "", IPProtocol: "udp"},
	// 	},
	// }

	client := connect(credentialFilePath)
	//createFireWall(securityReq, client)
	//getVNic(client)
	listVNic(client)
	//zone := "asia-northeast1-b"
	//instanceName := "cscmcloud"
	//diskname := "mzcsc21"
	//region := "asia-northeast1"
	//ctx := context.Background()

	// getFireWall(client, "firewall1")
	// fmt.Println(reflect.TypeOf(client))
	// fmt.Println("config Project ID : ", config.ProjectID)

	//createInstance(client, config, zone, instanceName, diskname)
	//instance := getInstance(ctx, client, zone, instanceName, config)
	//fmt.Println("output instance : ", instance)
	//getInstance(ctx, client, zone, instanceName, config)
	//stopVM(ctx, client, zone, instanceName, config)
	//startVM(ctx, client, zone, instanceName, config)
	//getGlobalAddressList(ctx, client, config)
	//getPublicIP(ctx, client, region, "natip", config)
	//CreatePublicIP(ctx, client, "publicip6", region, config)
	//getPublicIP(ctx, client, region, "publicip6", config)
	// name, address := ListPublicIP(ctx, client, config, region)
	// fmt.Println("output name : ", name)
	// fmt.Println("output address : ", address)
	//createVnet(ctx, client, config, "mynetwork2")
	//getVnet(ctx, client, config, "test1")
	//getVMlist := ListVM(ctx, client, zone, config)
	//fmt.Println("getVMList : ", string(getVMlist))
	//getImagelist := ListImage(ctx, client, config)
	//fmt.Println("getVMList : ", string(getImagelist))

}
