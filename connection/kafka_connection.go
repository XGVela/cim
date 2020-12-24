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

package connection

import (
	"cim/logs"
	"cim/types"
	"strconv"
	"time"

	"github.com/segmentio/kafka-go"
	//"github.com/segmentio/kafka-go/snappy"
)

const (
	LastOffset  int64 = -1
	FirstOffset int64 = -2
)

var (
	KafkaConn *kafka.Conn
)

func Configure(kafkaBrokerUrls []string, clientId string, topic string) (w *kafka.Writer, err error) {
	dialer := &kafka.Dialer{
		Timeout:  10 * time.Second,
		ClientID: clientId,
	}

	config := kafka.WriterConfig{
		Brokers:      kafkaBrokerUrls,
		Topic:        topic,
		Balancer:     &kafka.Hash{},
		Dialer:       dialer,
		BatchTimeout: 100 * time.Nanosecond,
		Async:        false,
		BatchSize:    1000,
	}
	w = kafka.NewWriter(config)
	return w, nil
}
func tryKafkaConn() (err error) {
	dialer := &kafka.Dialer{
		Timeout:  10 * time.Second,
		ClientID: types.CimConfigObj.CimConfig.App.KafkaClientID,
	}

	for _, broker := range types.CommonInfraConfigObj.KafkaBrokers {
		KafkaConn, err = dialer.Dial("tcp", broker)
		if err != nil {
			continue
		} else {
			break
		}
	}
	return
}

func ConnKafka() (err error) {
	var i int
	for i = 0; i < types.ConfigDetails.RemoteSvcRetryCount; i++ {
		err = tryKafkaConn()
		if err != nil {
			logs.LogConf.Error("Error while dialing kafka. Retrying...")
			time.Sleep(1 * time.Second)
			continue
		} else {
			logs.LogConf.Info("Kafka connection successfull")
			break
		}
	}
	if err != nil {
		// cannot raise event because kafka itself not available
		logs.LogConf.Exception("Unable to connect kafka even after", i, "reiterations", "Terminanting CIM", err)
	}
	return
}

//GetKafkaReader reader object for consumption
func GetKafkaReader(topic, consumerGroupName string, offsetMode int64) *kafka.Reader {
	readerConf := kafka.ReaderConfig{
		Brokers:     types.CommonInfraConfigObj.KafkaBrokers,
		Topic:       topic,
		MinBytes:    10e3, // 10KB
		MaxBytes:    10e6, // 10MB
		GroupID:     consumerGroupName,
		StartOffset: offsetMode,
	}
	return kafka.NewReader(readerConf)
}

// CreateKafkaTopicIfNotExist will create kafka topic if not exist
// If Topic exist Already it will not effect existing one
func CreateKafkaTopic(topicName string, numPartitions int, replicationFactor int, retentionTime string) (err error) {

	getCurrentControllerAndCreateKafkaConn()

	topicConf := kafka.TopicConfig{
		Topic:             topicName,
		NumPartitions:     numPartitions,
		ReplicationFactor: replicationFactor,
	}
	if retentionTime != "" {
		confEntries := kafka.ConfigEntry{
			ConfigName:  "retention.ms",
			ConfigValue: retentionTime,
		}
		var topicConfConfigEntries []kafka.ConfigEntry
		topicConfConfigEntries = append(topicConfConfigEntries, confEntries)
		topicConf.ConfigEntries = topicConfConfigEntries
	}
	err = KafkaConn.CreateTopics(topicConf)
	if err != nil {
		logs.LogConf.Error("Error while creating topic", topicName, err.Error())
		return
	}
	logs.LogConf.Info("Topic Created with", topicName)
	return
}
func getCurrentControllerAndCreateKafkaConn() {
	brokers, err := KafkaConn.Controller()
	if err != nil {
		logs.LogConf.Error("Error while requesting for controller", err.Error())
		return
	}
	logs.LogConf.Info("Current kafka Controller", "host", brokers.Host, "port", brokers.Port, "id", brokers.ID, "rack", brokers.Rack)
	dialer := &kafka.Dialer{
		Timeout:  10 * time.Second,
		ClientID: types.CimConfigObj.CimConfig.App.KafkaClientID,
	}

	KafkaConn, err = dialer.Dial("tcp", brokers.Host+":"+strconv.Itoa(brokers.Port))
	if err != nil {
		logs.LogConf.Error("Error while dialing kafka", err.Error())
	}
}
