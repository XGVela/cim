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
	"time"

	"github.com/nats-io/go-nats"
)

//NewConnection nats connection
func NewConnection(url string, opts ...nats.Option) (*nats.Conn, error) {
	nc, err := nats.Connect(url, opts...)
	return nc, err
}

//SetupConnOptions setup connection options
func SetupConnOptions(opts []nats.Option) []nats.Option {
	totalWait := 10 * time.Minute
	reconnectDelay := time.Second

	opts = append(opts, nats.ReconnectWait(reconnectDelay))
	opts = append(opts, nats.MaxReconnects(int(totalWait/reconnectDelay)))
	opts = append(opts, nats.DisconnectHandler(func(nc *nats.Conn) {
		logs.LogConf.Debug("Disconnected: will attempt reconnects for", totalWait.Minutes())
	}))
	opts = append(opts, nats.ReconnectHandler(func(nc *nats.Conn) {
		logs.LogConf.Info("Reconnected [", nc.ConnectedUrl(), "]")
	}))
	opts = append(opts, nats.ClosedHandler(func(nc *nats.Conn) {
		logs.LogConf.Exception("Exiting, no servers available")
	}))
	return opts
}
