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

package common_infra

import (
	httpUtility "cim/http_utility"
	"cim/kubernetes"
	"cim/logs"
	"cim/types"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"time"
)

func ReadCommonInfraConf() {
	var content []byte
	_, err := os.Stat("/opt/conf/static/common-infra.json")
	logs.LogConf.Info("Common Infra directory info : ", err)
	if os.IsNotExist(err) {
		logs.LogConf.Info("Fetching Common Infra information from TMaaS")
		content = getCommonInfraInfoFromTMaas()
	} else {
		content, err = ioutil.ReadFile("/opt/conf/static/common-infra.json")
		if err != nil {
			logs.LogConf.Exception("Error while reading common infra config from /opt/conf/static/common-infra.json location", err.Error(), "Terminating Self.")
		}
	}
	logs.LogConf.Info("Common infra config data", string(content[:]))
	data := types.CommonInfraConfigObj
	err = json.Unmarshal(content, &data)
	if err != nil {
		logs.LogConf.Exception("Error while unmarshalling common-infra configuration data", err.Error(), "Terminating Self")
	}
	types.CommonInfraConfigObj = data
	logs.LogConf.Debug("Unmarshall common infra config data", *types.CommonInfraConfigObj)
	fmt.Println("common infra", string(content[:]))
	fmt.Println("log retrx ....................", types.CommonInfraConfigObj.EnableRetx)
	return
}
func getCommonInfraInfoFromTMaas() (content []byte) {
	if types.TMaasFqdn == "" {
		types.TMaasFqdn = GetTopoGwFqdn()
	}
	endPoint := "http://" + types.TMaasFqdn + "/api/v1/tmaas/config/common-infra"
	logs.LogConf.Info("TMaaS End Point : ", endPoint)
	var response *http.Response
	var err error
	for i := 0; i < types.ConfigDetails.RemoteSvcRetryCount; i++ {
		response, err = httpUtility.HttpGet(endPoint, "application/json", []byte(""), types.ProtocolHTTP)
		if err != nil {
			logs.LogConf.Error("Error while sending Http Get Request to TMaaS", err.Error())
			time.Sleep(1 * time.Second)
			continue
		}
		if response.StatusCode != 200 {
			logs.LogConf.Error("Error Response recieved from TMaaS", response.StatusCode)
			time.Sleep(1 * time.Second)
			continue
		}
		if response.StatusCode == 200 {
			logs.LogConf.Info("Http GET request sent and recieved", response.StatusCode, "TMaaS")
			break
		}
	}
	if err != nil {
		logs.LogConf.Exception("Unable to get common-infra config details from TMaaS even after", types.ConfigDetails.RemoteSvcRetryCount, "Retries", "Terminating Self.")
	}
	content, err = ioutil.ReadAll(response.Body)
	if err != nil {
		logs.LogConf.Exception("Error while reading TMaaS response for common-infra configuration request", err.Error(), "Terminating Self.")
	}
	logs.LogConf.Debug("http Response body .........", string(content[:]))
	return
}

func GetTopoGwFqdn() (tmaasFqdn string) {
	var err error
	if types.CommonInfraConfigObj != nil {
		return "topo-gw." + types.Namespace + ".svc.cluster.local:8080"
	}
	tmaasFqdn, err = kubernetes.KubeConnect.GetTopoGwFqdn()
	if err != nil {
		logs.LogConf.Exception("Error while getting topo gw FQDN.", err.Error(), ". Terminating Self.")
	}
	return
}

func GetTopoEngineFqdn() (topoEngineFqdn string) {

	var err error

	if types.CommonInfraConfigObj != nil {
		topoEngineFqdn = types.CommonInfraConfigObj.TopoEngineFqdn
		if topoEngineFqdn == "" {
			topoEngineFqdn = "topo-engine." + types.Namespace + ".svc.cluster.local:8080"
		}
		return
	}
	if types.XGVelaNamespace == "" {
		types.XGVelaNamespace, err = kubernetes.KubeConnect.GetXGVelaNamespace()
		if err != nil {
			logs.LogConf.Exception("Error while getting XGVela Namespace", err.Error(), ". Terminating Self.")
		}
	}
	topoEngineFqdn = "topo-engine." + types.XGVelaNamespace + ".svc.cluster.local:8080"
	return
}

func GetApigwRestFqdn() string {
	var err error
	if types.CommonInfraConfigObj != nil {
		if types.CommonInfraConfigObj.APIGwFqdn == "" {
			types.CommonInfraConfigObj.APIGwFqdn = "config-service." + types.Namespace + ".svc.cluster.local:8008"
		}
		return types.CommonInfraConfigObj.APIGwFqdn
	}
	if types.XGVelaNamespace == "" {
		types.XGVelaNamespace, err = kubernetes.KubeConnect.GetXGVelaNamespace()
		if err != nil {
			logs.LogConf.Exception("Error while getting XGVela Namespace", err.Error(), ". Terminating Self.")
		}
	}
	apigwRestFqdn := "config-service." + types.XGVelaNamespace + ".svc.cluster.local:8008"
	return apigwRestFqdn
}
