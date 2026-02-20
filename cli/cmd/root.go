package cmd

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/TylerBrock/colorjson"
	"github.com/spf13/cobra"
)

//go:embed swagger.json
var swaggerJSON []byte

var (
	Version   string // Populated by ldflags
	CommitSHA string // Populated by ldflags
	BuildTime string // Populated by ldflags
)

var rootCmd = &cobra.Command{
	Use:   "spctl",
	Short: "Spider CLI for managing multi-cloud infrastructure",
	Run: func(cmd *cobra.Command, args []string) {
		if versionFlag, _ := cmd.Flags().GetBool("version"); versionFlag {
			fmt.Printf("Version:    %s\n", Version)
			fmt.Printf("Commit SHA: %s\n", CommitSHA)
			fmt.Printf("Build Time: %s\n", BuildTime)
			return
		}
		cmd.Help()
	},
}

var serverURL string
var apiUsername string
var apiPassword string

func Execute() {
	rootCmd.PersistentFlags().StringVarP(&serverURL, "server", "s", "localhost:1024", "Spider server URL")
	rootCmd.PersistentFlags().StringVarP(&apiUsername, "username", "u", "", "API username (default: $API_USERNAME)")
	rootCmd.PersistentFlags().StringVarP(&apiPassword, "password", "p", "", "API password (default: $API_PASSWORD)")
	rootCmd.Flags().BoolP("version", "v", false, "Print the version information")

	loadSwagger()
	cobra.CheckErr(rootCmd.Execute())
}

// getCredentials returns username and password from flags or environment variables.
func getCredentials() (string, string) {
	user := apiUsername
	pass := apiPassword
	if user == "" {
		user = os.Getenv("API_USERNAME")
	}
	if pass == "" {
		pass = os.Getenv("API_PASSWORD")
	}
	return user, pass
}

var swaggerDefinitions map[string]interface{}

func loadSwagger() {
	var swagger map[string]interface{}
	if err := json.Unmarshal(swaggerJSON, &swagger); err != nil {
		fmt.Println("Error decoding embedded Swagger JSON:", err)
		return
	}

	if definitions, ok := swagger["definitions"].(map[string]interface{}); ok {
		swaggerDefinitions = definitions
	} else {
		fmt.Println("No definitions found in Swagger JSON")
	}

	if paths, ok := swagger["paths"].(map[string]interface{}); ok {
		for path, pathItem := range paths {
			if pathItemMap, ok := pathItem.(map[string]interface{}); ok {
				for method, operation := range pathItemMap {
					addCommand(path, method, operation)
				}
			}
		}
	}
}

