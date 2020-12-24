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
	"bytes"
	fbLog "cim/flatbuf-data/log_flatbuf_data"
	"cim/metrics"
	"cim/types"
	"fmt"

	"github.com/nats-io/go-nats"

	// "cim/connection"
	"cim/logs"
	// "cim/notification"
	"cim/promutil"
	"cim/utility"

	// "context"
	"encoding/json"
	"log"
	"os"

	// "strings"
	"sync"
	"time"

	//"gopkg.in/natefinch/lumberjack.v2"
	"cim/utility/lumberjack.v2"
)

type TickerSyncMap struct {
	syncMap sync.Map
}

type fileTicker struct {
	period time.Duration
	ticker *time.Timer
}

func createTicker(period time.Duration) *fileTicker {
	return &fileTicker{period, time.NewTimer(period)}
}
func (t *fileTicker) resetTicker(period time.Duration) {
	ok := t.ticker.Reset(period)
	if !ok {
		t.ticker = time.NewTimer(period)
	}
}

var (
	logBufMap      sync.Map
	flushedFileMap sync.Map

	loggerMap       sync.Map
	tickerMap       TickerSyncMap
	totalMsg        uint64
	totalByte       uint64
	enableStartTime = false
	ljFilename      string
	KafkaLogConn    *KafkaPublisher
	LogFormatJson   bool
)

type JsonLog struct {
	Containername string     `json:"container_name"`
	Log           string     `json:"log"`
	K8s           Kubernetes `json:"kubernetes"`
}

type Kubernetes struct {
	Namespace string `json:"namespace_name"`
	Podname   string `json:"pod_name"`
}

//NewLogger create a new logger obeject
func NewLogger(fileName string, conf types.Lmaas) *lumberjack.Logger {
	logs.LogConf.Debug("MaxSize", conf.MaxFileSizeInMB, "MaxBackup file", conf.MaxBackupFiles, "MaxAge", conf.MaxAge)
	l := &lumberjack.Logger{
		Filename:   "/opt/logs/" + fileName,
		MaxSize:    conf.MaxFileSizeInMB, // megabytes
		MaxBackups: conf.MaxBackupFiles,
		MaxAge:     conf.MaxAge, //days
		Compress:   false,       // disabled by default
	}

	return l
}

type NatsSubscriber struct {
	Subject       string
	QueueGroup    string
	containerID   string
	containerName string
	PodID         string
	Namespace     string
	FileConfig    types.Lmaas
	FlushTimeout  time.Duration
}

//Name of the handler
func (s *NatsSubscriber) Name() string {
	return s.Subject
}

func (m *TickerSyncMap) Load(key string) *fileTicker {
	val, ok := m.syncMap.Load(key)
	if ok {
		return val.(*fileTicker)
	} else {
		return nil
	}
}

func (m *TickerSyncMap) Store(key string, value *fileTicker) {
	m.syncMap.Store(key, value)
}

//NewNatsSubscription return new nats subscription
func NewNatsSubscription(subject string) *NatsSubscriber {
	subscription := &NatsSubscriber{
		Subject:      subject,
		QueueGroup:   types.Userinputs.QueueGroup,
		Namespace:    types.Namespace,
		PodID:        types.PodID,
		FileConfig:   *types.CimConfigObj.CimConfig.Lmaas, //*inputDetails,
		FlushTimeout: time.Duration(types.CimConfigObj.CimConfig.Lmaas.FlushTimeout) * time.Second,
	}

	return subscription
}

