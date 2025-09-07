package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/expression"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

type GetBookmarksHandler struct {
	AbstractApiHandler
}

func (h *GetBookmarksHandler) Invoke(request events.APIGatewayV2HTTPRequest) (events.APIGatewayV2HTTPResponse, error) {
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
		TableName:                 aws.String(h.BookmarksTableName),
		KeyConditionExpression:    expr.KeyCondition(),
		ExpressionAttributeNames:  expr.Names(),
		ExpressionAttributeValues: expr.Values(),
	}

	result, err := h.DbClient.Query(context.TODO(), input)
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