func addCommand(path, method string, operation interface{}) {
	operationMap, ok := operation.(map[string]interface{})
	if !ok {
		return
	}

	operationID, _ := operationMap["operationId"].(string)
	if operationID == "" {
		return
	}

	// Get the description from the operationMap
	description, _ := operationMap["description"].(string)
	if description == "" {
		description = fmt.Sprintf("%s %s", method, path) // Fallback if no description is provided
	}

	parts := strings.SplitN(operationID, "-", 2)
	if len(parts) != 2 {
		return
	}

	resource := parts[1]
	command := parts[0]

	mainCmd := findOrCreateMainCmd(resource)

	cmd := &cobra.Command{
		Use:   command,
		Short: description, // Include the description
		RunE: func(cmd *cobra.Command, args []string) error {
			dataFlag, _ := cmd.Flags().GetString("data")
			if dataFlag == "" {
				if err := checkRequiredFlags(cmd, operationMap); err != nil {
					fullExampleJSON, requiredExampleJSON, jsonErr := generateExampleJSON(operationMap)
					if jsonErr != nil {
						if jsonErr.Error() == "no body parameter found" {
							cmd.Usage()
							return nil
						}
						return fmt.Errorf("error generating example JSON: %v", jsonErr)
					}

					cmd.Usage()

					fmt.Println()
					examplesIndent := "    "

					fmt.Println("Examples:")
					indent := examplesIndent

					printIndentedJSON := func(title, jsonStr string) {
						fmt.Println(indent + title)
						var obj map[string]interface{}
						if err := json.Unmarshal([]byte(jsonStr), &obj); err == nil {
							formatter := colorjson.NewFormatter()
							formatter.Indent = 2
							colorizedJSON, _ := formatter.Marshal(obj)
							jsonLines := strings.Split(string(colorizedJSON), "\n")
							if len(jsonLines) > 0 {
								fmt.Println(indent + "'" + jsonLines[0])
								for i := 1; i < len(jsonLines)-1; i++ {
									fmt.Println(indent + "  " + jsonLines[i])
								}
								if len(jsonLines) > 1 {
									fmt.Println(indent + "  " + jsonLines[len(jsonLines)-1] + "'")
								} else {
									fmt.Println(indent + "'")
								}
							} else {
								fmt.Println(indent + "''")
							}
							fmt.Println()
						} else {
							fmt.Println(indent + "'" + jsonStr + "'")
						}
					}

					exampleTitleAllFields := "Example JSON"
					exampleTitleRequiredFields := "Example JSON"

					var bodyParamName string
					if parameters, ok := operationMap["parameters"].([]interface{}); ok {
						for _, param := range parameters {
							paramMap, _ := param.(map[string]interface{})
							paramName, _ := paramMap["name"].(string)
							paramIn, _ := paramMap["in"].(string)
							if paramIn == "body" || paramIn == "formData" {
								bodyParamName = paramName
								break
							}
						}
					}

					if bodyParamName != "" {
						exampleTitleAllFields += fmt.Sprintf(" for %s with all fields:", bodyParamName)
						exampleTitleRequiredFields += fmt.Sprintf(" for %s with required fields only:", bodyParamName)
					} else {
						exampleTitleAllFields += " with all fields:"
						exampleTitleRequiredFields += " with required fields only:"
					}

					printIndentedJSON(exampleTitleAllFields, fullExampleJSON)
					printIndentedJSON(exampleTitleRequiredFields, requiredExampleJSON)

					return nil
				}
			}
			return executeRequest(path, method, operationMap, cmd)
		},
	}

	addFlags(cmd, operationMap)

	mainCmd.AddCommand(cmd)
}

func checkRequiredFlags(cmd *cobra.Command, operationMap map[string]interface{}) error {
	dataFlag, _ := cmd.Flags().GetString("data")
	if dataFlag != "" {
		return nil
	}

	if parameters, ok := operationMap["parameters"].([]interface{}); ok {
		for _, param := range parameters {
			paramMap, _ := param.(map[string]interface{})
			paramName, _ := paramMap["name"].(string)
			paramIn, _ := paramMap["in"].(string)
			required, _ := paramMap["required"].(bool)
			if required {
				if paramIn == "body" || paramIn == "formData" || paramIn == "query" || paramIn == "path" {
					flag := cmd.Flags().Lookup(paramName)
					if flag == nil || !flag.Changed {
						return fmt.Errorf("required flag '--%s' not set", paramName)
					} else if flag.Value.String() == "" {
						return fmt.Errorf("required flag '--%s' is set but empty", paramName)
					}
				}
			}
		}
	}
	return nil
}

func generateExampleJSON(operationMap map[string]interface{}) (string, string, error) {
	if parameters, ok := operationMap["parameters"].([]interface{}); ok {
		for _, param := range parameters {
			paramMap, _ := param.(map[string]interface{})
			paramIn, _ := paramMap["in"].(string)
			if paramIn == "body" {
				schema, _ := paramMap["schema"].(map[string]interface{})
				fullExampleObj, err := buildExampleFromSchema(schema, true)
				if err != nil {
					return "", "", err
				}
				fullExampleJSONBytes, err := json.MarshalIndent(fullExampleObj, "", "  ")
				if err != nil {
					return "", "", err
				}
				requiredExampleObj, err := buildExampleFromSchema(schema, false)
				if err != nil {
					return "", "", err
				}
				requiredExampleJSONBytes, err := json.MarshalIndent(requiredExampleObj, "", "  ")
				if err != nil {
					return "", "", err
				}
				return string(fullExampleJSONBytes), string(requiredExampleJSONBytes), nil
			}
		}
	}
	return "", "", fmt.Errorf("no body parameter found")
}

