package env

import (
	"os"
)

func init() {
	SetEnv()
}

func SetEnv() {
	os.Setenv("CBSPIDER_ROOT", "/path")
	os.Setenv("CBLOG_ROOT", "/path")
	os.Setenv("CLIENT_ID", "***")
	os.Setenv("CLIENT_SECRET", "***")
	os.Setenv("REGION", "cn-beijing")
	os.Setenv("ZONE", "ap-beijing-2")
}
