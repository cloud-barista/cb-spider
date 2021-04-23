package resources

import (
	"crypto/md5"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/bramvdbogaerde/go-scp"
	"github.com/bramvdbogaerde/go-scp/auth"
	cblog "github.com/cloud-barista/cb-log"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	"github.com/sirupsen/logrus"
	"golang.org/x/crypto/ssh"

	call "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/call-log"
	"github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/drivers/cloudit/client"
	"github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/drivers/cloudit/client/ace/nic"
	"github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/drivers/cloudit/client/ace/specs"
)

const (
	CBVMUser      = "cb-user"
	CBKeyPairPath = "/cloud-driver-libs/.ssh-cloudit/"
)

var once sync.Once
var cblogger *logrus.Logger
var calllogger *logrus.Logger

func InitLog() {
	once.Do(func() {
		// cblog is a global variable.
		cblogger = cblog.GetLogger("CB-SPIDER")
		calllogger = call.GetLogger("HISCALL")
	})
}

func LoggingError(hiscallInfo call.CLOUDLOGSCHEMA, err error) {
	cblogger.Error(err.Error())
	hiscallInfo.ErrorMSG = err.Error()
	calllogger.Info(call.String(hiscallInfo))
}

func LoggingInfo(hiscallInfo call.CLOUDLOGSCHEMA, start time.Time) {
	hiscallInfo.ElapsedTime = call.Elapsed(start)
	calllogger.Info(call.String(hiscallInfo))
}

func GetCallLogScheme(endpoint string, resourceType call.RES_TYPE, resourceName string, apiName string) call.CLOUDLOGSCHEMA {
	cblogger.Info(fmt.Sprintf("Call %s %s", call.CLOUDIT, apiName))
	return call.CLOUDLOGSCHEMA{
		CloudOS:      call.CLOUDIT,
		RegionZone:   endpoint,
		ResourceType: resourceType,
		ResourceName: resourceName,
		CloudOSAPI:   apiName,
	}
}

// VM Spec 정보 조회
func GetVMSpecByName(authHeader map[string]string, reqClient *client.RestClient, specName string) (*string, error) {
	requestOpts := client.RequestOpts{
		MoreHeaders: authHeader,
	}
	specList, err := specs.List(reqClient, &requestOpts)
	if err != nil {
		cblogger.Error(fmt.Sprintf("failed to get security group list, err : %s", err))
		return nil, err
	}

	specInfo := specs.VMSpecInfo{}
	for _, s := range *specList {
		if strings.EqualFold(specName, s.Name) {
			specInfo = s
			break
		}
	}

	// VM Spec 정보가 없을 경우 에러 리턴
	if specInfo.Id == "" {
		cblogger.Error(fmt.Sprintf("failed to get image, err : %s", err))
		return nil, err
	}
	return &specInfo.Id, nil
}

// VNic 목록 조회
func ListVNic(authHeader map[string]string, reqClient *client.RestClient, vmId string) (*[]nic.VmNicInfo, error) {
	requestOpts := client.RequestOpts{
		MoreHeaders: authHeader,
	}
	vNicList, err := nic.List(reqClient, vmId, &requestOpts)
	if err != nil {
		return nil, err
	}
	return vNicList, nil
}

// KeyPair 해시 생성 함수
func CreateHashString(credentialInfo idrv.CredentialInfo) (string, error) {
	keyString := credentialInfo.IdentityEndpoint + credentialInfo.AuthToken + credentialInfo.TenantId
	hasher := md5.New()
	_, err := io.WriteString(hasher, keyString)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", hasher.Sum(nil)), nil
}

// Public KeyPair 정보 가져오기
func GetPublicKey(credentialInfo idrv.CredentialInfo, keyPairName string) (string, error) {
	keyPairPath := os.Getenv("CBSPIDER_ROOT") + CBKeyPairPath
	hashString, err := CreateHashString(credentialInfo)
	if err != nil {
		return "", err
	}

	publicKeyPath := keyPairPath + hashString + "--" + keyPairName + ".pub"
	publicKeyBytes, err := ioutil.ReadFile(publicKeyPath)
	if err != nil {
		return "", err
	}
	return string(publicKeyBytes), nil
}

func GetSSHClient(serverIp string, serverPort int, username string, password string) (scp.Client, error) {
	clientConfig, err := auth.PasswordKey(username, password, ssh.InsecureIgnoreHostKey())
	if err != nil {
		return scp.Client{}, err
	}
	sshClient := scp.NewClient(fmt.Sprintf("%s:%d", serverIp, serverPort), &clientConfig)
	err = sshClient.Connect()
	return sshClient, err
}

func RunCommand(serverIp string, serverPort int, username string, password string, command string) (string, error) {
	clientConfig, err := auth.PasswordKey(username, password, ssh.InsecureIgnoreHostKey())
	if err != nil {
		return "", err
	}
	sshClient := scp.NewClient(fmt.Sprintf("%s:%d", serverIp, serverPort), &clientConfig)
	err = sshClient.Connect()
	if err != nil {
		return "", err
	}
	defer sshClient.Close()

	session := sshClient.Session
	sshOut, err := session.StdoutPipe()
	if err != nil {
		return "", err
	}
	session.Stderr = os.Stderr

	err = session.Run(command)
	if err != nil {
		fmt.Println(err)
		return "", err
	}
	defer session.Close()
	return stdoutToString(sshOut), err
}

func stdoutToString(sshOut io.Reader) string {
	buf := make([]byte, 1000)
	num, err := sshOut.Read(buf)
	outStr := ""
	if err == nil {
		outStr = string(buf[:num])
	}
	for err == nil {
		num, err = sshOut.Read(buf)
		outStr += string(buf[:num])
		if err != nil {
			if err.Error() != "EOF" {
				//cblog.Error(err)
			}
		}

	}
	return strings.Trim(outStr, "\n")
}
