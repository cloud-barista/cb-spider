package resources

import (
	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"

	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"

	"github.com/aws/aws-sdk-go/service/pricing"
)

type AwsPriceInfoHandler struct {
	Region idrv.RegionInfo
	Client *pricing.Pricing
}

func (priceInfoHandler *AwsPriceInfoHandler) ListProductFamily(targetRegion string) ([]string, error) {
	mySession := session.Must(session.NewSession())
	// Create a Pricing client with additional configuration
	svc := pricing.New(mySession, aws.NewConfig().WithRegion("us-west-2"))
	input := &pricing.DescribeServicesInput{
		FormatVersion: aws.String("aws_v1"),
		MaxResults:    aws.Int64(1),
		ServiceCode:   aws.String("AmazonEC2"),
	}

	// result, err := priceInfoHandler.Client.DescribeServices(input)
	result, err := svc.DescribeServices(input)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			case pricing.ErrCodeInternalErrorException:
				fmt.Println(pricing.ErrCodeInternalErrorException, aerr.Error())
			case pricing.ErrCodeInvalidParameterException:
				fmt.Println(pricing.ErrCodeInvalidParameterException, aerr.Error())
			case pricing.ErrCodeNotFoundException:
				fmt.Println(pricing.ErrCodeNotFoundException, aerr.Error())
			case pricing.ErrCodeInvalidNextTokenException:
				fmt.Println(pricing.ErrCodeInvalidNextTokenException, aerr.Error())
			case pricing.ErrCodeExpiredNextTokenException:
				fmt.Println(pricing.ErrCodeExpiredNextTokenException, aerr.Error())
			default:
				fmt.Println(aerr.Error())
			}
		} else {
			// Print the error, cast err to awserr.Error to get the Code and
			// Message from an error.
			fmt.Println(err.Error())
		}
	}

	fmt.Println(result)
	return nil, nil
}

func (priceInfoHandler *AwsPriceInfoHandler) GetPriceInfo(productFamily string, regionName string, filterList []irs.KeyValue) (string, error) {
	return "", nil
}
