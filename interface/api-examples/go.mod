module spider/examples

go 1.16

replace (
	github.com/coreos/bbolt => go.etcd.io/bbolt v1.3.3
	github.com/coreos/go-systemd => github.com/coreos/go-systemd/v22 v22.0.0
	github.com/dgrijalva/jwt-go => github.com/golang-jwt/jwt v3.2.1+incompatible
	gopkg.in/yaml.v3 => gopkg.in/yaml.v3 v3.0.0
)

require github.com/cloud-barista/cb-spider v0.5.6
