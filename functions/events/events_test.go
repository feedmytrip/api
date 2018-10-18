package main

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/feedmytrip/api/common"
	"github.com/feedmytrip/api/db"
	"github.com/feedmytrip/api/resources"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

const (
	TableName = "EventsTest"
)

type FeedMyTripAPITestSuite struct {
	suite.Suite
	db      *dynamodb.DynamoDB
	eventID string
}

func (suite *FeedMyTripAPITestSuite) SetupTest() {
	common.EventsTable = TableName
	db.CreateTable(TableName, "eventId", 1, 1)
}

func (suite *FeedMyTripAPITestSuite) Test0010SaveNewEvent() {
	req := events.APIGatewayProxyRequest{
		Body: `{
			"translations": [
				{
					"code": "pt",
					"title": "FMT - Testing suite #1",
					"description": "Loren ipsum ea est atqui iisque placerat, est nobis videre."
				}
			]
		}`,
	}

	event := resources.Event{}
	response, err := event.SaveNew(req)
	json.Unmarshal([]byte(response.Body), &event)
	suite.eventID = event.EventID

	assert.Nil(suite.T(), err)
	assert.Equal(suite.T(), http.StatusCreated, response.StatusCode)
}

func (suite *FeedMyTripAPITestSuite) Test0020GetAllEvents() {
	req := events.APIGatewayProxyRequest{}

	event := resources.Event{}
	response, err := event.GetAll(req)

	assert.Nil(suite.T(), err)
	assert.Equal(suite.T(), http.StatusOK, response.StatusCode)
}

func (suite *FeedMyTripAPITestSuite) Test0030UpdateEvent() {
	req := events.APIGatewayProxyRequest{
		PathParameters: map[string]string{
			"id": suite.eventID,
		},
		Body: `{
			"active": false
		}`,
	}

	event := resources.Event{}
	response, err := event.Update(req)

	assert.Nil(suite.T(), err)
	assert.Equal(suite.T(), http.StatusOK, response.StatusCode)
}

func (suite *FeedMyTripAPITestSuite) Test0040UpdateEventTranslation() {
	req := events.APIGatewayProxyRequest{
		PathParameters: map[string]string{
			"id": suite.eventID,
		},
		Body: `{
			"code": "en",
			"title": "New title in english test #001",
			"description": "New description in english"
		}`,
	}

	et := resources.EventTranslation{}
	response, err := et.Save(req)

	assert.Nil(suite.T(), err)
	assert.Equal(suite.T(), http.StatusOK, response.StatusCode)
}

func (suite *FeedMyTripAPITestSuite) Test1000DeleteEvent() {
	req := events.APIGatewayProxyRequest{
		PathParameters: map[string]string{
			"id": suite.eventID,
		},
	}

	event := resources.Event{}
	response, err := event.Delete(req)

	assert.Nil(suite.T(), err)
	assert.Equal(suite.T(), http.StatusOK, response.StatusCode)
}

func TestFeedMyTripAPITestSuite(t *testing.T) {
	suite.Run(t, new(FeedMyTripAPITestSuite))
}