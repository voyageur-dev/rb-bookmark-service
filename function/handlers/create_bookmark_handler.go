package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

type CreateBookmarkHandler struct {
	AbstractApiHandler
}

func (h *CreateBookmarkHandler) Invoke(request events.APIGatewayV2HTTPRequest) (events.APIGatewayV2HTTPResponse, error) {
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

	_, err = h.DbClient.PutItem(context.TODO(), &dynamodb.PutItemInput{
		TableName: aws.String(h.BookmarksTableName),
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
