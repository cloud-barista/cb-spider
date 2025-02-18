package main

import "github.com/cloud-barista/cb-spider/permissionTest/cloudTest"

//const baseURL = "http://localhost:1024/spider"

func main(){
	cloudTest.RunVPCTest()
	// cloudTest.SubnetTest()
	// cloudTest.SecurityGroupTest() 
	// cloudTest.VMTest() 

}