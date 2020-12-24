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
	"cim/promutil"
	"fmt"
	"log"
)

//LogMetricInit function to register log metrics
func LogMetricInit() {

	fmt.Println("inside log_metrics.go")
	var err error

	err = promutil.CreateCounter(promutil.CounterOpts{
		Name:   "Total_msg_count",
		Help:   "Total no of messages recieved",
		Labels: []string{"pod"},
	})

	if err != nil {
		log.Println("error while creating counter metric for Total_msg_count", err)
		return
	}

	err = promutil.CreateCounter(promutil.CounterOpts{
		Name:   "tcp_disconnect_count",
		Help:   "Total no of time tcp connection get disconnected",
		Labels: []string{"pod"},
	})

	if err != nil {
		log.Println("error while creating counter metric for tcp_disconnect_count", err)
		return
	}

	err = promutil.CreateCounter(promutil.CounterOpts{
		Name:   "Total_msg_sent_to_tcp",
		Help:   "Total no of messages sent to tcp",
		Labels: []string{"pod"},
	})

	if err != nil {
		log.Println("error while creating counter metric for Total no of messages sent to tcp", err)
		return
	}

	err = promutil.CreateCounter(promutil.CounterOpts{
		Name:   "Total_msg_write_to_file",
		Help:   "Total no of messages write to file",
		Labels: []string{"pod"},
	})

	if err != nil {
		log.Println("error while creating counter metric for Total_msg_write_to_file", err)
		return
	}

	err = promutil.CreateCounter(promutil.CounterOpts{
		Name:   "cim_total_messages_counter",
		Help:   "Total no of messages recieved",
		Labels: []string{"pod"},
	})

	if err != nil {
		log.Println("error while creating counter metric for log", err)
		return
	}

	err = promutil.CreateCounter(promutil.CounterOpts{
		Name:   "cim_total_bytes_counter",
		Help:   "Total no of bytes recieved",
		Labels: []string{"pod"},
	})
	if err != nil {
		log.Println("error while creating counter metric for log bytes", err)
		return
	}
	err = promutil.CreateGauge(promutil.GaugeOpts{
		Name:   "cim_packet_size",
		Help:   "packetSize",
		Labels: []string{"pod"},
	})

	if err != nil {
		log.Println("error while creating gauge metric for log", err)
		return
	}
}
