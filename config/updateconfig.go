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
	httpUtility "cim/http_utility"
	"cim/kubernetes"
	"cim/logs"
	"cim/types"
	"encoding/json"
	"io/ioutil"
	"log"
	"os"
	"reflect"
)

type Register struct {
	ContainerName string         `json:"container_name,omitempty"`
	ContainerID   string         `json:"container_id"`
	Lms           types.Lmaas    `json:"lmaas"`
	AppSetting    types.Subjects `json:"app_settings"`
}

// UpdateConfigMap will update the cim.json and create configmap
func UpdateConfigMap(registerDetails Register) {
	ReadConfig()
	UpdateConfig(registerDetails)
	WriteConfig()
	if !kubernetes.ConfigMapExist() {
		logs.LogConf.Debug("Creating configmap")
		err := kubernetes.KubeConnect.CreateCimConfigMap()

		if err != nil {
			logs.LogConf.Error("Error while creating configmap", err)
			panic(err)
		}
		logs.LogConf.Info("ConfigMap is created")

	} else {
		logs.LogConf.Info("ConfigMap is already exist")
	}

	content, err := ioutil.ReadFile("/opt/config/cim.json")
	if err != nil {
		log.Println("Error in reading cim.json", err.Error())
	}
	log.Println("Initial config data", string(content[:]))
	data := types.CimConfigObj
	err = json.Unmarshal(content, &data)
	if err != nil {
		logs.LogConf.Error("Error while marshaling cim data", err)
		panic(err)
	}

	types.ConfigDetails = data.CimConfig.App
	types.CimConfigObj = data
	log.Println("Loaded data after registration", types.CimConfigObj.CimConfig.Subjects)
	log.Println("Loaded data after registration", types.CimConfigObj.CimConfig.App)
	log.Println("Loaded data after registration", types.CimConfigObj.CimConfig.Lmaas)
}

func WriteConfig() {

	file, _ := json.MarshalIndent(types.CimConfigObj, "", " ")
	err := ioutil.WriteFile("/opt/config/cim.json", file, 0644)
	if err != nil {
		logs.LogConf.Exception("Error while overiding cim.json", err)
	}
}

func UpdateConfig(registerDetails Register) {
	if !reflect.DeepEqual(registerDetails.Lms, types.Lmaas{}) {
		if registerDetails.Lms.BufferSize <= 0 {
			registerDetails.Lms.BufferSize = types.CimConfigObj.CimConfig.Lmaas.BufferSize
		}
		if registerDetails.Lms.FlushTimeout <= 0 {
			registerDetails.Lms.FlushTimeout = types.CimConfigObj.CimConfig.Lmaas.FlushTimeout
		}
		if registerDetails.Lms.LoggingMode == "" {
			registerDetails.Lms.LoggingMode = types.CimConfigObj.CimConfig.Lmaas.LoggingMode
		}
		if registerDetails.Lms.MaxAge <= 0 {
			registerDetails.Lms.MaxAge = types.CimConfigObj.CimConfig.Lmaas.MaxAge
		}
		if registerDetails.Lms.MaxBackupFiles <= 0 {
			registerDetails.Lms.MaxBackupFiles = types.CimConfigObj.CimConfig.Lmaas.MaxBackupFiles
		}
		if registerDetails.Lms.MaxFileSizeInMB <= 0 {
			registerDetails.Lms.MaxFileSizeInMB = types.CimConfigObj.CimConfig.Lmaas.MaxFileSizeInMB
		}
		types.CimConfigObj.CimConfig.Lmaas = &registerDetails.Lms
	}

	if !reflect.DeepEqual(registerDetails.AppSetting, types.Subjects{}) {
		if registerDetails.AppSetting.AppPort <= 0 {
			registerDetails.AppSetting.AppPort = types.CimConfigObj.CimConfig.Subjects.AppPort
		}
		if registerDetails.AppSetting.Http2Enabled != types.CimConfigObj.CimConfig.Subjects.Http2Enabled {
			types.CimConfigObj.CimConfig.Subjects = &registerDetails.AppSetting
			// Http Cli update
			httpUtility.InitializeHttpClient()
		} else {
			types.CimConfigObj.CimConfig.Subjects = &registerDetails.AppSetting
		}
	}
}

func ReadConfig() {
	jsonFile, err := os.Open("/opt/config/cim.json")
	// // if we os.Open returns an error then handle it
	if err != nil {
		logs.LogConf.Exception("Error While Reading /opt/config/cim.json File", err)
	}
	logs.LogConf.Debug("Successfully Opened /opt/config/cim.json")
	// // defer the closing of our jsonFile so that we can parse it later on
	defer jsonFile.Close()

	byteValue, _ := ioutil.ReadAll(jsonFile)

	json.Unmarshal([]byte(byteValue), &types.CimConfigObj)
}
