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
	"cim/types"
	"time"

	"cim/logs"
	"fmt"

	"context"

	"github.com/coreos/etcd/clientv3"
)

var (
	ConnectEtcd *clientv3.Client
	err         error
)

//connect to etcd
func EtcdConnect() (*clientv3.Client, error) {
	fmt.Println("etcd url ....................", types.CommonInfraConfigObj.EtcdURL)
	var i int
	for i = 0; i < types.ConfigDetails.RemoteSvcRetryCount; i++ {
		ConnectEtcd, err = clientv3.New(clientv3.Config{
			Endpoints:   []string{types.CommonInfraConfigObj.EtcdURL},
			DialTimeout: 5 * time.Second,
		})
		if err != nil {
			logs.LogConf.Error("error while etcd connection.", err.Error(), " Retrying....")
			time.Sleep(1 * time.Second)
			continue
		}
		timeoutCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_, err = ConnectEtcd.Status(timeoutCtx, types.CommonInfraConfigObj.EtcdURL)
		if err != nil {
			logs.LogConf.Error("Error while checking etcd status", err.Error())
			continue
		}
		logs.LogConf.Info("Connected to etcd successfully")
		break
	}
	if err != nil {
		logs.LogConf.Error("Unable to connect etcd even after", i, "reiterations", "Terminanting CIM")
	}
	return ConnectEtcd, err
}
