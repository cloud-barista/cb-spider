package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/TylerBrock/colorjson"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

func executeRequest(path, method string, operation map[string]interface{}, cmd *cobra.Command) error {
	if parameters, ok := operation["parameters"].([]interface{}); ok {
		for _, param := range parameters {
			paramMap, _ := param.(map[string]interface{})
			paramName, _ := paramMap["name"].(string)
			paramIn, _ := paramMap["in"].(string)
			if paramIn == "path" {
				value, _ := cmd.Flags().GetString(paramName)
				if value != "" {
					path = strings.Replace(path, "{"+paramName+"}", value, -1)
				} else {
					return fmt.Errorf("path parameter '%s' is required", paramName)
				}
			}
		}
	}

	// Construct the full URL with "http://" prefix and "/spider" path
	fullURL := fmt.Sprintf("http://%s/spider%s", serverURL, path)

	queryParams := ""
	if parameters, ok := operation["parameters"].([]interface{}); ok {
		var queryParamList []string
		for _, param := range parameters {
			paramMap, _ := param.(map[string]interface{})
			paramName, _ := paramMap["name"].(string)
			paramIn, _ := paramMap["in"].(string)
			if paramIn == "query" {
				value, _ := cmd.Flags().GetString(paramName)
				if value != "" {
					queryParamList = append(queryParamList, fmt.Sprintf("%s=%s", paramName, value))
				}
			}
		}
		if len(queryParamList) > 0 {
			queryParams = "?" + strings.Join(queryParamList, "&")
		}
	}
	fullURL += queryParams

	var req *http.Request
	var err error

	switch strings.ToUpper(method) {
	case "POST", "PUT", "DELETE":
		if isFileUpload(operation) {
			req, err = createMultipartRequest(fullURL, cmd)
		} else {
			req, err = createJSONRequest(fullURL, method, cmd)
		}
	default:
		req, err = http.NewRequest(strings.ToUpper(method), fullURL, nil)
	}

	if err != nil {
		return fmt.Errorf("error creating HTTP request: %v", err)
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("error sending HTTP request: %v", err)
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("error reading response body: %v", err)
	}

	bodyStr := string(body)
	bodyStr = strings.Replace(bodyStr, "connectionName is empty!", "ConnectionName is empty!", -1)
	body = []byte(bodyStr)

	var obj map[string]interface{}
	if err := json.Unmarshal(body, &obj); err != nil {
		return fmt.Errorf("error parsing JSON response: %v", err)
	}

	formatter := colorjson.NewFormatter()
	formatter.Indent = 2
	colorizedJSON, err := formatter.Marshal(obj)
	if err != nil {
		return fmt.Errorf("error formatting JSON response: %v", err)
	}

	//fmt.Printf("Response Status: %s\n", resp.Status)
	//fmt.Println("Response Body:")
	fmt.Println(string(colorizedJSON))

	return nil
}

func createJSONRequest(fullURL, method string, cmd *cobra.Command) (*http.Request, error) {
	dataFlag, _ := cmd.Flags().GetString("data")
	var jsonBodyBytes []byte
	var err error

	if dataFlag != "" {
		jsonBodyBytes = []byte(dataFlag)
	} else {
		jsonBody := map[string]interface{}{}
		cmd.Flags().VisitAll(func(flag *pflag.Flag) {
			if flag.Changed && flag.Name != "data" {
				var value interface{}
				err := json.Unmarshal([]byte(flag.Value.String()), &value)
				if err != nil {
					jsonBody[flag.Name] = flag.Value.String()
				} else {
					if objMap, ok := value.(map[string]interface{}); ok {
						for k, v := range objMap {
							jsonBody[k] = v
						}
					} else {
						jsonBody[flag.Name] = value
					}
				}
			}
		})

		jsonBodyBytes, err = json.Marshal(jsonBody)
		if err != nil {
			return nil, fmt.Errorf("error marshaling JSON body: %v", err)
		}
	}

	req, err := http.NewRequest(strings.ToUpper(method), fullURL, bytes.NewBuffer(jsonBodyBytes))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	return req, nil
}

func isFileUpload(operation map[string]interface{}) bool {
	if consumes, ok := operation["consumes"].([]interface{}); ok {
		for _, consume := range consumes {
			if consume == "multipart/form-data" {
				return true
			}
		}
	}
	return false
}

func createMultipartRequest(fullURL string, cmd *cobra.Command) (*http.Request, error) {
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	var fileError error
	cmd.Flags().VisitAll(func(flag *pflag.Flag) {
		if flag.Changed {
			if strings.HasPrefix(flag.Name, "file:") {
				filePath := flag.Value.String()
				file, err := os.Open(filePath)
				if err != nil {
					fileError = err
					return
				}
				defer file.Close()

				part, err := writer.CreateFormFile(strings.TrimPrefix(flag.Name, "file:"), filepath.Base(filePath))
				if err != nil {
					fileError = err
					return
				}
				if _, err = io.Copy(part, file); err != nil {
					fileError = err
					return
				}
			} else {
				writer.WriteField(flag.Name, flag.Value.String())
			}
		}
	})

	if fileError != nil {
		return nil, fileError
	}

	if err := writer.Close(); err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", fullURL, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())
	return req, nil
}
