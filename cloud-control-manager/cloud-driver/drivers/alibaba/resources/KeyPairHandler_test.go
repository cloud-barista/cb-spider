// Cloud Driver Interface of CB-Spider.
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// This is Resouces interfaces of Cloud Driver.
//
// by zephy@mz.co.kr, 2019.09.

package resources

import (
	"reflect"
	"testing"

	"github.com/aliyun/alibaba-cloud-sdk-go/services/ecs"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
)

func TestAlibabaKeyPairHandler_ListKey(t *testing.T) {
	type fields struct {
		Region idrv.RegionInfo
		Client *ecs.Client
	}
	tests := []struct {
		name    string
		fields  fields
		want    []*irs.KeyPairInfo
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			keyPairHandler := &AlibabaKeyPairHandler{
				Region: tt.fields.Region,
				Client: tt.fields.Client,
			}
			got, err := keyPairHandler.ListKey()
			if (err != nil) != tt.wantErr {
				t.Errorf("AlibabaKeyPairHandler.ListKey() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("AlibabaKeyPairHandler.ListKey() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestAlibabaKeyPairHandler_CreateKey(t *testing.T) {
	type fields struct {
		Region idrv.RegionInfo
		Client *ecs.Client
	}
	type args struct {
		keyPairReqInfo irs.KeyPairReqInfo
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    irs.KeyPairInfo
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			keyPairHandler := &AlibabaKeyPairHandler{
				Region: tt.fields.Region,
				Client: tt.fields.Client,
			}
			got, err := keyPairHandler.CreateKey(tt.args.keyPairReqInfo)
			if (err != nil) != tt.wantErr {
				t.Errorf("AlibabaKeyPairHandler.CreateKey() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("AlibabaKeyPairHandler.CreateKey() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestAlibabaKeyPairHandler_GetKey(t *testing.T) {
	type fields struct {
		Region idrv.RegionInfo
		Client *ecs.Client
	}
	type args struct {
		keyPairName string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    irs.KeyPairInfo
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			keyPairHandler := &AlibabaKeyPairHandler{
				Region: tt.fields.Region,
				Client: tt.fields.Client,
			}
			got, err := keyPairHandler.GetKey(tt.args.keyPairName)
			if (err != nil) != tt.wantErr {
				t.Errorf("AlibabaKeyPairHandler.GetKey() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("AlibabaKeyPairHandler.GetKey() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestAlibabaKeyPairHandler_DeleteKey(t *testing.T) {
	type fields struct {
		Region idrv.RegionInfo
		Client *ecs.Client
	}
	type args struct {
		keyPairName string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    bool
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			keyPairHandler := &AlibabaKeyPairHandler{
				Region: tt.fields.Region,
				Client: tt.fields.Client,
			}
			got, err := keyPairHandler.DeleteKey(tt.args.keyPairName)
			if (err != nil) != tt.wantErr {
				t.Errorf("AlibabaKeyPairHandler.DeleteKey() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("AlibabaKeyPairHandler.DeleteKey() = %v, want %v", got, tt.want)
			}
		})
	}
}
