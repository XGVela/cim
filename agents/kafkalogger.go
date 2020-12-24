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
	"cim/types"
	"time"

	"github.com/segmentio/kafka-go"

	"cim/logs"
)

// Create kafka connection when it goes down
func CheckKafkaConn() {
	logs.LogConf.Info("Checking for kafka connection")

	dialer := &kafka.Dialer{
		Timeout:  10 * time.Second,
		ClientID: types.CimConfigObj.CimConfig.App.KafkaClientID,
	}
	for {
		for _, broker := range types.CommonInfraConfigObj.KafkaBrokers {
			_, err = dialer.Dial("tcp", broker)
			if err != nil {
				continue
			}
			LogConnExist = true

			go WatchLogDirectory()
			return
		}
	}
}

// Initialise kafka producer
func IntialiseKafkaProducer() (*KafkaPublisher, error) {

	kafkaProducer, err := connection.Configure(types.CommonInfraConfigObj.KafkaBrokers, types.ConfigDetails.KafkaClientID, "LOGS")
	if err != nil {
		logs.LogConf.Error("Unable to configure Kafka", err.Error())
		return nil, err
	}
	logs.LogConf.Info("Kafka Intialised")
	LogConnExist = true

	return NewKafkaPublisher("LOGS", kafkaProducer), nil
}

// Write logs to kafka
func (k KafkaPublisher) writeToKafka(msg []byte, containerName string) bool {
	if !LogConnExist {
		return false
	}
	if LogFormatJson {
		msg = CreateJsonHeader(msg, containerName)
	}
	err = Push(k.Context, nil, msg, &k, nil)
	if err != nil {
		LogConnExist = false
		go CheckKafkaConn()
		logs.LogConf.Error("error while pushing data to kafka for topic "+k.Subject, err)
		return false
	}
	return true
}
