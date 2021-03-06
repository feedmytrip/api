package trips

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/feedmytrip/api/common"
	"github.com/feedmytrip/api/db"
	"github.com/feedmytrip/api/resources/shared"
	"github.com/gocraft/dbr"
	"github.com/google/uuid"
)

// Trip represents a user trip
type Trip struct {
	ID          string             `json:"id" db:"id"`
	ItineraryID string             `json:"itinerary_id" db:"itinerary_id"`
	Active      bool               `json:"active" db:"active"`
	Title       shared.Translation `json:"title" table:"translation" alias:"title" on:"title.parent_id = trip.id and title.field = 'title'" embedded:"true" persist:"true"`
	Description shared.Translation `json:"description" table:"translation" alias:"description" on:"description.parent_id = trip.id and description.field = 'description'" embedded:"true" persist:"true"`
	Scope       string             `json:"scope" db:"scope" lock:"true"`
	CountryID   string             `json:"country_id" db:"country_id"`
	Country     shared.Translation `json:"country" table:"translation" alias:"country" on:"country.parent_id = trip.country_id and country.field = 'title'" embedded:"true"`
	RegionID    string             `json:"region_id" db:"region_id"`
	Region      shared.Translation `json:"region" table:"translation" alias:"region" on:"region.parent_id = trip.region_id and region.field = 'title'" embedded:"true"`
	CityID      string             `json:"city_id" db:"city_id"`
	City        shared.Translation `json:"city" table:"translation" alias:"city" on:"city.parent_id = trip.city_id and city.field = 'title'" embedded:"true"`
	CreatedBy   string             `json:"created_by" db:"created_by" lock:"true"`
	CreatedDate time.Time          `json:"created_date" db:"created_date" lock:"true"`
	UpdatedBy   string             `json:"updated_by" db:"updated_by"`
	UpdatedDate time.Time          `json:"updated_date" db:"updated_date"`
	CreatedUser shared.User        `json:"created_user" table:"user" alias:"created_user" on:"created_user.id = trip.created_by" embedded:"true"`
	UpdatedUser shared.User        `json:"updated_user" table:"user" alias:"updated_user" on:"updated_user.id = trip.updated_by" embedded:"true"`
}

//Get return a trip
func (t *Trip) Get(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	conn, err := db.Connect()
	if err != nil {
		return common.APIError(http.StatusInternalServerError, err)
	}

	session := conn.NewSession(nil)
	defer session.Close()
	defer conn.Close()

	result, err := db.QueryOne(session, db.TableTrip, request.PathParameters["id"], Trip{})
	if err != nil {
		return common.APIError(http.StatusInternalServerError, err)
	}

	return common.APIResponse(result, http.StatusOK)
}

//GetAll returns all trips available in the database
func (t *Trip) GetAll(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	tokenUser := common.GetTokenUser(request)
	if !tokenUser.IsAdmin() {
		return common.APIError(http.StatusForbidden, errors.New("only admin users can access all trips"))
	}

	conn, err := db.Connect()
	defer conn.Close()
	if err != nil {
		return common.APIError(http.StatusInternalServerError, err)
	}

	session := conn.NewSession(nil)
	defer session.Close()

	result, err := db.Select(session, db.TableTrip, request.QueryStringParameters, Trip{})
	if err != nil {
		return common.APIError(http.StatusInternalServerError, err)
	}

	return common.APIResponse(result, http.StatusOK)
}

