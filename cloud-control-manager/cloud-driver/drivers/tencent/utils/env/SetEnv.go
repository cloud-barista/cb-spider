package env

import (
	"os"
)

func init() {
	SetEnv()
}

func SetEnv() {
	os.Setenv("SECRET_ID", "***")
	os.Setenv("SECRET_KEY", "***")
	os.Setenv("REGION_ID", "cn-beijing")
}
