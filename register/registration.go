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

package register

import (
	"cim/agents"
	common_infra "cim/common-infra"
	"cim/config"
	"cim/kubernetes"
	"cim/logs"
	"cim/types"
	"cim/utility"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
)

type Register struct {
	ContainerName string         `json:"container_name"`
	ContainerID   string         `json:"container_id"`
	Lms           types.Lmaas    `json:"lmaas,omitempty"`
	AppSetting    types.Subjects `json:"app_settings,omitempty"`
}

//RegisterResp registration Response struct
type RegisterResp struct {
	Status             string                    `json:"status"`
	TMaasFqdn          string                    `json:"tmaas_fqdn"`
	ErrMsg             string                    `json:"err_msg,omitempty"`
	ApigwRestFqdn      string                    `json:"apigw_rest_fqdn"`
	ApigwRestURIPrefix string                    `json:"apigw_rest_uri_prefix"`
	ApigwAuthType      string                    `json:"apigw_auth_type"`
	ApigwUsername      string                    `json:"apigw_username"`
	ApigwPassword      string                    `json:"apigw_password"`
	TopoEngineFqdn     string                    `json:"topo_engine_fqdn"`
	TmaasHeaders       *utility.TmaaSAnnotations `json:"tmaas_annotations"`
}

var (
	once      sync.Once
	opRegonce sync.Once

	XGVelaRegistrationDone = make(chan bool)
)

//XGVelaRegistration register
func XGVelaRegistration(w http.ResponseWriter, r *http.Request) {
	var registerDetails Register
	var registerRespDetails RegisterResp
	reqBody, err := ioutil.ReadAll(r.Body)
	if err != nil {
		logs.LogConf.Error("error while reading the request body", err)
		logs.LogConf.Info("MICROSERVICE REGISTRATION FAILURE")
		w.WriteHeader(http.StatusBadRequest)
		registerRespDetails.Status = "failure"
		registerRespDetails.ErrMsg = "Unable to process the request body."
		json.NewEncoder(w).Encode(registerRespDetails)
		return
	}

	err = json.Unmarshal(reqBody, &registerDetails)
	if err != nil {
		logs.LogConf.Error("XGVelaRegistration: error while unmarshalling the request body", err)
		logs.LogConf.Info("MICROSERVICE REGISTRATION FAILURE")
		w.WriteHeader(http.StatusBadRequest)
		registerRespDetails.Status = "failure"
		registerRespDetails.ErrMsg = "Invalid request body."
		json.NewEncoder(w).Encode(registerRespDetails)
		return
	}

	logs.LogConf.Debug("XGVela Registeration Request Body:", string(reqBody))
	if os.Getenv("CTR_FIELDS_MANDATORY") == "true" {
		if len(strings.TrimSpace(registerDetails.ContainerName)) == 0 ||
			len(strings.TrimSpace(registerDetails.ContainerID)) == 0 {
			w.WriteHeader(http.StatusBadRequest)
			registerRespDetails.Status = "failure"
			registerRespDetails.ErrMsg = "Invalid request body. Please provide mandatory fields container_name and container_id."
			json.NewEncoder(w).Encode(registerRespDetails)
			logs.LogConf.Error("Invalid XGVela Registration request body. Please provide mandatory fields container_name and container_id.")
			return
		}
	}
	types.ContainerName = registerDetails.ContainerName
	types.ContainerID = registerDetails.ContainerID

	//TODO Container id validation need to be implemented.
	req, _ := json.Marshal(registerDetails)
	logs.LogConf.Info("Registration request recieved  :", string(req))
	if !config.BackwardCompatible() && !kubernetes.ConfigMapExist() {
		config.UpdateConfigMap(config.Register(registerDetails))
	} else {
		logs.LogConf.Info("Configmap is Exist, Loading Cim Parameters From Configmap")
	}
	config.PrintConfigurationUsedByCim()
	//start watching container state
	//This watch should be invoked only once as app container can restart anytime.
	once.Do(func() {
		//handle all nats subscription
		agents.HandleNatsSubscriptions()
		time.Sleep(3 * time.Second)
		types.XGVelaRegistration <- true
	})

	//reset the notConfigured flag in case of app container restart
	agents.InitialNotConfiguredEvent = false

	logs.LogConf.Info("MICROSERVICE REGISTRATION SUCCCESSFUL")
	w.WriteHeader(http.StatusOK)
	registerRespDetails.Status = "success"
	if types.TMaasFqdn == "" {
		types.TMaasFqdn = common_infra.GetTopoGwFqdn()
	}
	if types.ApigwRestFqdn == "" {
		types.ApigwRestFqdn = common_infra.GetApigwRestFqdn()
	}
	if types.TopoEngineFqdn == "" {
		types.TopoEngineFqdn = common_infra.GetTopoEngineFqdn()
	}

	types.ApigwRestURIPrefix = "/restconf/tailf/query"
	registerRespDetails.TmaasHeaders, _ = utility.GetAnnotations()

	registerRespDetails.TMaasFqdn = types.TMaasFqdn
	registerRespDetails.ApigwRestFqdn = types.ApigwRestFqdn
	registerRespDetails.TopoEngineFqdn = types.TopoEngineFqdn
	registerRespDetails.ApigwRestURIPrefix = types.ApigwRestURIPrefix
	registerRespDetails.ApigwAuthType = "Basic"
	registerRespDetails.ApigwUsername = "admin"
	registerRespDetails.ApigwPassword = "admin"

	json.NewEncoder(w).Encode(registerRespDetails)
	types.CurrentCimState = types.CimStateRegistered
}
