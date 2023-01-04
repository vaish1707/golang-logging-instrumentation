package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/google/uuid"
	"github.com/vaish1707/golang-logging-instrumentation/utils"
	"go.mongodb.org/mongo-driver/bson"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"

	log "github.com/vaish1707/golang-logging-instrumentation/logger"
)

type orderData struct {
	ID          string `json:"id"`
	UserID      string `json:"userid"`
	ProductName string `json:"product_name" validate:"required"`
	Price       int    `json:"price" validate:"required"`
}

type orderDat struct {
	ID          string `json:"id"`
	UserID      string `json:"userid"`
	Account     string `json:"account"`
	OrderStatus string `json:"order_status"`
	ProductName string `json:"product_name"`
	Price       int    `json:"price"`
}

type userData struct {
	UserID   string `json:"userid"`
	UserName string `json:"username"`
	Account  string `json:"account"`
	Amount   int
}

func createOrder() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var request orderData
		if err := utils.ReadBody(w, r, &request); err != nil {
			return
		}

		metadata := utils.GetExtraFields(r, request.UserID, "order-service", "createOrder")

		log.Ctx(r.Context()).Info("Order controller called", metadata...)

		// get user details from user service
		url := fmt.Sprintf("http://%s/users/%s", userUrl, request.UserID)
		userResponse, err := utils.SendRequest(r.Context(), http.MethodGet, url, nil)
		if err != nil {
			log.Ctx(r.Context()).Error(err.Error(), metadata...)
			utils.WriteResponse(w, http.StatusInternalServerError, err)
			return
		}

		b, err := ioutil.ReadAll(userResponse.Body)
		if err != nil {
			log.Ctx(r.Context()).Error(err.Error(), metadata...)
			utils.WriteErrorResponse(w, http.StatusInternalServerError, err)
			return
		}
		defer userResponse.Body.Close()

		if userResponse.StatusCode != http.StatusOK {
			log.Ctx(r.Context()).Error(fmt.Errorf("payment failed. got response: %s", b).Error(), metadata...)
			utils.WriteErrorResponse(w, userResponse.StatusCode, fmt.Errorf("payment failed. got response: %s", b))
			return
		}

		var userDat userData
		if err := json.Unmarshal(b, &userDat); err != nil {
			log.Ctx(r.Context()).Error(err.Error(), metadata...)
			utils.WriteErrorResponse(w, http.StatusInternalServerError, err)
			return
		}
		ctx := r.Context()
		span := trace.SpanFromContext(ctx)

		// basic check for the user balance
		if userDat.Amount < request.Price {
			span.RecordError(errors.New("insufficient balance"))
			span.SetStatus(codes.Error, "failed due to insufficient balance")
			log.Ctx(r.Context()).Warn(fmt.Errorf("insufficient balance. add %d more amount to account", request.Price-userDat.Amount).Error(), metadata...)
			utils.WriteErrorResponse(w, http.StatusUnprocessableEntity, fmt.Errorf("insufficient balance. add %d more amount to account", request.Price-userDat.Amount))
			return
		}

		// insert the order into order table
		ordercollection := mongodbClient.MongoClient.Database("otel").Collection("orders")

		var orderData orderDat
		orderid := uuid.New()
		orderData.ID = orderid.String()
		orderData.UserID = userDat.UserID
		orderData.Account = userDat.Account
		orderData.ProductName = request.ProductName
		orderData.Price = request.Price
		orderData.OrderStatus = "SUCCESS"

		_, mongoErr := ordercollection.InsertOne(r.Context(), orderData)
		if mongoErr != nil {
			log.Ctx(r.Context()).Error(mongoErr.Error(), metadata...)
			utils.WriteErrorResponse(w, http.StatusInternalServerError, mongoErr)
			return
		}

		// update the pending amount in user table
		usercollection := mongodbClient.MongoClient.Database("otel").Collection("users")

		filter := bson.M{"userid": userDat.UserID}
		singleUserData := &userData{}
		singleUser := usercollection.FindOne(r.Context(), filter)

		if err := singleUser.Decode(singleUserData); err != nil {
			log.Ctx(r.Context()).Error(err.Error(), metadata...)
			utils.WriteErrorResponse(w, http.StatusInternalServerError, err)
			return
		}

		singleUserData.Amount = singleUserData.Amount - request.Price

		_, updateErr := usercollection.ReplaceOne(r.Context(), filter, singleUserData)
		if updateErr != nil {
			log.Ctx(r.Context()).Error(updateErr.Error(), metadata...)
			utils.WriteErrorResponse(w, http.StatusInternalServerError, err)
			return
		}

		log.Ctx(r.Context()).Info("Successfully completed order request", metadata...)
		// send response
		response := request
		utils.WriteResponse(w, http.StatusCreated, response)
	}
}
