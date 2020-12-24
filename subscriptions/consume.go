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
	"cim/connection"
	"cim/logs"
	"cim/types"
	"context"
)

//ConsumeMessages consume messages
func ConsumeMessages() {
	// connect to kafka
	//LastOffset  int64 = -1 // The most recent offset available for a partition.
	reader := connection.GetKafkaReader("FMAASEVENTS", types.PodID, connection.LastOffset)
	defer reader.Close()

	logs.LogConf.Info("start consuming events subscribed... !!")
	for {

		if !StartConsume {
			logs.LogConf.Info("no subscription exist. discontinue watching topic")
			return
		}

		message, err := reader.ReadMessage(context.Background())
		if err != nil {
			logs.LogConf.Exception(err)
		}

		//logs.LogConf.Info("message at topic:", message.Topic+"	key:"+string(message.Key)+" value:", string(message.Value))

		go filterMessages(message)
	}
}
