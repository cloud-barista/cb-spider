/*
You must grant the following policies to IAM users
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Action": "sts:GetSessionToken",
      "Resource": "*"
    }
  ]
}
*/

package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sts"
)

func main() {
	clientID := os.Getenv("CLIENT_ID")
	clientSecret := os.Getenv("CLIENT_SECRET")

	if clientID == "" || clientSecret == "" {
		log.Fatal("Environment variables CLIENT_ID or CLIENT_SECRET are not set. Please set them first.")
	}

	region := promptRegionWithDefault("ap-northeast-2")

	// Create a new AWS session with static credentials
	sess, err := session.NewSession(&aws.Config{
		Region:      aws.String(region),
		Credentials: credentials.NewStaticCredentials(clientID, clientSecret, ""),
	})
	if err != nil {
		log.Fatalf("Failed to create AWS session: %v", err)
	}

	// Create STS client
	stsSvc := sts.New(sess)

	// Request session token (valid for 1 hour)
	input := &sts.GetSessionTokenInput{
		DurationSeconds: aws.Int64(3600),
	}
	result, err := stsSvc.GetSessionToken(input)
	if err != nil {
		log.Fatalf("Failed to get session token from STS: %v", err)
	}

	creds := result.Credentials
	fmt.Println("STS session token acquired successfully!")
	fmt.Printf("AccessKeyId: %s\n", *creds.AccessKeyId)
	fmt.Printf("SecretAccessKey: %s\n", *creds.SecretAccessKey)
	fmt.Printf("SessionToken: %s\n", *creds.SessionToken)
	fmt.Printf("Expiration: %s\n", creds.Expiration.Format(time.RFC3339))

	// Print reusable credential code snippet
	fmt.Println("\nUse the following in your client code:")
	fmt.Printf(`credentials.NewStaticCredentials(
	"%s",
	"%s",
	"%s",
)
`, *creds.AccessKeyId, *creds.SecretAccessKey, *creds.SessionToken)
}

// Prompt user to input region, fallback to default if blank
func promptRegionWithDefault(defaultRegion string) string {
	fmt.Printf("Enter AWS region (default: %s): ", defaultRegion)
	reader := bufio.NewReader(os.Stdin)
	input, _ := reader.ReadString('\n')
	region := strings.TrimSpace(input)
	if region == "" {
		return defaultRegion
	}
	return region
}