//HandleMessages handles the log messages
func (s *NatsSubscriber) HandleMessages(msg *nats.Msg) {
	metrics.NatsIncrementMessageCount(msg.Subject)
	metrics.NatsUpdateMessageSizeHistogram(msg.Subject, len(msg.Data))

	if msg.Subject == "CONFIG" {
		cID := utility.GenerateCorrelationId(100000, 999999)
		logs.LogConf.Audit(1, cID, "Recieved config update status from App..................")
		err1 := promutil.CounterAdd("cim_config_resp_received_total", 1, map[string]string{"pod": types.PodID})
		if err1 != nil {
			logs.LogConf.Error("Error while adding values in  counter  for total config response recieved", err1)
			// return
		}
		var commitDetails map[string]string
		err := json.Unmarshal(msg.Data, &commitDetails)
		if err != nil {
			logs.LogConf.Error("Error while unmarshalling the CONFIG data", err.Error())
			return
		}
		logs.LogConf.Audit(1, cID, "Config update status : ", commitDetails["status"])
		UpdateConfigCommitDetails(commitDetails, cID, "APP")
	} else if msg.Subject == "TEST" {
		logs.LogConf.Info("Recieved test subscription message")
	} else {
		logmsg := fbLog.GetRootAsLogMessage(msg.Data, 0)

		if promutil.PromEnabled {

			byteLen := len(logmsg.Payload())

			err := promutil.CounterAdd("cim_total_messages_counter", 1, map[string]string{
				"pod": types.PodID})

			if err != nil {
				log.Println("error while adding values in  counter  for total messages", err)
				return
			}

			err = promutil.CounterAdd("cim_total_bytes_counter", (float64(byteLen)), map[string]string{
				"pod": types.PodID})
			if err != nil {
				log.Println("error while adding values in  counter  for total bytes", err)
				return
			}

			err = promutil.GaugeSet("cim_packet_size", (float64(byteLen)), map[string]string{
				"pod": types.PodID})
			if err != nil {
				log.Println("error while adding values in  gauge  for packet size", err)
				return
			}
		}

		s.containerID = string(logmsg.ContainerId())
		s.containerName = string(logmsg.ContainerName())

		fileName := s.PodID + "_" + s.Namespace + "_" + s.containerName + "-" + s.containerID + ".log"
		ljFilename = fileName

		switch types.CimConfigObj.CimConfig.Lmaas.LoggingMode {
		case "TCP":
			LogModeTCP(logmsg.Payload(), fileName, s)
		case "KAFKA":
			LogModeKafka(logmsg.Payload(), fileName, s)
		case "STDOUT":
			LogModeSTDOUT(logmsg.Payload(), s.containerName)
		case "FILEANDSTDOUT":
			LogModeSTDOUT(logmsg.Payload(), s.containerName)
			writeBuffer(append(logmsg.Payload(), '\n'), fileName, s)
		default:
			writeBuffer(append(logmsg.Payload(), '\n'), fileName, s)
		}

	}
}

func CreateJsonHeader(logMsg []byte, containerName string) []byte {

	jsonConfig := JsonLog{
		Containername: containerName,
		Log:           string(logMsg),
		K8s: Kubernetes{
			Namespace: types.Namespace,
			Podname:   types.PodID,
		},
	}
	logB, err := json.Marshal(jsonConfig)
	if err != nil {
		log.Println("Error while creating json header", err)
	}

	return logB
}

func LogModeSTDOUT(logmsg []byte, containerName string) {
	if LogFormatJson {
		logmsg = CreateJsonHeader(logmsg, containerName)
	}
	fmt.Println(string(logmsg))
}

func LogModeTCP(logMsg []byte, fileName string, s *NatsSubscriber) {
	if !writeTCP(logMsg, s.containerName) {
		writeBuffer(append(logMsg, '\n'), fileName, s)
	}
}

func LogModeKafka(logMsg []byte, fileName string, s *NatsSubscriber) {
	if KafkaLogConn == nil {
		writeBuffer(append(logMsg, '\n'), fileName, s)
		return
	}
	if !KafkaLogConn.writeToKafka(logMsg, s.containerName) {
		writeBuffer(append(logMsg, '\n'), fileName, s)
	}
}

func writeBuffer(b []byte, fileName string, conf *NatsSubscriber) {
	var logBuf bytes.Buffer
	v, found := logBufMap.Load(fileName)
	if found {
		logBuf = v.(bytes.Buffer)
		totalLen := logBuf.Len() + len(b)
		if totalLen >= types.CimConfigObj.CimConfig.Lmaas.BufferSize {
			writeFile(fileName, logBuf.Bytes(), conf.FileConfig)
			//keep track of flushed time
			flushedFileMap.Store(fileName, time.Now())
			//reset the flush timeout
			if tickerMap.Load(fileName) == nil {
				logs.LogConf.Error("ticker not found for", fileName)
				return
			}

			tickerMap.Load(fileName).resetTicker(conf.FlushTimeout)

			logBuf.Reset()
			logBuf.Write(b)

			logBufMap.Store(fileName, logBuf)
		} else {
			logBuf.Write(b)
			logBufMap.Store(fileName, logBuf)

		}
	} else {
		newBuf := bytes.NewBuffer(make([]byte, 0, conf.FileConfig.BufferSize))
		newBuf.Write(b)
		logBufMap.Store(fileName, *newBuf)
		flushedFileMap.Store(fileName, time.Now())
		//reset the flush timeout
		tickerMap.Store(fileName, createTicker(conf.FlushTimeout))
		// start monitoring files flush log
		go conf.StartMonitoringFlushLog(fileName)
	}
}

