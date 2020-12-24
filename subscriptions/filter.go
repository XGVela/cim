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

package subscriptions

import (
	"cim/logs"
	"cim/types"
	"github.com/segmentio/kafka-go"
	"sync"
)

//KafkaSubscriptions subscription lists
var (
	KafkaSubscriptions []string
	headers            = make(map[string]string)
	lock               = sync.RWMutex{}
)

func filterMessages(message kafka.Message) {

	//parse the headers
	writeHeaders(message.Headers)

	lock.RLock()
	defer lock.RUnlock()

	exist := isNotify()
	if !exist {
		return
	}

	//send it to microservice
	NotifySubscriptionMessageToApp(message.Value)

}

func writeHeaders(messageHeaders []kafka.Header) {
	lock.Lock()
	defer lock.Unlock()
	for _, value := range messageHeaders {
		headers[value.Key] = string(value.Value)
	}
}

func isNotify() bool {
	//incase of replay just ignore the messages
	if headers["replay"] == "true" {
		return false
	}

	for _, v := range Subscriptions {
		eventNameExist := isValueInList(headers["localEventName"], v.LocalEventName)
		//eventname is not exist in subscription then do not send it app
		if !eventNameExist {
			continue
		}

		if len(v.NfNamingCode) != 0 {
			nfNamingExist := isValueInList(headers["nfNamingCode"], v.NfNamingCode)
			if !nfNamingExist {
				continue
			}
		}

		if len(v.LocalNfID) != 0 {
			localNfIDExist := isValueInList(headers["localNfID"], v.LocalNfID)
			if !localNfIDExist {
				continue
			}
		}

		if len(v.NfcNamingCode) != 0 {
			nfcNamingExist := isValueInList(headers["nfcNamingCode"], v.NfcNamingCode)
			if nfcNamingExist {
				return true
			}
			continue

		}

		return true

	}
	return false
}

func isValueInList(value string, list []string) bool {
	for _, v := range list {
		if v == value {
			return true
		}
	}
	return false
}

//NotifySubscriptionMessageToApp send messages to app
func NotifySubscriptionMessageToApp(message []byte) {

	err := types.NatsConnection.Publish("EVENT-NOTIFICATION", message)
	if err != nil {
		logs.LogConf.Error("error while publishing the event subscribed messages", err)
		return
	}

	types.NatsConnection.Flush()
	logs.LogConf.Debug("successfully published to NATs", string(message))

}
