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
	"cim/connection"
	httpUtility "cim/http_utility"
	"cim/kubernetes"
	"cim/logs"
	notify_util "cim/notification"
	"cim/types"
	"cim/utility"
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"

	"github.com/nats-io/go-nats"

	//"strings"
	"syscall"
	"time"
)

var (
	logfileSubscription *NatsSubscriber
	wg                  sync.WaitGroup
)

func selfTestNatsPublish() bool {
	for i := 0; i < types.ConfigDetails.RemoteSvcRetryCount; i++ {
		logs.LogConf.Info("Attempting to publish test message to NATS")
		err = types.NatsConnection.Publish("TEST", []byte("Test Message"))
		if err != nil {
			logs.LogConf.Warning("Publishing test message to NATS failed. Error:", err, ". Re-attempting.")
			time.Sleep(500 * time.Millisecond)
			continue
		}
		logs.LogConf.Info("Test message published successfully to NATS")
		return true
	}

	logs.LogConf.Exception("Publishing test message to NATS failed after", types.ConfigDetails.RemoteSvcRetryCount, "attempts.")
	return false
}

//HandleNatsSubscriptions handles the topic subscribed to nats
func HandleNatsSubscriptions() {
	subscriptionStartTime := time.Now()
	types.Userinputs, _ = utility.GetUserInputs( /*args*/ )
	logs.LogConf.Info("topics are ===========>", types.Userinputs.Subject)
	logs.LogConf.Info("Existing Nats subjects", types.NatsSubscribedSubj)

	for _, subject := range types.Userinputs.Subject {
		if _, ok := types.NatsSubscribedSubj[subject]; ok {
			logs.LogConf.Info("Subject Already Exist And subscribed")
			continue
		} else {
			types.NatsSubscribedSubj[subject] = true
		}

		if subject == "" {
			continue
		}

		switch subject {
		case "LOG":
			logfileSubscription = NewNatsSubscription(subject)
			subscribe(logfileSubscription, subject)
			types.LOGTopicSubscribed = true
		case "CONFIG":
			if types.MsConfigVersion != "" {
				commitMSConfig := "commit-config/" + types.Namespace + "/" + types.MicroserviceName + "/" + types.AppVersion + "/ms/" + types.PodID
				connection.ConnectEtcd.Put(context.Background(), commitMSConfig, types.MsConfigVersion)
			}

			if types.NfConfigVersion != "" {
				commitNFConfig := "commit-config/" + types.Namespace + "/" + types.MicroserviceName + "/" + types.AppVersion + "/nf/" + types.PodID
				connection.ConnectEtcd.Put(context.Background(), commitNFConfig, types.NfConfigVersion)
			}
			if kubernetes.ConfigMapExist() {
				if types.ConfigMapRevision == "" {
					kubernetes.KubeConnect.GetCimCMRevision()
				}
				commitConfig := "commit-config/" + types.Namespace + "/" + types.MicroserviceName + "/" + types.AppVersion + "/" + "ms/cim/" + types.PodID
				connection.ConnectEtcd.Put(context.Background(), commitConfig, types.ConfigMapRevision)
			}
			logs.LogConf.Info("started waiting for config update")
			configSubscription := NewNatsSubscription(subject)

			go subscribeSync(configSubscription, subject)
		case "EVENT":
			logs.LogConf.Info("Subcribed to EVENT topic, Kafka enabled")
			// connect to kafka
			kafkaProducer, err := connection.Configure(types.CommonInfraConfigObj.KafkaBrokers, types.ConfigDetails.KafkaClientID, subject)
			if err != nil {
				logs.LogConf.Error("Unable to configure Kafka", err.Error())
				return
			}
			//defer kafkaProducer.Close()
			KafkaPublisherEvent = NewKafkaPublisher(subject, kafkaProducer)
			go subscribeSync(KafkaPublisherEvent, subject)
		case "TEST":
			testSubscription := NewNatsSubscription(subject)
			subscribe(testSubscription, subject)
		}
	}

	selfTestNatsPublish()
	logs.LogConf.Info("Done Subscription. Time taken:", time.Since(subscriptionStartTime))
}

