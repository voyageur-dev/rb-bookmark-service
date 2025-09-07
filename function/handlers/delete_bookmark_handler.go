package handlers

import (
	"context"
	"fmt"
	"log"
	"net/http"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

type DeleteBookmarkHandler struct {
	AbstractApiHandler
}

func (h *DeleteBookmarkHandler) Invoke(request events.APIGatewayV2HTTPRequest) (events.APIGatewayV2HTTPResponse, error) {
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

	_, err := h.DbClient.DeleteItem(context.TODO(), &dynamodb.DeleteItemInput{
		TableName: aws.String(h.BookmarksTableName),
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
