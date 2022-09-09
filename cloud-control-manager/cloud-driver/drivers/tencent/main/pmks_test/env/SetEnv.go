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
	os.Setenv("ACCESS_KEY", "***")
	os.Setenv("ACCESS_SECRET", "***")
	os.Setenv("REGION_ID", "cn-beijing")
}
