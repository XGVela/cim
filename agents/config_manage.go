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

package agents

import (
	"cim/connection"
	httpUtility "cim/http_utility"
	"cim/kubernetes"
	"cim/logs"
	notify_util "cim/notification"
	"cim/promutil"
	"cim/types"
	"cim/utility"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/coreos/etcd/clientv3"
)

var (
	DayOneConfig string
)

//WatchConfigChange for configuration
func WatchConfigChange(etcd *clientv3.Client, etcdWatchKey, appPort string) {
	watchChan := etcd.Watch(context.Background(), etcdWatchKey, clientv3.WithPrefix())
	if watchChan == nil {
		logs.LogConf.Error("watch channel is nill")
		return
	}

	for watchResp := range watchChan {
		go func(resp clientv3.WatchResponse) {
			for _, event := range resp.Events {
				handleEtcdUpdate(event)
			}
		}(watchResp)
	}
}
func prepareCommitDetails(key string, revision string, err error) map[string]string {
	commitMsg := make(map[string]string)
	commitMsg["change-set-key"] = key
	commitMsg["revision"] = revision
	if err != nil {
		commitMsg["status"] = "failure"
		commitMsg["remarks"] = "failed to apply the config patch"
	} else {
		commitMsg["status"] = "success"
		commitMsg["remarks"] = "Successfully applied the config patch"
	}
	return commitMsg
}
func handleEtcdUpdate(event *clientv3.Event) {
	logs.LogConf.Info("Event received!", event.Type, "executed on ", string(event.Kv.Key[:]), "with value", string(event.Kv.Value[:]))
	eventType := fmt.Sprintf("%s", event.Type)
	keys := strings.Split(string(event.Kv.Key), "/")

	////// CIM SELF CONFIG UPDATE
	if keys[5] == "cim.json" && eventType == "PUT" {
		handleSelfConfigUpdate(event, keys, eventType)
	} else if types.MicroserviceName == keys[2] && types.AppVersion == keys[3] && eventType == "PUT" {
		handleAppConfigUpdate(event, keys, eventType)
	}

}
func handleSelfConfigUpdate(event *clientv3.Event, keys []string, eventType string) {
	logs.LogConf.Info("Event received!", event.Type, "executed on ", string(event.Kv.Key[:]), "with value", string(event.Kv.Value[:]))
	err := utility.ApplyJsonPatch(event.Kv.Value)
	commitMsg := prepareCommitDetails(string(event.Kv.Key[:]), keys[6], err)
	UpdateConfigCommitDetails(commitMsg, utility.GenerateCorrelationId(100000, 999999), "SELF")
	if strings.Contains(string(event.Kv.Value), "lmaas") {
		if !(strings.Contains(string(event.Kv.Value), "tcp")) {
			RecreateLumberjackObj()
		}

		switch types.CimConfigObj.CimConfig.Lmaas.LoggingMode {
		case "TCP":
			logs.LogConf.Info("Logging Method switched to TCP")
			InitialiseTCPLoggingParams()
			if TCPcon == nil {
				go CreateConnection()
			}
		case "KAFKA":
			logs.LogConf.Info("Logging Method switched to KAFKA")
			if KafkaLogConn == nil {
				KafkaLogConn, err = IntialiseKafkaProducer()
				if err != nil {
					logs.LogConf.Error("Getting error while switching", err)
				}
			}
		default:
			logs.LogConf.Info("Logging Method Switched to ", types.CimConfigObj.CimConfig.Lmaas.LoggingMode)
		}
	}
	if strings.Contains(string(event.Kv.Value[:]), "log_level") || strings.Contains(string(event.Kv.Value[:]), "cim_file_log") {
		logs.InitializeLogInfo(types.CimConfigObj.CimConfig.App.LogLevel, types.CimConfigObj.CimConfig.App.CimFileLog)
	}
	// Only for subscribe at runtime
	// No unsubscribe
	if strings.Contains(string(event.Kv.Value[:]), "app_settings/enable") {
		HandleNatsSubscriptions()
	}
	return
}
func handleAppConfigUpdate(event *clientv3.Event, keys []string, eventType string) {
	err := promutil.CounterAdd("cim_config_push_attempt_total", 1, map[string]string{"pod": types.PodID})
	if err != nil {
		log.Println("error while adding values in  counter  for total push config", err)
	}

	var data = map[string]string{
		"change-set-key": string(event.Kv.Key),
		"data-key":       keys[5],
		"config-patch":   string(event.Kv.Value),
		"revision":       keys[6],
	}
	//form commit-config key for failure handling
	configKeys := strings.Split(string(event.Kv.Key), "/")
	commitKey := configKeys[0:5]
	commitKey = append(commitKey, types.PodID)
	commitKey[0] = "commit-config"
	commitConfigKeys := strings.Join(commitKey, "/")

	jsonValue, err := json.Marshal(data)
	if err != nil {
		logs.LogConf.Error("error while marshalling update config data", err)
		_, etcdErr := connection.ConnectEtcd.Put(context.Background(), commitConfigKeys, "-1")
		if etcdErr != nil {
			logs.LogConf.Error("error while updating etcd", etcdErr)
		}
		return
	}

	var appPort = types.AppPort
	if appPort == "" {
		appPort = "9999"
	}

	response, err := httpUtility.HttpPost("http://localhost:"+appPort+"/updateConfig", "application/json", jsonValue)
	if err != nil {
		logs.LogConf.Error("The HTTP request failed with error", err.Error())
		_, etcdErr := connection.ConnectEtcd.Put(context.Background(), commitConfigKeys, "-1")
		if etcdErr != nil {
			logs.LogConf.Error("error while updating etcd", etcdErr)
		}
		err = promutil.CounterAdd("cim_config_push_failure_total", 1, map[string]string{"pod": types.PodID})
		if err != nil {
			log.Println("error while adding values in  counter  for total push config failure", err)
		}
		return
	}

	err = promutil.CounterAdd("cim_config_push_success_total", 1, map[string]string{"pod": types.PodID})
	if err != nil {
		log.Println("error while adding values in  counter  for total push config", err)
	}
	logs.LogConf.Info("http request succeeded with status code ", response.Status)
}

