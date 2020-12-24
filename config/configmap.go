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

package config

import (
	"cim/kubernetes"
	"cim/logs"
	"cim/types"
	"encoding/json"
	"io/ioutil"
	"log"
	"os"
	"strconv"
)

func BackwardCompatible() bool {
	_, err := os.Stat("/opt/conf/cim.json")
	logs.LogConf.Error("Logs directory info : ", err)
	if os.IsNotExist(err) {
		return false
	}
	return true

}

func LoadDefaultCimConfig() {
	var content []byte
	var err error
	if BackwardCompatible() {
		content, err = ioutil.ReadFile("/opt/conf/cim.json")
	} else {
		if kubernetes.ConfigMapExist() {
			log.Println("Inside LoadDefaultCimConfig if kubernetes.ConfigMapExist()")
			lst, _ := kubernetes.KubeConnect.GetCimConfigMap()
			log.Println(lst.Data["cim.json"])
			content = []byte(lst.Data["cim.json"])
		} else {
			content, err = ioutil.ReadFile("/opt/config/cim.json")
		}
	}
	if err != nil {
		logs.LogConf.Exception("Error in reading cim config", err.Error())
	}
	log.Println("Initial config data", string(content[:]))
	data := types.CimConfigObj
	err = json.Unmarshal(content, &data)
	if err != nil {
		panic(err)
	}
	types.ConfigDetails = data.CimConfig.App
	types.CimConfigObj = data

	types.AppPort = strconv.Itoa(types.CimConfigObj.CimConfig.Subjects.AppPort)

	log.Println("Unmarshall data", types.CimConfigObj.CimConfig.Subjects)
	log.Println("Unmarshall data", types.CimConfigObj.CimConfig.App)
	log.Println("Unmarshall data", types.CimConfigObj.CimConfig.Lmaas)
	return
}

func PrintConfigurationUsedByCim() {
	var content []byte
	if BackwardCompatible() {
		content, _ = ioutil.ReadFile("/opt/conf/cim.json")
	} else {
		if kubernetes.ConfigMapExist() {
			lst, _ := kubernetes.KubeConnect.GetCimConfigMap()
			content = []byte(lst.Data["cim.json"])
		} else {
			content, _ = ioutil.ReadFile("/opt/config/cim.json")
		}
	}
	logs.LogConf.Info("Final cim.json used", string(content[:]))
}
