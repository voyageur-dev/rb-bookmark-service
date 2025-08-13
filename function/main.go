package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/expression"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

const (
	getBookMarksPath   = "GET /rb/bookmarks"
	createBookmarkPath = "POST /rb/bookmarks"
	deleteBookmarkPath = "DELETE /rb/bookmarks/{examId}/{questionId}"
)

var (
	bookmarksTableName string

	dbClient *dynamodb.Client
)

func init() {
	bookmarksTableName = os.Getenv("BOOKMARKS_TABLE_NAME")

	cfg, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		panic(err)
	}

	dbClient = dynamodb.NewFromConfig(cfg)
}

func handler(request events.APIGatewayV2HTTPRequest) (events.APIGatewayV2HTTPResponse, error) {
	path := request.RouteKey

	return func() (events.APIGatewayV2HTTPResponse, error) {
		switch path {
		case getBookMarksPath:
			return getBookmarks(request)
		case createBookmarkPath:
			return createBookmark(request)
		case deleteBookmarkPath:
			return deleteBookmark(request)
		default:
			return events.APIGatewayV2HTTPResponse{
				Body:       "Path Not Found",
				StatusCode: http.StatusNotFound,
			}, nil
		}
	}()
}

func getBookmarks(request events.APIGatewayV2HTTPRequest) (events.APIGatewayV2HTTPResponse, error) {
	queryParams := request.QueryStringParameters
	providerId, providerIdExt := queryParams["providerId"]
	examId, examIdExt := queryParams["examId"]

	userId, _ := request.RequestContext.Authorizer.JWT.Claims["sub"]

	builder := expression.Key("userId").Equal(expression.Value(userId))
	if providerIdExt && examIdExt {
		builder = builder.And(expression.Key("providerExamQuestionKey").BeginsWith(fmt.Sprintf("%s#%s", providerId, examId)))
	}
	expr, _ := expression.NewBuilder().WithKeyCondition(builder).Build()

	input := &dynamodb.QueryInput{
		TableName:                 aws.String(bookmarksTableName),
		KeyConditionExpression:    expr.KeyCondition(),
		ExpressionAttributeNames:  expr.Names(),
		ExpressionAttributeValues: expr.Values(),
	}

	result, err := dbClient.Query(context.TODO(), input)
	if err != nil {
		log.Println(fmt.Sprintf("Error getting bookmarks for %s, %v", userId, err))
		return events.APIGatewayV2HTTPResponse{
			Body:       "Error getting bookmarks",
			StatusCode: http.StatusInternalServerError,
		}, nil
	}

	bookmarks := make(map[string]map[string][]string)
	for _, item := range result.Items {
		parts := strings.Split(item["providerExamQuestionKey"].(*types.AttributeValueMemberS).Value, "#")

		if _, ok := bookmarks[parts[0]]; !ok {
			bookmarks[parts[0]] = make(map[string][]string)
		}
		bookmarks[parts[0]][parts[1]] = append(bookmarks[parts[0]][parts[1]], parts[2])
	}

	response, _ := json.Marshal(bookmarks)

	return events.APIGatewayV2HTTPResponse{
		Body:       string(response),
		StatusCode: http.StatusOK,
	}, nil
}

func createBookmark(request events.APIGatewayV2HTTPRequest) (events.APIGatewayV2HTTPResponse, error) {
	userId, _ := request.RequestContext.Authorizer.JWT.Claims["sub"]

	body := make(map[string]interface{})
	err := json.Unmarshal([]byte(request.Body), &body)
	if err != nil {
		log.Println(fmt.Sprintf("Error creating bookmark for %s, %v", userId, err))
		return events.APIGatewayV2HTTPResponse{
			Body:       "Error creating bookmark",
			StatusCode: http.StatusBadRequest,
		}, nil
	}

	providerId := body["providerId"].(string)
	examId := body["examId"].(string)
	questionId := body["questionId"].(string)

	item := map[string]types.AttributeValue{
		"userId": &types.AttributeValueMemberS{Value: userId},
		"providerExamQuestionKey": &types.AttributeValueMemberS{
			Value: fmt.Sprintf("%s#%s#%s", providerId, examId, questionId),
		},
		"createdAt": &types.AttributeValueMemberS{
			Value: time.Now().UTC().Format(time.RFC3339),
		},
	}

	_, err = dbClient.PutItem(context.TODO(), &dynamodb.PutItemInput{
		TableName: aws.String(bookmarksTableName),
		Item:      item,
	})

	if err != nil {
		log.Println(fmt.Sprintf("Error creating bookmark for %s, %v", userId, err))
		return events.APIGatewayV2HTTPResponse{
			Body:       "Error creating bookmark",
			StatusCode: http.StatusInternalServerError,
		}, nil
	}

	return events.APIGatewayV2HTTPResponse{
		StatusCode: http.StatusCreated,
	}, nil
}

func deleteBookmark(request events.APIGatewayV2HTTPRequest) (events.APIGatewayV2HTTPResponse, error) {
	userId, _ := request.RequestContext.Authorizer.JWT.Claims["sub"]

	providerId := request.PathParameters["providerId"]
	examId := request.PathParameters["examId"]
	questionId := request.PathParameters["questionId"]

	key := map[string]types.AttributeValue{
		"userId": &types.AttributeValueMemberS{Value: userId},
		"providerExamQuestionKey": &types.AttributeValueMemberS{
			Value: fmt.Sprintf("%s#%s#%s", providerId, examId, questionId),
		},
	}

	_, err := dbClient.DeleteItem(context.TODO(), &dynamodb.DeleteItemInput{
		TableName: aws.String(bookmarksTableName),
		Key:       key,
	})

	if err != nil {
		log.Println(fmt.Sprintf("Error deleting bookmark for %s, %v", userId, err))
		return events.APIGatewayV2HTTPResponse{
			Body:       "Error deleting bookmark",
			StatusCode: http.StatusInternalServerError,
		}, nil
	}

	return events.APIGatewayV2HTTPResponse{
		StatusCode: http.StatusOK,
	}, nil
}

func main() {
	lambda.Start(handler)
}