func UpdateConfigCommitDetails(commitDetails map[string]string, cId int, cmType ...string) (err error) {
	configKeys := strings.Split(commitDetails["change-set-key"], "/")
	commitKey := configKeys[0:5]
	if cmType[0] == "SELF" && kubernetes.ConfigMapExist() {
		commitKey = append(commitKey, "cim")
	}
	commitKey = append(commitKey, types.PodID)
	commitKey[0] = "commit-config"
	commitConfigKeys := strings.Join(commitKey, "/")
	logs.LogConf.Audit(1, cId, "Update commit details to etcd", commitConfigKeys, commitDetails["revision"])
	logs.LogConf.Debug("Commit Details", commitDetails, "Executed on", cmType)
	var etcd_put_resp *clientv3.PutResponse
	rev := ""
	if commitDetails["status"] == "success" {
		err = promutil.CounterAdd("cim_config_resp_success_total", 1, map[string]string{"pod": types.PodID})
		if err != nil {
			log.Println("Error while adding values in  counter  for total config response", err)
		}
		rev = commitDetails["revision"]
		logs.LogConf.Info("config patch is success")
		notify_util.RaiseEvent("CimConfigUpdateSuccess", []string{"Description", "STATUS"}, []string{"Config update success", "SUCCESS"}, []string{"Commit_Keys"}, []string{commitConfigKeys})
		etcd_put_resp, err = connection.ConnectEtcd.Put(context.Background(), commitConfigKeys, rev)
	} else if commitDetails["status"] == "unused" {
		err = promutil.CounterAdd("cim_config_resp_unused_total", 1, map[string]string{"pod": types.PodID})
		if err != nil {
			logs.LogConf.Error("Error while adding values in  counter  for total config response unused", err.Error())
		}
		rev = "-2"
		logs.LogConf.Info("config patch is unused")
		notify_util.RaiseEvent("CimConfigUpdateUnused", []string{"Update_Config_Status"}, []string{"UNUSED"}, []string{"Error", "Action"}, []string{"Config patch unused", "Check Patch Path and Patch Version"})
		etcd_put_resp, err = connection.ConnectEtcd.Put(context.Background(), commitConfigKeys, rev)
	} else {
		err = promutil.CounterAdd("cim_config_resp_failure_total", 1, map[string]string{"pod": types.PodID})
		if err != nil {
			log.Println("Error while adding values in  counter  for total config response failure", err)
		}
		rev = "-1"
		logs.LogConf.Error("config patch is failed, apply the config changes manually")
		notify_util.RaiseEvent("CimConfigUpdateFailure", []string{"Update_Config_Status"}, []string{"FAIL"}, []string{"Error", "Action"}, []string{"Config patch failed", "Update Changes Manually"})
		etcd_put_resp, err = connection.ConnectEtcd.Put(context.Background(), commitConfigKeys, rev)
	}
	logs.LogConf.Debug("Etcd Put Response", etcd_put_resp.Header.Revision, etcd_put_resp.OpResponse(), etcd_put_resp.PrevKv)
	if err != nil {
		logs.LogConf.Audit(1, cId, "Commit config push etcd failure", commitConfigKeys, rev)
		err_pr := promutil.CounterAdd("cim_config_push_failure_total", 1, map[string]string{"pod": types.PodID})
		if err_pr != nil {
			log.Println("Error while adding values in  counter  for commit push config failure", err_pr)
		}
		notify_util.RaiseEvent("CimCommitConfigFailure", []string{"Update_Etcd_Commit_Status", "Commit_Keys"}, []string{"FAIL", commitConfigKeys}, []string{"Error"}, []string{err.Error()})
	} else {
		logs.LogConf.Audit(1, cId, "Commit config push etcd success", commitConfigKeys, rev)
		notify_util.RaiseEvent("CimCommitConfigSuccess", []string{"Update_Etcd_Commit_Status"}, []string{"SUCCESS"}, []string{"Commit_Keys"}, []string{commitConfigKeys})
	}
	return
}

