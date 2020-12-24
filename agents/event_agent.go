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
	eventDetails "cim/flatbuf-data/EventInterface"
	"cim/kubernetes"
	"cim/logs"
	"cim/metrics"
	"cim/types"
	"cim/utility"
	"encoding/json"
	"log"
	"os"
	"strings"

	// "cim/notification"
	"context"

	// TODO: Fix import issues here
	"github.com/nats-io/go-nats"
	"github.com/segmentio/kafka-go"

	//"natsSubFb/utility"
	"time"
)

var (
	//inc                 = 0
	startTime                 = time.Now()
	KafkaPublisherEvent       *KafkaPublisher
	srcDetails                *utility.TmaaSAnnotations
	err                       error
	AppConfigured             bool
	InitialNotConfiguredEvent bool
)

//KafkaPublisher struct
type KafkaPublisher struct {
	Context   context.Context
	Subject   string
	Namespace string
	PodID     string
	Writer    *kafka.Writer
	//MsgCount  int
}

//NewKafkaPublisher initialize
func NewKafkaPublisher(subject string, w *kafka.Writer) *KafkaPublisher {
	subscription := &KafkaPublisher{
		Subject:   subject,
		Namespace: types.Namespace,
		PodID:     types.PodID,
		Writer:    w,
		Context:   context.Background(),
		//MsgCount:  inputDetails.MsgCount,
	}
	return subscription
}

//Name returns the name of handler
func (k KafkaPublisher) Name() string {
	return k.Subject
}

