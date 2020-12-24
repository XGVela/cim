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

package logs

import (
	"cim/types"
	"cim/utility"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"
	"time"

	fbLog "cim/flatbuf-data/LogInterface"

	"cim/utility/lumberjack.v2"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	fb "github.com/google/flatbuffers/go"
)

// More loggers can be added if required
// For more dynamism it can even be initialized from main.
var (
	AUDIT_LOG_HEAD = "ACTIVITY_CIM_"
	LogService     = "ACTIVITY_CIM_"
	AUDIT_LOG_Func = map[int]string{
		1: "ConfigUpdate",
	}
	ContainerID string
)

type LogLevel int

const (
	LogLevelAll LogLevel = iota
	LogLevelDebug
	LogLevelInfo
	LogLevelWarning
	LogLevelError
	LogLevelException
)

type LogConfiguration struct {
	StandardOutput bool     // true to print o/p on console. by dafault false. No console print
	LogFile        bool     // true to print to file. by default false.
	DateFormat     string   //  date format. Check time package for info
	TimeFormat     string   // Time format. Check time package for info
	TimeLocation   string   // Time and Date Location. time package ex: Asia/Kolkata
	Fd             *os.File // For file operations
	PrintException bool     // If true exception type will be printed. else not
	PrintError     bool     // If true error type will be printed. else not
	PrintWarning   bool     // If true Warning type will be printed. else not
	PrintInfo      bool     // If true Info  type will be printed. else not
	PrintDebug     bool     // If true Debug type will be printed. else not
}

var LogConf LogConfiguration

func init() {
	setLogLevel("ALL")
}

// getLogLevelEnum takes log level as string and returns LogLevel enum
func getLogLevelEnum(level string) LogLevel {
	levelMap := map[string]LogLevel{
		"ALL":       LogLevelAll,
		"DEBUG":     LogLevelDebug,
		"INFO":      LogLevelInfo,
		"WARNING":   LogLevelWarning,
		"ERROR":     LogLevelError,
		"EXCEPTION": LogLevelException,
	}
	if _, ok := levelMap[level]; !ok {
		return LogLevelAll
	}
	return levelMap[level]
}

var lJack *lumberjack.Logger

func NewLogger(fileName string) *lumberjack.Logger {
	var conf types.Lmaas = *types.CimConfigObj.CimConfig.Lmaas
	l := &lumberjack.Logger{
		Filename:   fileName,
		MaxSize:    conf.MaxFileSizeInMB, // megabytes
		MaxBackups: conf.MaxBackupFiles,
		MaxAge:     conf.MaxAge, //days
		Compress:   false,       // disabled by default
	}
	return l
}

func InitializeLogInfo(logLevel string, file_log bool) {
	LogConf.DateFormat = "YY-MM-DD"
	LogConf.TimeFormat = "SS-MM-HH-MIC"
	LogConf.TimeLocation = ""

	setLogLevel(logLevel)
	GetContainerID()
	ContainerID = os.Getenv("K8S_CONTAINER_ID")
	types.CurrentCimState = types.CimLogInitialized
}

func setLogLevel(logLevel string) {
	logLevelEnum := getLogLevelEnum(logLevel)
	if logLevelEnum <= LogLevelException {
		LogConf.PrintException = true
	}

	if logLevelEnum <= LogLevelError {
		LogConf.PrintError = true
	}

	if logLevelEnum <= LogLevelWarning {
		LogConf.PrintWarning = true
	}

	if logLevelEnum <= LogLevelInfo {
		LogConf.PrintInfo = true
	}

	if logLevelEnum <= LogLevelDebug {
		LogConf.PrintDebug = true
	}
}