//SaveNew creates a new trip
func (t *Trip) SaveNew(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	tokenUser := common.GetTokenUser(request)

	err := json.Unmarshal([]byte(request.Body), t)
	if err != nil {
		return common.APIError(http.StatusBadRequest, err)
	}

	if t.Title.IsEmpty() {
		return common.APIError(http.StatusBadRequest, errors.New("invalid request empty title"))
	}

	t.ID = uuid.New().String()
	t.Active = true
	t.Title.ID = uuid.New().String()
	t.Title.Table = db.TableTrip
	t.Title.Field = "title"
	t.Title.ParentID = t.ID
	t.Description.ID = uuid.New().String()
	t.Description.Table = db.TableTrip
	t.Description.Field = "description"
	t.Description.ParentID = t.ID
	t.CreatedBy = tokenUser.UserID
	t.CreatedDate = time.Now()
	t.UpdatedBy = tokenUser.UserID
	t.UpdatedDate = time.Now()

	t.Scope = "user"
	if tokenUser.IsAdmin() {
		t.Scope = "global"
	}

	t.ItineraryID = uuid.New().String()
	defaultItinerary := Itinerary{}
	defaultItinerary.ID = t.ItineraryID
	defaultItinerary.TripID = t.ID
	defaultItinerary.OwnerID = tokenUser.UserID
	defaultItinerary.StartDate = time.Now()
	defaultItinerary.EndDate = time.Now()
	defaultItinerary.Title.ID = uuid.New().String()
	defaultItinerary.Title.ParentID = defaultItinerary.ID
	defaultItinerary.Title.Table = db.TableTripItinerary
	defaultItinerary.Title.Field = "title"
	defaultItinerary.Title.PT = "Padrão"
	defaultItinerary.Title.EN = "Default"
	defaultItinerary.Title.ES = "Estándar"
	defaultItinerary.CreatedBy = tokenUser.UserID
	defaultItinerary.CreatedDate = time.Now()
	defaultItinerary.UpdatedBy = tokenUser.UserID
	defaultItinerary.UpdatedDate = time.Now()

	ownerParticipant := Participant{}
	ownerParticipant.ID = uuid.New().String()
	ownerParticipant.TripID = t.ID
	ownerParticipant.UserID = tokenUser.UserID
	ownerParticipant.Role = ParticipantOwnerRole
	ownerParticipant.CreatedBy = tokenUser.UserID
	ownerParticipant.CreatedDate = time.Now()
	ownerParticipant.UpdatedBy = tokenUser.UserID
	ownerParticipant.UpdatedDate = time.Now()

	conn, err := db.Connect()
	if err != nil {
		return common.APIError(http.StatusInternalServerError, err)
	}

	session := conn.NewSession(nil)
	tx, err := session.Begin()
	if err != nil {
		return common.APIError(http.StatusInternalServerError, err)
	}
	defer tx.RollbackUnlessCommitted()
	defer session.Close()
	defer conn.Close()

	err = db.Insert(tx, db.TableTrip, *t)
	if err != nil {
		return common.APIError(http.StatusInternalServerError, err)
	}

	err = db.Insert(tx, db.TableTripItinerary, defaultItinerary)
	if err != nil {
		return common.APIError(http.StatusInternalServerError, err)
	}

	err = db.Insert(tx, db.TableTripParticipant, ownerParticipant)
	if err != nil {
		return common.APIError(http.StatusInternalServerError, err)
	}

	tx.Commit()

	result, err := db.QueryOne(session, db.TableTrip, t.ID, Trip{})
	if err != nil {
		return common.APIError(http.StatusInternalServerError, err)
	}

	return common.APIResponse(result, http.StatusCreated)
}

//Update change trip attributes in the database
func (t *Trip) Update(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	conn, err := db.Connect()
	defer conn.Close()
	if err != nil {
		return common.APIError(http.StatusInternalServerError, err)
	}

	session := conn.NewSession(nil)
	defer session.Close()

	tokenUser := common.GetTokenUser(request)
	if !tokenUser.IsAdmin() {
		filter := dbr.And(
			dbr.Eq("trip_id", request.PathParameters["id"]),
			dbr.Eq("user_id", tokenUser.UserID),
			dbr.Or(
				dbr.Eq("role", ParticipantOwnerRole),
				dbr.Eq("role", ParticipantAdminRole),
			),
		)
		total, err := db.Validate(session, []string{"count(id) total"}, db.TableTripParticipant, filter)
		if err != nil {
			return common.APIError(http.StatusInternalServerError, err)
		}
		if total <= 0 {
			return common.APIError(http.StatusForbidden, errors.New("only trip admin or owner can make changes"))
		}
	}

	jsonMap := make(map[string]interface{})
	err = json.Unmarshal([]byte(request.Body), &jsonMap)
	if err != nil {
		return common.APIError(http.StatusBadRequest, err)
	}

	jsonMap["updated_by"] = tokenUser.UserID
	jsonMap["updated_date"] = time.Now()

	tx, err := session.Begin()
	if err != nil {
		return common.APIError(http.StatusInternalServerError, err)
	}
	defer tx.RollbackUnlessCommitted()

	err = db.Update(tx, db.TableTrip, request.PathParameters["id"], *t, jsonMap)
	if err != nil {
		return common.APIError(http.StatusInternalServerError, err)
	}

	tx.Commit()

	result, err := db.QueryOne(session, db.TableTrip, request.PathParameters["id"], Trip{})
	if err != nil {
		return common.APIError(http.StatusInternalServerError, err)
	}

	return common.APIResponse(result, http.StatusOK)
}

//Delete removes event from the database
func (t *Trip) Delete(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	conn, err := db.Connect()
	defer conn.Close()
	if err != nil {
		return common.APIError(http.StatusInternalServerError, err)
	}

	session := conn.NewSession(nil)
	defer session.Close()

	tokenUser := common.GetTokenUser(request)
	if !tokenUser.IsAdmin() {
		filter := dbr.And(
			dbr.Eq("trip_id", request.PathParameters["id"]),
			dbr.Eq("user_id", tokenUser.UserID),
			dbr.Eq("role", ParticipantOwnerRole),
		)
		total, err := db.Validate(session, []string{"count(id) total"}, db.TableTripParticipant, filter)
		if err != nil {
			return common.APIError(http.StatusInternalServerError, err)
		}
		if total <= 0 {
			return common.APIError(http.StatusForbidden, errors.New("only trip participant can access this resource"))
		}
	}

	err = db.Delete(session, db.TableTrip, request.PathParameters["id"])
	if err != nil {
		return common.APIError(http.StatusInternalServerError, err)
	}

	return common.APIResponse(nil, http.StatusOK)
}
