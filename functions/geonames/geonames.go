package main

import (
	"net/http"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/feedmytrip/api/resources"
)

func router(req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	geoname := resources.Geoname{}
	switch req.Resource {
	case "/geonames":
		switch req.HTTPMethod {
		case "POST":
			return geoname.SaveNew(req)
		}
	case "/geonames/{id}":
		switch req.HTTPMethod {
		case "DELETE":
			return geoname.Delete(req)
		case "PATCH":
			return geoname.Update(req)
		}
	case "/geonames/countries":
		switch req.HTTPMethod {
		case "GET":
			return geoname.GetAll(req)
		}
	case "/geonames/countries/{id}":
		switch req.HTTPMethod {
		case "GET":
			return geoname.GetAll(req)
		}
	}

	return events.APIGatewayProxyResponse{
		StatusCode: http.StatusMethodNotAllowed,
	}, nil
}

func main() {
	lambda.Start(router)
}