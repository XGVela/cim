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

// This package is designed to generate notifications with in cim
// Every time no need to write all code, just call Raise Event.
package notify_util

import (
	"cim/connection"
	"cim/logs"
	"cim/types"
	"cim/utility"
	"context"
	"encoding/json"
	"errors"
	"os"
	"time"

	"github.com/segmentio/kafka-go"
)

type KafkaPublisher struct {
	Context   context.Context
	Subject   string
	Namespace string
	PodID     string
	Writer    *kafka.Writer
	MsgCount  int
}

//NewKafkaPublisher initialize
func NewKafkaPublisher(inputDetails *types.UserInput, subject string, w *kafka.Writer) *KafkaPublisher {
	subscription := &KafkaPublisher{
		Subject:   subject,
		Namespace: inputDetails.Namespace,
		PodID:     inputDetails.PodID,
		Writer:    w,
		Context:   context.Background(),
		MsgCount:  inputDetails.MsgCount,
	}
	return subscription
}

var kp *KafkaPublisher
var InputDetails *types.UserInput

// This function configures kafka publisher
func InitNotificationFramework() {
	logs.LogConf.Info("Inside notification_init")
	kafkaProducer, err := connection.Configure(types.CommonInfraConfigObj.KafkaBrokers, types.ConfigDetails.KafkaClientID, "EVENT")
	if err != nil {
		logs.LogConf.Error("Unable to configure Kafka", err.Error())
		return
	}

	//defer kafkaProducer.Close()
	kp = NewKafkaPublisher(InputDetails, "EVENT", kafkaProducer)
}

//RaiseEvent is a frameworks main function to raise events.
// event name is manditory argument and rest are optional

func RaiseEvent(event_name string, manage_details_keys []string, manage_details_val []string, additional_info_key []string, additional_info_val []string) (err error) {
	if len(event_name) < 1 {
		logs.LogConf.Error("Event name is mandatory")
		err = errors.New("Mandatory parameter event name missing.")
		return
	}
	logs.LogConf.Debug("Raising Event for", event_name)
	event_byte_data, err := buildEventJsonByteData(event_name, manage_details_keys, manage_details_val, additional_info_key, additional_info_val)
	if err != nil {
		return
	}
	logs.LogConf.Info("Raising Event for", event_name, "Events data : ", string(event_byte_data))
	logs.LogConf.Info("Pushing " + event_name + " to Kafka")
	logs.LogConf.Debug("kafka subject is =======>", kp.Subject)
	err = Push(kp.Context, []byte(kp.Subject), event_byte_data, kp)
	if err != nil {
		logs.LogConf.Error("Error while pushing "+event_name+" to kafka :", err.Error())
		return
	}
	logs.LogConf.Info("Push " + event_name + " successful")
	return
}

// Push pushesh event to kafka
func Push(ctx context.Context, key, value []byte, k *KafkaPublisher) (err error) {
	message := kafka.Message{
		Key:   key,
		Value: value,
		Time:  time.Now(),
	}

	return k.Writer.WriteMessages(ctx, message)
}
func buildKeyValueObject(input_object_key []string, input_object_val []string) (key_val_obj []types.KeyValueObjects) {
	for i, val := range input_object_key {
		var obj types.KeyValueObjects
		obj.Name = val
		obj.Value = input_object_val[i]
		key_val_obj = append(key_val_obj, obj)
	}
	return
}
func getSourceHeaderDetails() (sourceHeadersDetails types.SourceHeaderDetails) {
	tmaasAnns, err := utility.GetAnnotations()
	if err != nil {
		logs.LogConf.Error("error while getting annotations", err)
	}

	sourceHeadersDetails.PodID = types.PodID
	sourceHeadersDetails.Microservice = types.MicroserviceName
	sourceHeadersDetails.NfPrefix = types.PodDetails.Namespace
	if os.Getenv("NF") != "" {
		sourceHeadersDetails.Nf = os.Getenv("NF")
	} else {
		sourceHeadersDetails.Nf = tmaasAnns.NfID
	}
	if os.Getenv("NF_TYPE") != "" {
		sourceHeadersDetails.NfType = os.Getenv("NF_TYPE")
	} else {
		sourceHeadersDetails.NfType = tmaasAnns.NfType
	}

	sourceHeadersDetails.SvcVersion = types.AppVersion
	return
}

func buildEventJsonByteData(event_name string, manage_details_keys []string, manage_details_val []string, additional_info_key []string, additional_info_val []string) (event_byte_data []byte, err error) {
	eventjson := &types.EventDetails{
		EventName:      event_name,
		EventTime:      time.Now().UnixNano() / int64(time.Millisecond),
		ContainerId:    types.ContainerID,
		ManagedObject:  buildKeyValueObject(manage_details_keys, manage_details_val),
		AdditionalInfo: buildKeyValueObject(additional_info_key, additional_info_val),
		SourceHeaders:  getSourceHeaderDetails(),
	}

	event_byte_data, err = json.Marshal(eventjson)
	if err != nil {
		logs.LogConf.Error("Error while unmarshalling the EVENTS data", err)
		return
	}
	return
}

func CreateKeyValuesInArrayFormat(key []string, val []string) (key_val_obj [][]string) {
	for i, j := range key {
		key_val_obj[i][0] = j
		key_val_obj[i][1] = val[i]
	}
	return
}
func CreateKeyValuesInArrayFormatFromMap(key_val map[string]string) (key_val_obj [][]string) {
	i := 0
	for key, val := range key_val {
		key_val_obj[i][0] = key
		key_val_obj[i][1] = val
		i++
	}
	return
}
