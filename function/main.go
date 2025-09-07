package main

import (
	"context"
	"fmt"
	"function/handlers"
	"net/http"
	"os"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
)

const (
	getBookMarksPath   = "GET /rb/bookmarks"
	createBookmarkPath = "POST /rb/bookmarks"
	deleteBookmarkPath = "DELETE /rb/bookmarks/{providerId}/{examId}/{questionId}"
)

var (
	apiHandlers map[string]handlers.ApiHandler
)

func init() {
	bookmarksTableName := os.Getenv("BOOKMARKS_TABLE_NAME")

	cfg, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		panic(err)
	}

	dbClient := dynamodb.NewFromConfig(cfg)

	abstractHandler := handlers.AbstractApiHandler{
		DbClient:           dbClient,
		BookmarksTableName: bookmarksTableName,
	}

	apiHandlers = map[string]handlers.ApiHandler{
		getBookMarksPath:   &handlers.GetBookmarksHandler{AbstractApiHandler: abstractHandler},
		createBookmarkPath: &handlers.CreateBookmarkHandler{AbstractApiHandler: abstractHandler},
		deleteBookmarkPath: &handlers.DeleteBookmarkHandler{AbstractApiHandler: abstractHandler},
	}
}

func handler(request events.APIGatewayV2HTTPRequest) (events.APIGatewayV2HTTPResponse, error) {
	path := request.RouteKey

	if apiHandler, ok := apiHandlers[path]; ok {
		return apiHandler.Invoke(request)
	}

	return events.APIGatewayV2HTTPResponse{
		Body:       fmt.Sprintf("Path Not Found: %s", path),
		StatusCode: http.StatusNotFound,
	}, nil
}

func main() {
	lambda.Start(handler)
}
