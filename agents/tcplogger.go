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
	"cim/kubernetes"
	"cim/logs"
	notify_util "cim/notification"
	"cim/promutil"
	"cim/types"
	"errors"
	"net"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
)

var (
	TCPcon              *net.TCPConn
	LogConnExist        bool
	waitDuration        time.Duration
	EnableDaemonsetPush bool
	loggingSvcUrl       string
	fluentPodListenPort string
	nodeName            string
	EnableTCPLogging    bool
	once                = new(sync.Once)
	Address             string
)

// Description     : Function writes logs to fluent bit over tcp interface,
//                   if function writes succesfully to fluentbit it will return
//                   true else it will return false.
// Input param     : Logs recieved by cim using nats in bytes
// Return param    : bool

func ValidateSvcUrl(url string) bool {
	RegExp := regexp.MustCompile(`^(([a-zA-Z]{1})|([a-zA-Z]{1}[a-zA-Z]{1})|([a-zA-Z]{1}[0-9]{1})|([0-9]{1}[a-zA-Z]{1})|([a-zA-Z0-9][a-zA-Z0-9-_]{1,61}[a-zA-Z0-9]))\.([a-zA-Z]{2,6}|[a-zA-Z0-9-]{2,30}\.svc.cluster.local:[0-9]{1,6})$`)
	valid := RegExp.MatchString(url)
	if !valid {
		return false
	} else {
		ar := regexp.MustCompile("[\\:\\s]+").Split(url, -1)
		if ValidatePort(ar[1]) {
			return true
		} else {
			return false
		}
	}

}

func ValidatePort(port string) bool {
	i, err := strconv.ParseUint(port, 10, 16)
	if err != nil || i == 0 || i == 65535 {
		return false
	}
	return true
}

func GetLoggingListenPort(port string) string {
	if ValidatePort(port) {
		return port
	} else {
		EnableTCPLogging = false
		logs.LogConf.Error("Invalid logging_svc_tcp_port configuration in cim.json", time.Now())
		logs.LogConf.Info("CIM fallback to file mode", time.Now())
		return "5170"
	}

}

func GetLoggingSvc(svc string) string {
	if ValidateSvcUrl(svc) {
		return svc
	} else {
		EnableTCPLogging = false
		logs.LogConf.Error("Invalid logging_svc_url configuration in cim.json", time.Now())
		logs.LogConf.Info("CIM fallback to file mode", time.Now())
		return ""
	}
}

func InitialiseTCPLoggingParams() {
	if types.CimConfigObj.CimConfig.Lmaas.LoggingMode == "TCP" {
		EnableTCPLogging = true
	} else {
		EnableTCPLogging = false
		return
	}
	EnableDaemonsetPush = types.CimConfigObj.CimConfig.Lmaas.PreferLocalDeamonset
	if !EnableDaemonsetPush {
		loggingSvcUrl = GetLoggingSvc(types.CommonInfraConfigObj.LoggingSvcUrl)
	} else {
		fluentPodListenPort = GetLoggingListenPort(types.CommonInfraConfigObj.LoggingSvcTcpPort)
		//nodeName = os.Getenv("NODENAME")
		nodeName = types.PodDetails.Spec.NodeName
		if nodeName == "" {
			EnableTCPLogging = false
			logs.LogConf.Error("NodeName not found in pod details, falling back to file mode.")
		}
	}
}

func getValidIpFormat(ip string) string {
	// checking if it is ipv6
	if strings.Contains(ip, ":") {
		return "[" + ip + "]"
	}
	return ip

}

func GetLoggingEndpoint() (string, error) {
	var endpoint string
	if EnableDaemonsetPush {
		pods, err := kubernetes.KubeConnect.GetLoggingPod(nodeName)
		if err != nil {
			logs.LogConf.Error("Getting error while finding the pod", err.Error, time.Now())
			return "", err
		}
		if pods != nil && len(pods.Items) != 0 {
			ip := getValidIpFormat(pods.Items[0].Status.PodIP)
			endpoint = ip + ":" + fluentPodListenPort
		} else {
			once.Do(func() {
				logs.LogConf.Error("Unable to find Logging local daemonset, falling back to file mode.")
			})
			return "", errors.New("unable to find Logging local daemonset")
		}
	} else {
		svcURL := strings.Split(loggingSvcUrl, ":")
		addr, err := net.LookupIP(svcURL[0])
		if err != nil {
			logs.LogConf.Error("Unable to resolve", svcURL[0])
			return "", err
		} else {
			endpoint = addr[0].String() + ":" + svcURL[1]
		}
	}
	once = new(sync.Once)
	return endpoint, nil

}

