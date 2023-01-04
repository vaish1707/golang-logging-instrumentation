package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/gorilla/mux"
	log "github.com/vaish1707/golang-logging-instrumentation/logger"
	"github.com/vaish1707/golang-logging-instrumentation/utils"
)

type paymentData struct {
	Amount int `json:"amount" validate:"required"`
}

func transferAmount() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID := mux.Vars(r)["userID"]

		metadata := utils.GetExtraFields(r, userID, "payment-service", "transferAmount")

		log.Ctx(r.Context()).Info("Payment controller called", metadata...)

		var data paymentData
		if err := utils.ReadBody(w, r, &data); err != nil {
			log.Ctx(r.Context()).Error(err.Error(), metadata...)
			return
		}

		payload, err := json.Marshal(data)
		if err != nil {
			log.Ctx(r.Context()).Error(err.Error(), metadata...)
			utils.WriteErrorResponse(w, http.StatusInternalServerError, err)
			return
		}

		// send the request to user service
		url := fmt.Sprintf("http://%s/users/%s", userUrl, userID)
		resp, err := utils.SendRequest(r.Context(), http.MethodPut, url, payload)
		if err != nil {
			log.Ctx(r.Context()).Error(err.Error(), metadata...)
			utils.WriteErrorResponse(w, http.StatusInternalServerError, err)
			return
		}

		b, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			log.Ctx(r.Context()).Error(err.Error(), metadata...)
			utils.WriteErrorResponse(w, http.StatusInternalServerError, err)
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			log.Ctx(r.Context()).Error(fmt.Errorf("payment failed. got response: %s", b).Error(), metadata...)
			utils.WriteErrorResponse(w, http.StatusInternalServerError, fmt.Errorf("payment failed. got response: %s", b))
			return
		}

		log.Ctx(r.Context()).Info("Successfully completed payment request", metadata...)

		utils.WriteResponse(w, http.StatusOK, data)
	}
}
