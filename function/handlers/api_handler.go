package handlers

import (
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
)

type ApiHandler interface {
	Invoke(request events.APIGatewayV2HTTPRequest) (events.APIGatewayV2HTTPResponse, error)
}

type AbstractApiHandler struct {
	DbClient           *dynamodb.Client
	BookmarksTableName string
}
