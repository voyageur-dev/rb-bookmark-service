package main

import (
	"context"
	"encoding/json"
	"fmt"
	"function/models"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/expression"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"log"
	"os"
	"strconv"
	"strings"
)

const (
	getBookMarksPath = "GET /rb/bookmarks"
)

var bookmarksTableName string

var dbClient *dynamodb.Client

func init() {
	bookmarksTableName = os.Getenv("BOOKMARKS_TABLE_NAME")

	cfg, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		panic(err)
	}

	dbClient = dynamodb.NewFromConfig(cfg)
}

func handler(ctx context.Context, request events.APIGatewayV2HTTPRequest) (events.APIGatewayV2HTTPResponse, error) {
	path := request.RouteKey

	return func() (events.APIGatewayV2HTTPResponse, error) {
		switch path {
		case getBookMarksPath:
			return getBookmarks(ctx, request)
		default:
			return events.APIGatewayV2HTTPResponse{
				Body:       "Path Not Found",
				StatusCode: 404,
			}, nil
		}
	}()
}

func getBookmarks(ctx context.Context, request events.APIGatewayV2HTTPRequest) (events.APIGatewayV2HTTPResponse, error) {
	queryParams := request.QueryStringParameters
	examId, ext := queryParams["examId"]

	userId, _ := request.RequestContext.Authorizer.JWT.Claims["sub"]

	builder := expression.Key("user_id").Equal(expression.Value(userId))
	if ext {
		builder = builder.And(expression.Key("exam_question_key").BeginsWith(examId))
	}
	expr, _ := expression.NewBuilder().WithKeyCondition(builder).Build()

	input := &dynamodb.QueryInput{
		TableName:                 aws.String(bookmarksTableName),
		KeyConditionExpression:    expr.KeyCondition(),
		ExpressionAttributeNames:  expr.Names(),
		ExpressionAttributeValues: expr.Values(),
	}

	result, err := dbClient.Query(ctx, input)
	if err != nil {
		log.Println(fmt.Sprintf("Error getting bookmarks for %s, %v", userId, err))
		return events.APIGatewayV2HTTPResponse{
			Body:       "Error getting bookmarks",
			StatusCode: 500,
		}, nil
	}

	bookmarks := make(map[string][]int)
	for _, item := range result.Items {
		parts := strings.Split(item["exam_question_key"].(*types.AttributeValueMemberS).Value, "#")
		idx, err := strconv.Atoi(parts[1])
		if err != nil {
			continue
		}

		bookmarks[parts[0]] = append(bookmarks[parts[0]], idx)
	}

	response, _ := json.Marshal(models.GetBookmarksResponse{
		Bookmarks: bookmarks,
	})

	return events.APIGatewayV2HTTPResponse{
		Body:       string(response),
		StatusCode: 200,
	}, nil
}

func main() {
	lambda.Start(handler)
}