func (lc *LogConfiguration) Exception(arg ...interface{}) {
	if lc.PrintException {
		lc.createLogEntry("EXCEPTION", arg)
	}
	time.Sleep(2 * time.Second)
	defer lc.Fd.Close()

	if types.CurrentCimState >= types.CimStateRegistered {
		podID := os.Getenv("K8S_POD_ID")
		namespace := os.Getenv("K8S_NAMESPACE")
		fmt.Println("fetched podID and namespace")
		var kubeConnect *KubeClient
		var err error
		if kubeConnect == nil {
			kubeConnect, err = NewKubeConfig()
			if err != nil {
				lc.createLogEntry("EXCEPTION", "Error while connecting to k8s client while restarting the pod:", podID, ",while handling an exception. Error:", err)
				os.Exit(1)
			}
		}
		fmt.Println("connected to kube client")
		err = kubeConnect.Client.CoreV1().Pods(namespace).Delete(podID, &metav1.DeleteOptions{})
		if err != nil {
			lc.createLogEntry("EXCEPTION", "Error restarting the pod:", podID, ",while handling an exception. Error:", err)
			os.Exit(1)
		}

		lc.createLogEntry("INFO", "Restarted pod:", podID, "because of an exception")
	} else {
		os.Exit(1)
	}
}

func (lc *LogConfiguration) Error(arg ...interface{}) {
	if !LogConf.PrintError {
		return
	}
	lc.createLogEntry("ERROR", arg)
}

func (lc *LogConfiguration) Warning(arg ...interface{}) {
	if !LogConf.PrintWarning {
		return
	}
	lc.createLogEntry("WARNING", arg)
}

func (lc *LogConfiguration) Info(arg ...interface{}) {
	if !LogConf.PrintInfo {
		return
	}
	lc.createLogEntry("INFO", arg)
}

func (lc *LogConfiguration) Debug(arg ...interface{}) {
	if !LogConf.PrintDebug {
		return
	}
	lc.createLogEntry("DEBUG", arg)
}

func createNatsLogMessage(payload string) (outmsg []byte) {

	builder := fb.NewBuilder(0)

	cntId := builder.CreateString(ContainerID)
	cntName := builder.CreateString("cim")
	fbPayload := builder.CreateString(payload)

	fbLog.LogMessageStart(builder)
	fbLog.LogMessageAddContainerId(builder, cntId)
	fbLog.LogMessageAddContainerName(builder, cntName)
	fbLog.LogMessageAddPayload(builder, fbPayload)
	logMsg := fbLog.LogMessageEnd(builder)

	builder.Finish(logMsg)

	return builder.FinishedBytes()
}

func (lc *LogConfiguration) publishLogToNats(msg []byte) error {
	payload := createNatsLogMessage(string(msg))
	return types.NatsConnection.Publish("LOG", payload)
}

func (lc *LogConfiguration) createLogEntry(level string, arg ...interface{}) {
	ts, _ := utility.Get(lc.DateFormat, lc.TimeFormat, lc.TimeLocation)

	if !types.LOGTopicSubscribed || types.CurrentCimState < types.CimLogInitialized {
		fmt.Printf("["+level+"] : "+ts+" -> %v\n", arg...)
		return
	}

	err := lc.publishLogToNats([]byte(fmt.Sprintf("["+level+"] : "+ts+" -> %v\n", arg...)))
	if err != nil {
		fmt.Println("Error occured while publishing log to nats.", err)
		return
	}
}

func (lc *LogConfiguration) Audit(activity_functionality int, arg ...interface{}) {
	lc.createLogEntry(AUDIT_LOG_HEAD+AUDIT_LOG_Func[activity_functionality], arg)
}

//GetContainerID get containerid from script
func GetContainerID() {
	out, err := exec.Command("/bin/bash", "-c", "/opt/bin/get_containerid.sh").Output()
	if err != nil {
		fmt.Println("error while getting the container id from get_containerid.sh", err)
		types.ContainerID = "0000"
		os.Setenv("K8S_CONTAINER_ID", types.ContainerID)
		return
	}
	log.Println("Container ID is :", string(out))
	types.ContainerID = strings.TrimSpace(string(out))
	os.Setenv("K8S_CONTAINER_ID", types.ContainerID)

}
