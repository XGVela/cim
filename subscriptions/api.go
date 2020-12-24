// Copyright 2020 Mavenir
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package subscriptions

import (
	"cim/logs"
	"cim/utility"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
)

var (
	Subscriptions = make(map[string]*Subscribe)
	//StartConsume to start message conumption
	StartConsume bool
)

//Subscribe struct for event subscription
type Subscribe struct {
	LocalEventName []string `json:"localEventName"`
	NfNamingCode   []string `json:"nfNamingCode,omitempty"`
	NfcNamingCode  []string `json:"nfcNamingCode,omitempty"`
	LocalNfID      []string `json:"localNfId,omitempty"`
}

//SubscribeResponse
type SubscribeResponse struct {
	SubscriptionID string `json:"subscriptionId,omitempty"`
	ErrorReason    string `json:"error,omitempty"`
}

//SubscribeEvents gets list of events to subscribe
func SubscribeEvents(w http.ResponseWriter, r *http.Request) {
	var resp SubscribeResponse
	if !StartConsume {
		logs.LogConf.Info("Running Kafka Consumer ..........")
		StartConsume = true
		go ConsumeMessages()

	}

	w.Header().Set("Content-Type", "application/json")

	var subEvents *Subscribe
	reqBody, err := ioutil.ReadAll(r.Body)
	if err != nil {
		logs.LogConf.Error("error while reading the request body", err)
		w.WriteHeader(http.StatusInternalServerError)
		resp.ErrorReason = "Unable to process the request body."
		json.NewEncoder(w).Encode(resp)
		return
	}

	logs.LogConf.Info("SubscribeEvents: recieved subscription lists are:", string(reqBody))

	err = json.Unmarshal(reqBody, &subEvents)
	if err != nil {
		logs.LogConf.Error("SubscribeEvents: error while unmarshalling the request body", err)
		w.WriteHeader(http.StatusBadRequest)
		resp.ErrorReason = "Invalid request body."
		json.NewEncoder(w).Encode(resp)
		return
	}

	if len(subEvents.LocalEventName) == 0 {
		logs.LogConf.Error("localEventName is empty. Invalid request")
		w.WriteHeader(http.StatusBadRequest)
		resp.ErrorReason = "LocalEventName must not be empty."
		json.NewEncoder(w).Encode(resp)
		return
	}

	/*	if len(subEvents.NfNamingCode) == 0 {
		logs.LogConf.Error("nfNamingCode is empty. Invalid request")
		w.WriteHeader(http.StatusBadRequest)
		resp.ErrorReason = "NfNamingCode must not be empty."
		json.NewEncoder(w).Encode(resp)
		return
	}*/

	//generate a subscription id
	subscriptionID := utility.GenerateCorrelationId(100000, 999999)
	subID := strconv.Itoa(subscriptionID)
	Subscriptions[subID] = subEvents

	logs.LogConf.Info("subscription id is :", subID)

	resp.SubscriptionID = subID

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(resp)

}

//UnsubscribeEvents delete list of events  subscribed
func UnsubscribeEvents(w http.ResponseWriter, r *http.Request) {
	var resp SubscribeResponse
	params := mux.Vars(r)
	subID := params["subscriptionId"]
	logs.LogConf.Info("unsubscribe subId =", subID)

	w.Header().Set("Content-Type", "application/json")
	_, ok := Subscriptions[subID]
	if !ok {
		logs.LogConf.Info("UnsubscribeEvents: requested subscription ID does not exist")
		w.WriteHeader(http.StatusNotFound)
		resp.ErrorReason = "Subscription ID does not exist."
		json.NewEncoder(w).Encode(resp)
		return
	}

	logs.LogConf.Info("before unsubscribing the data is :", Subscriptions[subID])

	delete(Subscriptions, subID)
	logs.LogConf.Info("after unsubscribing the data is :", Subscriptions[subID])
	if len(Subscriptions) == 0 {
		logs.LogConf.Info("No subscription exist. Stopping the kafka consumer...")
		StartConsume = false
	}

	w.WriteHeader(http.StatusNoContent)
	json.NewEncoder(w).Encode("Unsubscribed successfully")

}

//GetSubscribedEvents gets list of events subscribed
func GetSubscribedEvents(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(Subscriptions)

}

//UpdateSubscribedEvents update the subscribed event lists
func UpdateSubscribedEvents(w http.ResponseWriter, r *http.Request) {
	var (
		subEvents *Subscribe
		resp      SubscribeResponse
	)

	params := mux.Vars(r)
	subID := params["subscriptionId"]

	w.Header().Set("Content-Type", "application/json")
	_, ok := Subscriptions[subID]
	if !ok {
		logs.LogConf.Info("UpdateSubscribedEvents: requested subscription ID does not exist")
		w.WriteHeader(http.StatusNotFound)
		resp.ErrorReason = "Subscription ID does not exist."
		json.NewEncoder(w).Encode(resp)
		return
	}

	reqBody, err := ioutil.ReadAll(r.Body)
	if err != nil {
		logs.LogConf.Error("error while reading the request body", err)
		w.WriteHeader(http.StatusInternalServerError)
		resp.ErrorReason = "Unable to process the request body."
		json.NewEncoder(w).Encode(resp)
		return
	}

	err = json.Unmarshal(reqBody, &subEvents)
	if err != nil {
		logs.LogConf.Error("UpdateSubscribedEvents: error while unmarshalling the request body", err)
		w.WriteHeader(http.StatusBadRequest)
		resp.ErrorReason = "Invalid request body."
		json.NewEncoder(w).Encode(resp)
		return
	}

	Subscriptions[subID] = subEvents

	fmt.Printf("updated subscribed event lists :%+v", Subscriptions[subID])

	json.NewEncoder(w).Encode("subscribed event updated successfully")

}
