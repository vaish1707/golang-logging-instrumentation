package main

import (
	"fmt"
	"net/http"

	"github.com/gorilla/mux"
	log "github.com/vaish1707/golang-logging-instrumentation/logger"
	"github.com/vaish1707/golang-logging-instrumentation/utils"
	"go.mongodb.org/mongo-driver/bson"
	"go.uber.org/zap/zapcore"
)

type user struct {
	UserID   string `json:"userid" validate:"-"`
	UserName string `json:"username" validate:"required"`
	Account  string `json:"account" validate:"required"`
	Amount   int
}

type paymentData struct {
	Amount int `json:"amount" validate:"required"`
}

func createUser() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var u user
		if err := utils.ReadBody(w, r, &u); err != nil {
			log.Ctx(r.Context()).Error(err.Error(), []zapcore.Field{}...)
			return
		}
		metadata := utils.GetExtraFields(r, u.UserID, "user-service", "createUser")

		log.Ctx(r.Context()).Info("Create user controller called", metadata...)

		usercollection := mongodbClient.MongoClient.Database("otel").Collection("users")
		_, mongoErr := usercollection.InsertOne(r.Context(), u)
		if mongoErr != nil {
			log.Ctx(r.Context()).Error(mongoErr.Error(), metadata...)
			utils.WriteErrorResponse(w, http.StatusInternalServerError, mongoErr)
			return
		}

		log.Ctx(r.Context()).Info("Successfully completed create user request", metadata...)

		utils.WriteResponse(w, http.StatusCreated, u)
	}
}

func getUser() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID := mux.Vars(r)["userID"]

		metadata := utils.GetExtraFields(r, userID, "user-service", "getUser")

		log.Ctx(r.Context()).Info("Get user controller called", metadata...)

		usercollection := mongodbClient.MongoClient.Database("otel").Collection("users")

		// create an empty struct
		data := &user{}
		filter := bson.M{"userid": userID}

		res := usercollection.FindOne(r.Context(), filter)
		if err := res.Decode(data); err != nil {
			log.Ctx(r.Context()).Error(err.Error(), metadata...)
			utils.WriteErrorResponse(w, http.StatusInternalServerError, fmt.Errorf("get user error: %w", err))
			return
		}

		log.Ctx(r.Context()).Info("Successfully completed get user request", metadata...)

		utils.WriteResponse(w, http.StatusOK, data)
	}
}

func updateUser() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID := mux.Vars(r)["userID"]

		metadata := utils.GetExtraFields(r, userID, "user-service", "updateUser")

		log.Ctx(r.Context()).Info("Update user controller called", metadata...)

		var data paymentData
		if err := utils.ReadBody(w, r, &data); err != nil {
			log.Ctx(r.Context()).Error(err.Error(), metadata...)
			return
		}
		usercollection := mongodbClient.MongoClient.Database("otel").Collection("users")

		// create an empty struct
		userDat := &user{}
		filter := bson.M{"userid": userID}

		singleUser := usercollection.FindOne(r.Context(), filter)

		if err := singleUser.Decode(userDat); err != nil {
			log.Ctx(r.Context()).Error(err.Error(), metadata...)
			utils.WriteErrorResponse(w, http.StatusInternalServerError, err)
			return
		}
		userDat.Amount = userDat.Amount + data.Amount

		_, updateErr := usercollection.ReplaceOne(r.Context(), filter, userDat)
		if updateErr != nil {
			log.Ctx(r.Context()).Error(updateErr.Error(), metadata...)
			utils.WriteErrorResponse(w, http.StatusInternalServerError, updateErr)
			return
		}

		log.Ctx(r.Context()).Info("Successfully completed update user request", metadata...)

		w.WriteHeader(http.StatusOK)
	}
}
