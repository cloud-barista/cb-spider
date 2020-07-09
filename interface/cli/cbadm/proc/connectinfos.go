package proc

import (
	gc "github.com/cloud-barista/cb-spider/api-runtime/grpc-runtime/common"
	pb "github.com/cloud-barista/cb-spider/api-runtime/grpc-runtime/stub/cbspider"
	"github.com/cloud-barista/cb-spider/interface/api"
)

// ===== [ Constants and Variables ] =====

const (
	// ConfigVersion - 설정 구조에 대한 버전
	ConfigVersion = 1
)

// ===== [ Types ] =====

// ConnectInfosConfig -
type ConnectInfosConfig struct {
	Version         int           `yaml:"Version" json:"Version"`
	ConnectInfoList []ConnectInfo `yaml:"ConnectInfos" json:"ConnectInfos"`
}

// ConnectInfo -
type ConnectInfo struct {
	ConfigName   string         `yaml:"ConfigName" json:"ConfigName"`
	ProviderName string         `yaml:"ProviderName" json:"ProviderName"`
	Driver       DriverInfo     `yaml:"Driver" json:"Driver"`
	Credential   CredentialInfo `yaml:"Credential" json:"Credential"`
	Region       RegionInfo     `yaml:"Region" json:"Region"`
}

// DriverInfo -
type DriverInfo struct {
	DriverName        string `yaml:"DriverName" json:"DriverName"`
	DriverLibFileName string `yaml:"DriverLibFileName" json:"DriverLibFileName"`
}

// CredentialInfo -
type CredentialInfo struct {
	CredentialName   string         `yaml:"CredentialName" json:"CredentialName"`
	KeyValueInfoList []KeyValueInfo `yaml:"KeyValueInfoList" json:"KeyValueInfoList"`
}

// RegionInfo -
type RegionInfo struct {
	RegionName       string         `yaml:"RegionName" json:"RegionName"`
	KeyValueInfoList []KeyValueInfo `yaml:"KeyValueInfoList" json:"KeyValueInfoList"`
}

// KeyValueInfo -
type KeyValueInfo struct {
	Key   string `yaml:"Key" json:"Key"`
	Value string `yaml:"Value" json:"Value"`
}

// ===== [ Implementations ] =====

// ===== [ Private Functions ] =====

// ===== [ Public Functions ] =====

// ListConnectInfos - 연결정보 목록 통합 제공
func ListConnectInfos(cim *api.CIMApi) (string, error) {

	result, err := cim.ListConnectionConfig()
	if err != nil {
		return "", err
	}

	outType, _ := cim.GetOutType()

	var connectConfigList pb.ListConnectionConfigInfoResponse
	err = gc.ConvertToMessage(outType, result, &connectConfigList)
	if err != nil {
		return "", err
	}

	connectInfoList := []ConnectInfo{}
	for _, connectConfig := range connectConfigList.Items {

		connectInfo := ConnectInfo{}
		connectInfo.ConfigName = connectConfig.ConfigName
		connectInfo.ProviderName = connectConfig.ProviderName

		result, err := cim.GetCloudDriverByParam(connectConfig.DriverName)
		if err != nil {
			return "", err
		}

		var driverItem pb.CloudDriverInfo
		err = gc.ConvertToMessage(outType, result, &driverItem)
		if err != nil {
			return "", err
		}

		connectInfo.Driver.DriverName = driverItem.DriverName
		connectInfo.Driver.DriverLibFileName = driverItem.DriverLibFileName

		result, err = cim.GetCredentialByParam(connectConfig.CredentialName)
		if err != nil {
			return "", err
		}

		var credentialItem pb.CredentialInfo
		err = gc.ConvertToMessage(outType, result, &credentialItem)
		if err != nil {
			return "", err
		}

		connectInfo.Credential.CredentialName = credentialItem.CredentialName
		err = gc.CopySrcToDest(&credentialItem.KeyValueInfoList, &connectInfo.Credential.KeyValueInfoList)
		if err != nil {
			return "", err
		}

		result, err = cim.GetRegionByParam(connectConfig.RegionName)
		if err != nil {
			return "", err
		}

		var regionItem pb.RegionInfo
		err = gc.ConvertToMessage(outType, result, &regionItem)
		if err != nil {
			return "", err
		}
		connectInfo.Region.RegionName = regionItem.RegionName
		err = gc.CopySrcToDest(&regionItem.KeyValueInfoList, &connectInfo.Region.KeyValueInfoList)
		if err != nil {
			return "", err
		}

		connectInfoList = append(connectInfoList, connectInfo)
	}

	var cfg ConnectInfosConfig
	cfg.Version = ConfigVersion
	cfg.ConnectInfoList = connectInfoList

	return gc.ConvertToOutput(outType, &cfg)
}

// GetConnectInfos - 연결정보 통합 제공
func GetConnectInfos(cim *api.CIMApi, configName string) (string, error) {

	result, err := cim.GetConnectionConfigByParam(configName)
	if err != nil {
		return "", err
	}

	outType, _ := cim.GetOutType()

	var connectConfig pb.ConnectionConfigInfo
	err = gc.ConvertToMessage(outType, result, &connectConfig)
	if err != nil {
		return "", err
	}

	connectInfoList := []ConnectInfo{}

	connectInfo := ConnectInfo{}
	connectInfo.ConfigName = connectConfig.ConfigName
	connectInfo.ProviderName = connectConfig.ProviderName

	result, err = cim.GetCloudDriverByParam(connectConfig.DriverName)
	if err != nil {
		return "", err
	}

	var driverItem pb.CloudDriverInfo
	err = gc.ConvertToMessage(outType, result, &driverItem)
	if err != nil {
		return "", err
	}

	connectInfo.Driver.DriverName = driverItem.DriverName
	connectInfo.Driver.DriverLibFileName = driverItem.DriverLibFileName

	result, err = cim.GetCredentialByParam(connectConfig.CredentialName)
	if err != nil {
		return "", err
	}

	var credentialItem pb.CredentialInfo
	err = gc.ConvertToMessage(outType, result, &credentialItem)
	if err != nil {
		return "", err
	}

	connectInfo.Credential.CredentialName = credentialItem.CredentialName
	err = gc.CopySrcToDest(&credentialItem.KeyValueInfoList, &connectInfo.Credential.KeyValueInfoList)
	if err != nil {
		return "", err
	}

	result, err = cim.GetRegionByParam(connectConfig.RegionName)
	if err != nil {
		return "", err
	}

	var regionItem pb.RegionInfo
	err = gc.ConvertToMessage(outType, result, &regionItem)
	if err != nil {
		return "", err
	}
	connectInfo.Region.RegionName = regionItem.RegionName
	err = gc.CopySrcToDest(&regionItem.KeyValueInfoList, &connectInfo.Region.KeyValueInfoList)
	if err != nil {
		return "", err
	}

	connectInfoList = append(connectInfoList, connectInfo)

	var cfg ConnectInfosConfig
	cfg.Version = ConfigVersion
	cfg.ConnectInfoList = connectInfoList

	return gc.ConvertToOutput(outType, &cfg)
}
