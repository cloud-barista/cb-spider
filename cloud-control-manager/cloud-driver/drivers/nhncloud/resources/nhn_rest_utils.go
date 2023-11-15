package resources

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// (변경필요) 기존에 받아온 토큰이 있으면 반환하고 없으면 새로 받아온 토큰을 반환한다.
func GetToken(host string, tenant_id string, username string, password string) (string, error) {
	url := "/v2.0/tokens"

	data := `{
		"auth": {
			"tenantId": "%s",
			"passwordCredentials": {
				"username": "%s",
				"password": "%s"
			}
		}
	}`
	data = fmt.Sprintf(data, tenant_id, username, password)

	req, err := http.NewRequest(http.MethodPost, host+url, bytes.NewBuffer([]byte(data)))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer res.Body.Close()

	bytes, _ := io.ReadAll(res.Body)

	var payload map[string]interface{}
	json.Unmarshal(bytes, &payload)
	token := payload["access"].(map[string]interface{})["token"].(map[string]interface{})["id"].(string)

	return token, nil
}

func CreateCluster(host string, token string, payload string) (string, error) {

	url := "clusters"

	req, err := http.NewRequest(http.MethodPost, host+url, bytes.NewBuffer([]byte(payload)))
	if err != nil {
		panic(err)
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("OpenStack-API-Version", "container-infra latest")
	req.Header.Set("X-Auth-Token", token)

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer res.Body.Close()

	bytes, _ := io.ReadAll(res.Body)

	return string(bytes), err
}

func GetClusters(host string, token string) (string, error) {

	url := "clusters"

	req, err := http.NewRequest(http.MethodGet, host+url, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("OpenStack-API-Version", "container-infra latest")
	req.Header.Set("X-Auth-Token", token)

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer res.Body.Close()

	bytes, _ := io.ReadAll(res.Body)

	return string(bytes), err
}

func GetCluster(host string, token string, cluster_id_or_name string) (string, error) {

	url := "clusters/" + cluster_id_or_name

	req, err := http.NewRequest(http.MethodGet, host+url, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("OpenStack-API-Version", "container-infra latest")
	req.Header.Set("X-Auth-Token", token)

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer res.Body.Close()

	bytes, _ := io.ReadAll(res.Body)

	return string(bytes), nil
}

func DeleteCluster(host string, token string, cluster_id_or_name string) (string, error) {

	url := "clusters/" + cluster_id_or_name

	req, err := http.NewRequest(http.MethodDelete, host+url, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("OpenStack-API-Version", "container-infra latest")
	req.Header.Set("X-Auth-Token", token)

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer res.Body.Close()

	bytes, _ := io.ReadAll(res.Body)

	return string(bytes), nil
}

func ResizeCluster(host string, token string, cluster_id_or_name string, payload string) (string, error) {

	url := "clusters/" + cluster_id_or_name + "/actions/resize"

	req, err := http.NewRequest(http.MethodPost, host+url, bytes.NewBuffer([]byte(payload)))
	if err != nil {
		return "", err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("OpenStack-API-Version", "container-infra latest")
	req.Header.Set("X-Auth-Token", token)

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer res.Body.Close()

	bytes, _ := io.ReadAll(res.Body)

	return string(bytes), nil
}

func GetNodeGroups(host string, token string, cluster_id_or_name string) (string, error) {

	url := "clusters/" + cluster_id_or_name + "/nodegroups"

	req, err := http.NewRequest(http.MethodGet, host+url, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("OpenStack-API-Version", "container-infra latest")
	req.Header.Set("X-Auth-Token", token)

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer res.Body.Close()

	bytes, _ := io.ReadAll(res.Body)

	return string(bytes), nil
}

func GetNodeGroup(host string, token string, cluster_id_or_name string, node_group_id_or_name string) (string, error) {

	url := "clusters/" + cluster_id_or_name + "/nodegroups/" + node_group_id_or_name

	req, err := http.NewRequest(http.MethodGet, host+url, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("OpenStack-API-Version", "container-infra latest")
	req.Header.Set("X-Auth-Token", token)

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer res.Body.Close()

	bytes, _ := io.ReadAll(res.Body)

	return string(bytes), nil
}

func CreateNodeGroup(host string, token string, cluster_id_or_name string, payload string) (string, error) {

	url := "clusters/" + cluster_id_or_name + "/nodegroups"

	req, err := http.NewRequest(http.MethodPost, host+url, bytes.NewBuffer([]byte(payload)))
	if err != nil {
		return "", err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("OpenStack-API-Version", "container-infra latest")
	req.Header.Set("X-Auth-Token", token)

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer res.Body.Close()

	bytes, _ := io.ReadAll(res.Body)

	return string(bytes), nil
}

func DeleteNodeGroup(host string, token string, cluster_id_or_name string, node_group_id_or_name string) (string, error) {

	url := "clusters/" + cluster_id_or_name + "/nodegroups/" + node_group_id_or_name

	req, err := http.NewRequest(http.MethodDelete, host+url, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("OpenStack-API-Version", "container-infra latest")
	req.Header.Set("X-Auth-Token", token)

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer res.Body.Close()

	bytes, _ := io.ReadAll(res.Body)

	return string(bytes), nil
}

func ChangeNodeGroupAutoScaler(host string, token string, cluster_id_or_name string, node_group_id_or_name string, payload string) (string, error) {

	url := "clusters/" + cluster_id_or_name + "/nodegroups/" + node_group_id_or_name + "/autoscale"

	req, err := http.NewRequest(http.MethodPost, host+url, bytes.NewBuffer([]byte(payload)))
	if err != nil {
		return "", err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("OpenStack-API-Version", "container-infra latest")
	req.Header.Set("X-Auth-Token", token)

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer res.Body.Close()

	bytes, _ := io.ReadAll(res.Body)

	return string(bytes), nil
}

func UpgradeCluster(host string, token string, cluster_id_or_name string, node_group_id_or_name string, payload string) (string, error) {

	url := "clusters/" + cluster_id_or_name + "/nodegroups/" + node_group_id_or_name + "/upgrade"

	req, err := http.NewRequest(http.MethodPost, host+url, bytes.NewBuffer([]byte(payload)))
	if err != nil {
		return "", err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("OpenStack-API-Version", "container-infra latest")
	req.Header.Set("X-Auth-Token", token)

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer res.Body.Close()

	bytes, _ := io.ReadAll(res.Body)

	return string(bytes), nil
}
