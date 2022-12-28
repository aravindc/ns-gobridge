package common

import (
	"context"
	"encoding/json"
	"os"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
	log "github.com/sirupsen/logrus"
)

func SetEnvWithAwsSSM(parameterName string, region string) {

	config, err := config.LoadDefaultConfig(context.TODO(), config.WithRegion(region))
	if err != nil {
		log.Fatal(err)
	}

	// Create SSM client
	svc := ssm.NewFromConfig(config)
	input := &ssm.GetParameterInput{
		Name:           aws.String(parameterName),
		WithDecryption: aws.Bool(true),
	}

	result, err := svc.GetParameter(context.TODO(), input)
	if err != nil {
		log.Fatal(err)
	}

	var secrets map[string]string
	err = json.Unmarshal([]byte(*result.Parameter.Value), &secrets)
	if err != nil {
		log.Fatal(err)
	}

	for key, value := range secrets {
		os.Setenv(key, value)
	}
}
