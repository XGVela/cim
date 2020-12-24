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

package kubernetes

import (
	"cim/logs"
	"cim/types"
	"cim/utility"
	"encoding/json"
	"io/ioutil"
	"strings"

	v1 "k8s.io/api/core/v1"
	errors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var (
	tmaasAnna *utility.TmaaSAnnotations
)

func UpdateCimYang() {
	path := "/opt/config/cim.yang"
	read, err := ioutil.ReadFile(path)
	if err != nil {
		logs.LogConf.Error("Error While Reading cim.yang File from path:", path, err)
		panic(err)
	}

	newContents := strings.Replace(string(read), "{{.Values.nf.nfId}}", types.NfID+"-"+types.MicroserviceName, -1)
	err = ioutil.WriteFile(path, []byte(newContents), 0)
	if err != nil {
		logs.LogConf.Exception(err)
	}
}

func ConfigMapExist() bool {
	lst, err := KubeConnect.GetCimConfigMap()
	if err != nil {
		return false
	}
	if lst == nil {
		return false
	}
	return true
}

// list out configmap in given namespace
func (c *KubeClient) GetCimConfigMap() (*v1.ConfigMap, error) {
	Configmap, err := c.Client.CoreV1().ConfigMaps(types.Namespace).Get(types.ConfigmapName, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	return Configmap, nil
}

func (c *KubeClient) CreateCimConfigMap() error {

	UpdateCimYang()
	tmaasAnna, err := utility.GetAnnotations()
	if err != nil {
		logs.LogConf.Exception("Error While Reading Tmaas Annotaion", err)
	}

	cimData, _ := ioutil.ReadFile("/opt/config/cim.json")
	yangData, _ := ioutil.ReadFile("/opt/config/cim.yang")
	var data interface{}
	err = json.Unmarshal(cimData, &data)

	if err != nil {
		logs.LogConf.Exception(err)
	}

	label := map[string]string{
		"microSvcName": tmaasAnna.NfServiceType,
	}

	cimUpdatePolicy := map[string]string{
		"cim": "Dynamic",
	}
	cimUpdatePolicyJson, _ := json.Marshal(cimUpdatePolicy)

	tmaasAnno := map[string]string{
		"vendorId":      tmaasAnna.VendorID,
		"xgvelaId":      tmaasAnna.XGVelaID,
		"nfClass":       tmaasAnna.NfClass,
		"nfType":        tmaasAnna.NfType,
		"nfId":          types.NfID,
		"nfServiceId":   tmaasAnna.NfServiceID,
		"nfServiceType": tmaasAnna.NfServiceType,
	}
	tmaasAnnoJson, _ := json.Marshal(tmaasAnno)
	annotn := map[string]string{
		"configMgmt":       "enabled",
		"svcVersion":       types.AppVersion,
		"xgvela.com/tmaas": string(tmaasAnnoJson),
		"nwFnPrefix":       types.Namespace,
	}

	cm := v1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ConfigMap",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:        types.ConfigmapName,
			Namespace:   types.Namespace,
			Annotations: annotn,
			Labels:      label,
		},
		Data: map[string]string{
			"__CFG_TYPE":        "mgmt-cfg",
			"cim.json":          string(cimData),
			"cim.yang":          string(yangData),
			"updatePolicy.json": string(cimUpdatePolicyJson),
			"revision":          "0",
		},
	}
	_, err = c.Client.CoreV1().ConfigMaps(types.Namespace).Create(&cm)
	if errors.IsAlreadyExists(err) || errors.IsConflict(err) {
		// Resource already exists. Carry on.
		logs.LogConf.Info("config map is already exist")
		return nil
	}
	return err

}

func (c *KubeClient) GetCimCMRevision() {
	cm, err := c.GetCimConfigMap()
	if err != nil {
		logs.LogConf.Error(err.Error())
		return
	}
	types.ConfigMapRevision = cm.Data["revision"]
}