//HandleMessages handles messages in the NATS
func (k KafkaPublisher) HandleMessages(msg *nats.Msg) {
	metrics.NatsIncrementMessageCount(msg.Subject)
	metrics.NatsUpdateMessageSizeHistogram(msg.Subject, len(msg.Data))

	var err error
	logs.LogConf.Info("NATS Handling message from subject:", msg.Subject)
	if msg.Subject == "EVENT" {
		logs.LogConf.Info("Received EVENT data")
		eventMsg := eventDetails.GetRootAsEvent(msg.Data, 0)

		latency := time.Now().UnixNano()/int64(time.Millisecond) - eventMsg.EventTime()
		metrics.NatsUpdateMessageLatency(msg.Subject, latency)

		if !AppConfigured {
			if string(eventMsg.EventName()[:]) == "Configured" {
				logs.LogConf.Info("Configured Event recieved from application.")
				// notify_util.RaiseEvent("CONFIGURED", []string{"Description"}, []string{"Configured Event recieved from application"}, []string{"STATUS"}, []string{"SUCCESS"})
				AppConfigured = true
			} else if string(eventMsg.EventName()[:]) == "NotConfigured" {
				if !InitialNotConfiguredEvent {
					InitialNotConfiguredEvent = true
					//get config values from etcd
					go getDayOneConfig()
				}
				logs.LogConf.Info("Configuration not yet done on application side.")
				// notify_util.RaiseEvent("NOTCONFIGURED", []string{"Description"}, []string{"NotConfigured Event recieved from application"}, []string{"STATUS"}, []string{"FAILURE"})
			}
		}

		var appShutdownCause string
		if len(types.AppShutDownCause) != 0 {
			appShutdownCause = types.AppShutDownCause[0]
		}
		if string(eventMsg.EventName()[:]) == "ShutdownSuccess" || string(eventMsg.EventName()[:]) == "ShutdownFailure" {
			logs.LogConf.Info("Shutdown event received from the application")
			if !types.AppLogsFlushed && appShutdownCause == types.AppShutDownCauseCimTermination {
				types.FlushAppLogs <- true
			}
		}

		var (
			managedDetails        []types.KeyValueObjects
			additionalInfoDetails []types.KeyValueObjects
			stateChangeDefinition []types.KeyValueObjects
			managedObj            = new(eventDetails.KeyValue)
		)

		for i := 0; i < eventMsg.ManagedObjectLength(); i++ {
			var m types.KeyValueObjects
			eventMsg.ManagedObject(managedObj, i)
			m.Name = string(managedObj.Key())
			m.Value = string(managedObj.Value())
			managedDetails = append(managedDetails, m)

		}
		for i := 0; i < eventMsg.AdditionalInfoLength(); i++ {
			var n types.KeyValueObjects
			eventMsg.AdditionalInfo(managedObj, i)
			n.Name = string(managedObj.Key())
			n.Value = string(managedObj.Value())
			additionalInfoDetails = append(additionalInfoDetails, n)
		}
		for i := 0; i < eventMsg.StateChangeDefinitionLength(); i++ {
			var sc types.KeyValueObjects
			eventMsg.StateChangeDefinition(managedObj, i)
			sc.Name = string(managedObj.Key())
			sc.Value = string(managedObj.Value())
			stateChangeDefinition = append(stateChangeDefinition, sc)
		}

		//read annotation from pod template
		if srcDetails == nil {
			srcDetails, err = utility.GetAnnotations()
			if err != nil {
				log.Println("error while getting annotations", err)
				return
			}
		}
		var sourceHeadersDetails types.SourceHeaderDetails
		//check for backward compatibilty
		if srcDetails == nil {
			sourceHeadersDetails = types.SourceHeaderDetails{
				PodUUID:      kubernetes.KubeConnect.GetPodUUID(),
				UHN:          kubernetes.KubeConnect.GetLabelValue("uhn"),
				CNFCUUID:     kubernetes.KubeConnect.GetLabelValue("cnfc_uuid"),
				PodID:        types.PodID,
				Microservice: types.MicroserviceName,
				NfPrefix:     os.Getenv("NF_PREFIX"),
				Nf:           os.Getenv("NF"),
				SvcVersion:   types.AppVersion,
				NfType:       os.Getenv("NF_TYPE"),
			}
		} else {
			sourceHeadersDetails = types.SourceHeaderDetails{
				PodUUID:       kubernetes.KubeConnect.GetPodUUID(),
				UHN:           kubernetes.KubeConnect.GetLabelValue("uhn"),
				CNFCUUID:      kubernetes.KubeConnect.GetLabelValue("cnfc_uuid"),
				PodID:         types.PodID,
				Microservice:  types.MicroserviceName,
				NfPrefix:      types.PodDetails.Namespace,
				Nf:            srcDetails.NfID,
				SvcVersion:    types.AppVersion,
				NfType:        srcDetails.NfType,
				VendorID:      srcDetails.VendorID,
				XGVelaID:      srcDetails.XGVelaID,
				NfClass:       srcDetails.NfClass,
				NfID:          srcDetails.NfID,
				NfServiceID:   srcDetails.NfServiceID,
				NfServiceType: srcDetails.NfServiceType,
			}
		}

		eventjson := &types.EventDetails{
			EventName:             string(eventMsg.EventName()),
			EventSourceType:       string(eventMsg.EventSourceType()),
			EventTime:             eventMsg.EventTime(),
			ContainerId:           string(eventMsg.ContainerId()),
			ManagedObject:         managedDetails,
			AdditionalInfo:        additionalInfoDetails,
			StateChangeDefinition: stateChangeDefinition,
			SourceHeaders:         sourceHeadersDetails,
			SourceName:            string(eventMsg.SourceName()),
			SourceID:              string(eventMsg.SourceId()),
		}

		eventFinalDetails, err := json.Marshal(eventjson)
		if err != nil {
			logs.LogConf.Error("Error while unmarshalling the EVENTS data", err)
			return
		}

		logs.LogConf.Info("Events data: ", string(eventFinalDetails))
		logs.LogConf.Info(time.Now(), "Pushing to Kafka ...")
		messageKey := utility.GetMD5Hash(strings.Join(
			[]string{
				srcDetails.NfType,
				srcDetails.XGVelaID,
				srcDetails.NfID,
			},
			":",
		))
		err = Push(k.Context, []byte(messageKey), eventFinalDetails, &k, nil)
		if err != nil {
			logs.LogConf.Error(time.Now(), "err while pushing to kafka :", err)
			return
		}

		logs.LogConf.Info(time.Now(), "Push successful ...")
	}

	if err != nil {
		logs.LogConf.Error("error while push message into kafka: ", err.Error())
		return
	}
	logs.LogConf.Info("message pushed to kafka successfully")
}

//Push messages to kafka
func Push(ctx context.Context, key, value []byte, k *KafkaPublisher, header []kafka.Header) (err error) {
	message := kafka.Message{
		Key:     key,
		Value:   value,
		Time:    time.Now(),
		Headers: header,
	}
	return k.Writer.WriteMessages(ctx, message)
}
