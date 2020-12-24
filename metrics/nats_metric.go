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

package metrics

import (
	"cim/logs"
	"cim/promutil"
	"cim/types"
	"strings"

	"github.com/prometheus/client_golang/prometheus"
)

var natsMetricInitialized = false

// NatsMetricInit initialize various metrics for different NATS topics
func NatsMetricInit() {
	logs.LogConf.Info("Initializing NATS metrics")
	var err error
	for subject, subscribed := range types.NatsSubscribedSubj {
		subject = strings.ReplaceAll(subject, ".", "_")
		if subscribed {
			err = promutil.CreateCounter(promutil.CounterOpts{
				Name:   "cim_nats_" + subject + "_message_count",
				Help:   "Count of number of messages for topic " + subject,
				Labels: []string{"pod"},
			})
			if err != nil {
				logs.LogConf.Error(
					"error while creating counter",
					"cim_nats_"+subject+"_message_count",
					err,
				)
				continue
			}

			err = promutil.CreateGauge(promutil.GaugeOpts{
				Name:   "cim_nats_" + subject + "_latency",
				Help:   "Latency to process messages for topic " + subject,
				Labels: []string{"pod"},
			})
			if err != nil {
				logs.LogConf.Error(
					"error while creating gauge",
					"cim_nats_"+subject+"_latency",
					err,
				)
				continue
			}

			err = promutil.CreateHistogram(promutil.HistogramOpts{
				Name:    "cim_nats_" + subject + "_message_size",
				Help:    "Message size for topic " + subject,
				Labels:  []string{"pod"},
				Buckets: prometheus.LinearBuckets(0, 500, 20),
			})
			if err != nil {
				logs.LogConf.Error(
					"error while creating histogram",
					"cim_nats_"+subject+"_message_size",
					err,
				)
				continue
			}
		}
	}
	natsMetricInitialized = true
}

// NatsIncrementMessageCount increments the count of message for a given NATS subject
func NatsIncrementMessageCount(subject string) {
	subject = strings.ReplaceAll(subject, ".", "_")
	if !natsMetricInitialized {
		return
	}
	counterName := "cim_nats_" + subject + "_message_count"
	err := promutil.CounterAdd(
		counterName,
		1,
		map[string]string{"pod": types.PodID},
	)
	if err != nil {
		logs.LogConf.Error("error while incrementing counter:", counterName, err)
		return
	}
}

// NatsUpdateMessageLatency updates the message latency in μs for a given NATS subject
func NatsUpdateMessageLatency(subject string, latency int64) {
	subject = strings.ReplaceAll(subject, ".", "_")
	if !natsMetricInitialized {
		return
	}
	gaugeName := "cim_nats_" + subject + "_latency"
	err := promutil.GaugeSet(
		gaugeName,
		float64(latency),
		map[string]string{"pod": types.PodID},
	)
	if err != nil {
		logs.LogConf.Error("error while updating message latency:", gaugeName, err)
		return
	}
}

// NatsUpdateMessageSizeHistogram updates the message size histogram for a given NATS subject
func NatsUpdateMessageSizeHistogram(subject string, messageSize int) {
	subject = strings.ReplaceAll(subject, ".", "_")
	if !natsMetricInitialized {
		return
	}
	histogramName := "cim_nats_" + subject + "_message_size"
	err := promutil.HistogramObserve(
		histogramName,
		float64(messageSize),
		map[string]string{"pod": types.PodID},
	)
	if err != nil {
		logs.LogConf.Error("error while updating message size histogram:", histogramName, err)
		return
	}
}
