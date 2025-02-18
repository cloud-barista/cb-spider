// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// By CB-Spider Team

package adminweb

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html/template"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"

	"github.com/labstack/echo/v4"
)

// CmdAgent: Render the cmd-agent.html template
func CmdAgent(c echo.Context) error {
	tmplPath := filepath.Join(os.Getenv("CBSPIDER_ROOT"), "/api-runtime/rest-runtime/admin-web/html/cmd-agent.html")
	tmpl, err := template.ParseFiles(tmplPath)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return tmpl.Execute(c.Response().Writer, nil)
}

// generates CB-Spider Curl commands based on user queries.
func GenerateCmd(c echo.Context) error {
	var userInput struct {
		Query string `json:"query"`
	}

	if err := c.Bind(&userInput); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid request")
	}

	apiKey, err := getClaudeAPIKey()
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to get API key")
	}

	messages := []ClaudeMessage{
		{
			Role: "user",
			Content: []ClaudeContent{
				{Type: "text", Text: userInput.Query},
			},
		},
	}

	requestBody := ClaudeRequestBody{
		Model:       "claude-3-5-sonnet-20241022",
		MaxTokens:   8192,
		Temperature: 0.2, // 0.0-0.3: consistent & factual, 0.4-0.7: balanced, 0.8-1.0: creative & varied
		System: `너는 최신 CB-Spider Rest API를 이용해서 curl 명령을 만들어 주는 Agent임.\n
					명령을 만들 때, CB-Spider API 문법은 다음 link를 참고해서 만들어줘.\n
					https://cloud-barista.github.io/api/?url=https://raw.githubusercontent.com/cloud-barista/cb-spider/master/api/swagger.yaml
					API 생성할때, force 옵션을 사용하지 말아줘. ConnectionName(또는 ConfigName, connection_name은 아님) 설정을 받드시 추가해줘.\n
					사용자가 명시적으로 요청하지 않으면, driver, credential, region, connection 관련 명령은 제공하지 말아줘.\n
					VPC를 만들때는, 반드시 Subnet을 함께 만들도록 명령을 만들어줘.\n
					VM 인프라 자원 생성 시 운선 순위는 VPC/Subnet -> SecurityGroup -> Key -> VM 순서임.\n
					VM 인프라 자원 삭제 시 운선 순위는 VM -> Key -> SecurityGroup -> VPC/Subnet 순서임.\n
					VM을 만들때는, ImageName은 다음 맵핑 관계에서 각 CSP별 값으로 설정해줘.\n
					  - AWS: ami-00978328f54e31526\n
					  - Azure: Canonical:UbuntuServer:18.04-LTS:18.04.202106220\n
					  - GCP: https://www.googleapis.com/compute/v1/projects/ubuntu-os-cloud/global/images/ubuntu-2404-noble-amd64-v20240423\n
					  - Alibaba: ubuntu_22_04_x64_20G_alibase_20250113.vhd\n
					  - Tencent: img-pi0ii46r\n
					  - IBM: r014-1696a049-e959-493d-9a97-1655ef4c942e\n
					  - OpenStack: 16681742-f408-444d-a430-dd21a4bef42c\n
					  - ncpvpc: 16187005\n					 
					VM을 만들때는, VMSpecName은 다음 맵핑 관계에서 각 CSP별 값으로 설정해줘.\n
					  - AWS: t2.micro\n
					  - Azure: Standard_B1ls\n
					  - GCP: e2-standard-2\n
					  - Alibaba: ecs.c7.large\n
					  - Tencent: S5.MEDIUM8\n
					  - IBM: bx2-2x8\n
					  - OpenStack: ETRI-small-2\n
					  - ncpvpc: c4-g2-s50\n	
					사용자가 CSP 이름을 알려주지 않으면 aws를 이용해서 작성해줘.\n
					사용자가 명시적으로 connection 이름을 알려주지 않으면, 
					{CSP-Name}-config01을 이용해서 작성해줘.\n
					생성하는 자원 이름은 '자원 타입-01' 형태로 만들어 줘.\n
					자원 생성시에 다른 자원 이름을 지정할 때도 '자원 타입-01' 형태로 지정해서 만들어 줘.\n
					자원 삭제시에는 ConnectionName은 path가 아닌 JSONBody에 포함해서 설정해줘.\n 
					다른 설명은 하지 말고, 간단히 자원 생성하는 curl 명령만 출력해줘.\n
					curl 명령은 -sX 옵션으로 제공해줘.\n
					curl 명령에 routing path에 security가 포함되어 있으면, securitygruop으로 변경해서 출력해줘.\n					
					curl 명령에 connection_name가 포함되어 있으면, ConnectionName으로 변경해서 출력해줘.\n
					curl 명령에 포함된 json 출력은 보기 좋게 json 들여쓰기를 해줘.\n
					명령 뒤에 json_pp를 추가 시켜서 만들어줘.\n
					`,
		Messages: messages,
	}

	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to marshal JSON")
	}

	req, err := http.NewRequest("POST", "https://api.anthropic.com/v1/messages", bytes.NewBuffer(jsonData))
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to create API request")
	}

	req.Header.Set("x-api-key", apiKey)
	req.Header.Set("anthropic-version", "2023-06-01")
	req.Header.Set("content-type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to call API")
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to read response")
	}

	type AnthropicResponse struct {
		Content []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"content"`
	}

	var anthropicResp AnthropicResponse
	if err := json.Unmarshal(body, &anthropicResp); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError,
			fmt.Sprintf("Error parsing response: %v", err))
	}

	var commandsOnly string
	for _, cData := range anthropicResp.Content {
		commandsOnly += cData.Text + "\n"
	}

	return c.String(http.StatusOK, commandsOnly)
}