func writeFile(fileName string, data []byte, conf types.Lmaas) {
	var (
		l *lumberjack.Logger
	)

	t, ok := loggerMap.Load(fileName)
	if !ok {
		l = NewLogger(fileName, conf)

		loggerMap.Store(fileName, l)
	} else {
		// if file pointer is already exist in the buffer then check if file is exist in the path
		l = t.(*lumberjack.Logger)
		if !Exists(l.Filename) {
			//close the FD opened (which is already deleted)
			l.Close()
			l = NewLogger(fileName, conf)
			//map the lgger object to filename

			loggerMap.Store(fileName, l)
		}
	}

	n, err := l.Write(data)
	if err != nil {
		logs.LogConf.Error(n, err)
		logs.LogConf.Error("Error while writing into file:", fileName, n, err)
		return
	}

}

//StartMonitoringFlushLog start monitorting the flush data
func (s *NatsSubscriber) StartMonitoringFlushLog(flushedfileName string) {
	var logBuf bytes.Buffer
	var flushedFileTime time.Time
	for {

		select {
		case <-tickerMap.Load(flushedfileName).ticker.C:
			v, _ := flushedFileMap.Load(flushedfileName)
			flushedFileTime = v.(time.Time)
			diff := time.Since(flushedFileTime)
			if diff.Seconds() >= s.FlushTimeout.Seconds()-1 {

				v, found := logBufMap.Load(flushedfileName)
				if found {
					logBuf = v.(bytes.Buffer)
					writeFile(flushedfileName, logBuf.Bytes(), s.FileConfig)

					flushedFileMap.Store(flushedfileName, time.Now())
					if tickerMap.Load(flushedfileName) == nil {
						logs.LogConf.Error("ticker not found for", flushedfileName)
						return
					}

					tickerMap.Load(flushedfileName).resetTicker(s.FlushTimeout)

					logBuf.Reset()

					logBufMap.Store(flushedfileName, logBuf)
				}
			}

		}
	}
}

//FlushDataBeforeExit flush data
func (s *NatsSubscriber) FlushDataBeforeExit() {

	logBufMap.Range(func(f, b interface{}) bool {
		bufferData := b.(bytes.Buffer)
		fileName := f.(string)
		logs.LogConf.Info("flushing", len(bufferData.Bytes()), " data to ", fileName)
		writeFile(fileName, bufferData.Bytes(), s.FileConfig)
		return true
	})

	logs.LogConf.Info("Done flushing the data before terminating")
}

func (s *NatsSubscriber) FlushBufferedLogsToFile() {

	logBufMap.Range(func(f, b interface{}) bool {
		bufferData := b.(bytes.Buffer)
		fileName := f.(string)
		logs.LogConf.Info("flushing", len(bufferData.Bytes()), " data to ", fileName)
		writeFile(fileName, bufferData.Bytes(), s.FileConfig)
		return true
	})

	logs.LogConf.Info("Done flushing the data before terminating")
}

// Exists reports whether the named file or directory exists.
func Exists(name string) bool {
	if _, err := os.Stat(name); err != nil {
		if os.IsNotExist(err) {
			return false
		}
	}
	return true
}

func RecreateLumberjackObj() {
	var l *lumberjack.Logger
	tmp, ok := loggerMap.Load(ljFilename)
	if !ok {
		logs.LogConf.Info("Lumberjack object not exist, while writing automatically created with new values")
		return
	}
	l = tmp.(*lumberjack.Logger)
	l.MaxSize = types.CimConfigObj.CimConfig.Lmaas.MaxFileSizeInMB
	l.MaxBackups = types.CimConfigObj.CimConfig.Lmaas.MaxBackupFiles
	l.MaxAge = types.CimConfigObj.CimConfig.Lmaas.MaxAge
	loggerMap.Delete(ljFilename)
	loggerMap.Store(ljFilename, l)
	logs.LogConf.Info("Lumberjack object Recreated with new values", l.MaxSize, l.MaxBackups, l.MaxAge)
	return
}