//WatchDayOneConfig for configuration
func WatchDayOneConfig() {
	DayOneConfig = "config/" + types.Namespace + "/" + types.MicroserviceName
	watchChan := connection.ConnectEtcd.Watch(context.Background(), DayOneConfig, clientv3.WithPrefix())
	if watchChan == nil {
		logs.LogConf.Error("watch channel is nill")
		return
	}
	for watchResp := range watchChan {
		for _, event := range watchResp.Events {
			eventType := fmt.Sprintf("%s", event.Type)
			if eventType == "PUT" {
				if AppConfigured {
					logs.LogConf.Info("Stopping Day1 Config Watcher as the app is already configured")
					return
				}

				//update nf config revision
				keys := strings.Split(string(event.Kv.Key), "/")
				if len(keys) > 3 {
					types.NfConfigVersion = keys[3]
				}

				if InitialNotConfiguredEvent && types.NfConfigVersion == "0" {
					logs.LogConf.Info("Recieved a Day1 config load event. Forwarding to application.")
					configLoader(event.Kv.Value)
				} else if !InitialNotConfiguredEvent {
					logs.LogConf.Info("Recieved a Day1 config load, not forwarding to app as not yet recieved NotConfigured event")
				} else {
					logs.LogConf.Info("Stopping Day1 Config Watcher")
					return
				}
			}
		}
	}

}

func configLoader(data []byte) error {
	var (
		appPort        = types.AppPort
		configJsonData = make(map[string]interface{})
	)
	if appPort == "" {
		appPort = "9999"
	}

	err := json.Unmarshal(data, &configJsonData)
	if err != nil {
		logs.LogConf.Info("error while unmarshalling data day1 config", err)
		notify_util.RaiseEvent("LoadConfigRequestFailure", []string{"Load_Config_Request"}, []string{"FAIL"}, []string{"Error", "Action"}, []string{"Config load failed", "Load config failed"})
		//restart pod
		kubernetes.KubeConnect.DeletePod(types.PodID, types.Namespace)
		return err
	}

	logs.LogConf.Info("recieved full config details are:", configJsonData)

	data, err = json.Marshal(configJsonData)
	if err != nil {
		logs.LogConf.Info("error while marshalling data day1 config", err)
		notify_util.RaiseEvent("LoadConfigRequestFailure", []string{"Load_Config_Request"}, []string{"FAIL"}, []string{"Error", "Action"}, []string{"Config load failed", "Load config failed"})
		//restart pod
		kubernetes.KubeConnect.DeletePod(types.PodID, types.Namespace)
		return err
	}

	response, err := httpUtility.HttpPost("http://localhost:"+appPort+"/api/v1/_operations/loadConfig", "application/json", data)
	if err != nil {
		logs.LogConf.Error("The HTTP request failed with error", err.Error())
		notify_util.RaiseEvent("LoadConfigRequestFailure", []string{"Load_Config_Request"}, []string{"FAIL"}, []string{"Error", "Action"}, []string{"Config load failed", "Load Changes Manually"})
		//restart pod
		kubernetes.KubeConnect.DeletePod(types.PodID, types.Namespace)
		return err
	}

	if response.StatusCode != http.StatusOK && response.StatusCode >= http.StatusBadRequest {
		logs.LogConf.Error("configLoader failed, error response from the application ", response.StatusCode)
		notify_util.RaiseEvent("LoadConfigRequestFailure", []string{"Load_Config_Request"}, []string{"FAIL"}, []string{"Error", "Action"}, []string{"Config load failed", "Load Changes Manually"})
		//restart pod
		kubernetes.KubeConnect.DeletePod(types.PodID, types.Namespace)
		return err
	}

	logs.LogConf.Info("config loaded to  application with status code ", response.Status)
	return nil
}

func getDayOneConfig() {
	configKey := DayOneConfig + "/" + types.NfConfigVersion
	response, err := connection.ConnectEtcd.Get(context.Background(), configKey)
	if err != nil {
		logs.LogConf.Error("etcdGet: error while getting the configKey ", configKey, err)
		return
	}

	if response == nil || len(response.Kvs) == 0 {
		logs.LogConf.Info("getDayOneConfig: no config details present in etcd, wait for day1 config update")
		return
	}

	logs.LogConf.Info("config details from config key ", configKey, response.Kvs, string(response.Kvs[0].Value))
	err = configLoader(response.Kvs[0].Value)
	if err != nil {
		logs.LogConf.Error("error while loading day1 configs to app", err)
	}
}