func updateMetricsOnSendingLogToTCP(n float64) {
	err := promutil.CounterAdd("Total_msg_sent_to_tcp", 1, map[string]string{
		"pod": types.PodID})

	if err != nil {
		logs.LogConf.Error("error while adding values in  counter  for Total_msg_sent_to_tcp", err)
		return
	}
}

func updateMetricsOnWritingLogToFile(n float64) {
	err := promutil.CounterAdd("Total_msg_write_to_file", 1, map[string]string{
		"pod": types.PodID})

	if err != nil {
		logs.LogConf.Error("error while adding values in  counter  for Total_msg_write_to_file", err)
		return
	}
}

func updateMetricsOnTCPDisconnect(n float64) {
	err := promutil.CounterAdd("tcp_disconnect_count", 1, map[string]string{
		"pod": types.PodID})

	if err != nil {
		logs.LogConf.Error("error while adding values in  counter  for Total_msg_write_to_file", err)
		return
	}
}

func updateMetricsOnNewMsg(n float64) {
	err := promutil.CounterAdd("Total_msg_count", 1, map[string]string{
		"pod": types.PodID})

	if err != nil {
		logs.LogConf.Error("error while adding values in  counter  for Total_msg_count", err)
		return
	}
}

func monitorTCPConnection(address string) {
	for LogConnExist {
		conn, err := net.DialTimeout("tcp", address, 1*time.Second)
		if err != nil {
			logs.LogConf.Error("Failure while monitoring TCP connection with fluentbit")
			LogConnExist = false
			TCPcon.Close()
			notify_util.RaiseEvent("LmaasTCPFailure",
				[]string{"Description", "STATUS"},
				[]string{"TCP connection with fluentbit is DOWN", "FAILURE"},
				[]string{"ACTION_IN_PROGRESS "},
				[]string{"CIM writing logs to the file"})
			go CreateConnection()
			return
		}
		conn.Close()
		time.Sleep(1 * time.Second)
	}
}

func GetConnection() (net.Conn, error) {
	c, err := net.DialTimeout("tcp", Address, 1*time.Second)
	if err != nil {
		return nil, err
	}
	return c, nil
}

func CreateConnection() {
	waitDuration, _ = time.ParseDuration("5s")
	LogConnExist = false
	var enableLogFlag bool = true
	logs.LogConf.Info("Trying to establish connection with fluentbit using TCP interface.", time.Now())
	for LogConnExist == false {
		Address, err = GetLoggingEndpoint()
		if err != nil {
			time.Sleep(waitDuration)
			continue
		}
		c, err := GetConnection()
		if err != nil {
			if enableLogFlag {
				logs.LogConf.Warning("CIM unable to make tcp connection, falling back to file mode,Error:", err.Error())
				enableLogFlag = false
			}
			time.Sleep(waitDuration)
			continue
		}
		TCPcon = c.(*net.TCPConn)
		LogConnExist = true
		/*
		  If TCP connection created watch DIR if logs files exist it will retransmit,
		*/
		go WatchLogDirectory()
		go monitorTCPConnection(Address)

		logs.LogConf.Info("CIM succesfully established connection with fluentbit using TCP interface", Address, time.Now())

	}
}

func writeTCP(b []byte, containerName string) bool {
	var err error
	var payload []byte
	if !EnableTCPLogging || !LogConnExist || TCPcon == nil {
		return false
	}

	// Some times 2048 packet size taking maximum 5.1s for writing to TCP interface
	TCPcon.SetWriteDeadline(time.Now().Add(6 * time.Second))
	if LogFormatJson {
		payload = CreateJsonHeader(b, containerName)
	} else {
		payload = append(b, byte('\n'))
	}
	_, err = TCPcon.Write(payload)

	if err != nil {
		logs.LogConf.Debug("error in writing to TCPcon in writeTCP", err.Error())
		notify_util.RaiseEvent("LmaasTCPFailure",
			[]string{"Description", "STATUS"},
			[]string{"TCP connection with fluentbit is DOWN", "FAILURE"},
			[]string{"ACTION_IN_PROGRESS "},
			[]string{"CIM writing logs to the file"})
		updateMetricsOnTCPDisconnect(1)
		LogConnExist = false
		go CreateConnection()
		return false
	}

	return true

}