func buildExampleFromSchema(schema map[string]interface{}, includeOptional bool) (interface{}, error) {
	if ref, ok := schema["$ref"].(string); ok {
		definitionName := strings.TrimPrefix(ref, "#/definitions/")
		definition, ok := swaggerDefinitions[definitionName]
		if !ok {
			return nil, fmt.Errorf("definition '%s' not found", definitionName)
		}

		definitionMap, ok := definition.(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("definition '%s' is not a valid object", definitionName)
		}

		return buildExampleFromSchema(definitionMap, includeOptional)
	}

	if schemaType, ok := schema["type"].(string); ok {
		switch schemaType {
		case "object":
			exampleObj := make(map[string]interface{})
			requiredFields := make(map[string]bool)
			if requiredList, ok := schema["required"].([]interface{}); ok {
				for _, fieldName := range requiredList {
					if fieldStr, ok := fieldName.(string); ok {
						requiredFields[fieldStr] = true
					}
				}
			}
			if properties, ok := schema["properties"].(map[string]interface{}); ok {
				for propName, propSchema := range properties {
					if !includeOptional && len(requiredFields) > 0 && !requiredFields[propName] {
						continue
					}
					propSchemaMap, _ := propSchema.(map[string]interface{})
					exampleValue, err := buildExampleFromSchema(propSchemaMap, includeOptional)
					if err != nil {
						return nil, err
					}
					exampleObj[propName] = exampleValue
				}
			}
			return exampleObj, nil
		case "array":
			items, _ := schema["items"].(map[string]interface{})
			exampleValue, err := buildExampleFromSchema(items, includeOptional)
			if err != nil {
				return nil, err
			}
			return []interface{}{exampleValue}, nil
		case "string":
			if example, ok := schema["example"]; ok {
				return example, nil
			}
			return "string", nil
		case "integer", "number":
			if example, ok := schema["example"]; ok {
				return example, nil
			}
			return 0, nil
		case "boolean":
			if example, ok := schema["example"]; ok {
				return example, nil
			}
			return false, nil
		}
	}
	return nil, fmt.Errorf("unsupported schema type")
}

func addFlags(cmd *cobra.Command, operationMap map[string]interface{}) {
	var bodyParamName string
	var hasBodyParam bool

	if parameters, ok := operationMap["parameters"].([]interface{}); ok {
		for _, param := range parameters {
			paramMap, _ := param.(map[string]interface{})
			paramName, _ := paramMap["name"].(string)
			paramType, _ := paramMap["in"].(string)
			if paramName != "" {
				switch paramType {
				case "query":
					if paramName == "ConnectionName" {
						// Add short flag '-c' for --ConnectionName
						cmd.Flags().StringP(paramName, "c", "", fmt.Sprintf("Query parameter: %s", paramName))
					} else {
						cmd.Flags().String(paramName, "", fmt.Sprintf("Query parameter: %s", paramName))
					}
				case "body", "formData":
					bodyParamName = paramName
					hasBodyParam = true
				case "path":
					if paramName == "ConnectionName" {
						// Add short flag '-c' for --ConnectionName
						cmd.Flags().StringP(paramName, "c", "", fmt.Sprintf("Path parameter: %s", paramName))
					} else if paramName == "Name" {
						// Add short flag '-n' for --Name
						cmd.Flags().StringP(paramName, "n", "", fmt.Sprintf("Path parameter: %s", paramName))
					} else {
						cmd.Flags().String(paramName, "", fmt.Sprintf("Path parameter: %s", paramName))
					}
				}
			}
		}
	}

	// Add the --data flag only if there is a body parameter
	if hasBodyParam {
		if bodyParamName != "" {
			cmd.Flags().StringP("data", "d", "", fmt.Sprintf("JSON Body/Form parameter: %s", bodyParamName))
		} else {
			cmd.Flags().StringP("data", "d", "", "HTTP request body data (in JSON format)")
		}
	}
}

func findOrCreateMainCmd(resource string) *cobra.Command {
	for _, cmd := range rootCmd.Commands() {
		if cmd.Use == resource || cmd.HasAlias(resource) {
			return cmd
		}
	}

	newMainCmd := &cobra.Command{
		Use:   resource,
		Short: fmt.Sprintf("Commands for %s", resource),
	}

	// Change the display name for "connection-config" to "connection"
	if resource == "connection-config" {
		newMainCmd.Use = "connection"
		newMainCmd.Aliases = []string{"connection-config"}
	}

	// Change the display name for "vm-spec" to "vmspec"
	if resource == "vm-spec" {
		newMainCmd.Use = "vmspec"
		newMainCmd.Aliases = []string{"vm-spec"}
	}

	rootCmd.AddCommand(newMainCmd)
	return newMainCmd
}