//HandleCIMTermination graceful termination
func HandleCIMTermination() {
	// signal capturing for graceful termination.
	quitChannel := make(chan os.Signal)
	signal.Notify(quitChannel, syscall.SIGKILL, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP, os.Interrupt, os.Kill)
	select {
	case sig := <-quitChannel:
		// send shutdown to app
		terminationTime := time.Now()

		// fetch the grace period
		shutDownWaitTime, err := kubernetes.KubeConnect.GetPodDeletionGracePeriod(types.Namespace, types.PodID)
		if err != nil {
			// on error use default grace period = 30 seconds
			logs.LogConf.Warning("HandleCIMTermination: Unable to fetch deletion grace period from k8s client. Proceeding with graceperiod defined in configuration. Error:", err)
			shutDownWaitTime = 30
		}

		// use 3 seconds less than the grace period
		shutDownWaitTime -= 3
		if shutDownWaitTime < 0 {
			shutDownWaitTime = 0
		}

		//close etcd connection
		connection.ConnectEtcd.Close()

		logs.LogConf.Info("HandleCIMTermination: sending shutdown request to the application")
		response, err := httpUtility.ShutDownApp(types.AppShutDownCauseCimTermination)
		if err != nil {
			logs.LogConf.Error("HandleCIMTermination: error while sending shutdown request to app. Error:", err)
			shutDownWaitTime = 0
		} else {
			if response.StatusCode == http.StatusOK {
				logs.LogConf.Info("HandleCIMTermination: Application accepted the shutdown request. Response code:", response.StatusCode)
				notify_util.RaiseEvent(
					"ApplicationShutdownInitiated",
					[]string{},
					[]string{},
					[]string{"status", "reason", "signal", "description"},
					[]string{"success", "cim_termination", sig.String(), "Application shutdown initiated successfully"})
			} else {
				logs.LogConf.Warning("HandleCIMTermination: Application rejected the shutdown request. Response code:", response.StatusCode)
				notify_util.RaiseEvent(
					"ApplicationShutdownInitiated",
					[]string{},
					[]string{},
					[]string{"status", "reason", "signal", "description"},
					[]string{"failure", "cim_termination", sig.String(), "Application shutdown initiation failed"})
				shutDownWaitTime = 0
			}
		}

		appShutdownTimer := time.NewTimer(time.Duration(shutDownWaitTime) * time.Second)
		for _, subj := range types.Userinputs.Subject {
			if subj == "LOG" {
				var appShutdownCause string
				if len(types.AppShutDownCause) != 0 {
					appShutdownCause = types.AppShutDownCause[0]
				}
				logs.LogConf.Info("HandleCIMTermination: Shutdown cause:", appShutdownCause)
				done := make(chan bool)
				if appShutdownCause == types.AppShutDownCauseCimTermination {
					logs.LogConf.Info("HandleCIMTermination: Waiting for application to shut down. Waiting time(in seconds):", shutDownWaitTime)
					go func() {
						select {
						case <-types.FlushAppLogs:
							logs.LogConf.Info("HandleCIMTermination: Signal received from app to flush the logs")
							break
						case <-appShutdownTimer.C:
							logs.LogConf.Warning("HandleCIMTermination: Timout happened while waiting for the app to signal the log flush. Please consider increasing the grace period.")
							break
						}
						done <- true
					}()
				} else {
					logs.LogConf.Info("HandleCIMTermination: Shutdown cause other than", types.AppShutDownCauseCimTermination, "no need to wait")
					go func() {
						done <- true
					}()
				}
				<-done
				if !types.AppLogsFlushed {
					// If tcp connection is down, During termination it will attempt last time.
					if types.CommonInfraConfigObj.EnableRetx != "true" {
						// if timeout happend then flush the data
						log.Println("starting flushing data", sig)
						logfileSubscription.FlushDataBeforeExit()
						log.Println("finished flushing data")
					} else {
						WatchLogDirectory()
					}

					types.AppLogsFlushed = true
				}
			}
		}

		logs.LogConf.Info("HandleCIMTermination: time taken to complete all termination activities:", time.Since(terminationTime))
		kubernetes.RestartPod(types.PodID)
		os.Exit(0)
	}
}

//subscribe to nats with a topic
func subscribe(s SubscribeHandler, subject string) {
	logs.LogConf.Info("Subscribing to", subject, "topic")
	types.NatsConnection.Subscribe(subject, s.HandleMessages)
	types.NatsConnection.Flush()
	if err := types.NatsConnection.LastError(); err != nil {
		logs.LogConf.Exception("Error occured while subscribing to the topic:", subject, "Error:", err)
	}
	logs.LogConf.Info("Successfully subscribed to the topic", subject)
}

//subscribe to nats with a topic
func subscribeSync(s SubscribeHandler, subject string) {
	logs.LogConf.Info("Subscribing to", subject, "topic")
	sub, _ := types.NatsConnection.SubscribeSync(subject)
	types.NatsConnection.Flush()
	if err := types.NatsConnection.LastError(); err != nil {
		logs.LogConf.Exception("Error occured while subscribing to the topic:", subject, "Error:", err)
	}
	logs.LogConf.Info("Successfully subscribed to the topic", subject)
	for {
		msg, err := sub.NextMsg(10 * time.Second)
		if err != nil {
			if err != nats.ErrTimeout {
				logs.LogConf.Error("Error occured while fetching message for topic:", subject, ".Error:", err)
			}
			continue
		}
		s.HandleMessages(msg)
	}
}
