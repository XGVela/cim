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

package utility

import (
	"cim/types"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	"os"
	"strconv"

	jsonpatch "github.com/evanphx/json-patch"

	"strings"
	"time"
)

func ReadConfigFile(filename string) *types.Config {

	file, err := ioutil.ReadFile(filename)
	if err != nil {
		panic(err)
	}

	data := types.Config{}

	err = json.Unmarshal([]byte(file), &data)
	if err != nil {
		panic(err)
	}
	return &data

}

func HomeDir() string {
	if h := os.Getenv("HOME"); h != "" {
		return h
	}
	return os.Getenv("USERPROFILE") // windows
}

//GetUserInputs get user inputs
func GetUserInputs() (*types.UserInput, error) {
	var (
		userInputs = new(types.UserInput)
		topics     string
	)

	userInputs.QueueGroup = "log"
	topics = topics + "CONFIG,"
	topics = topics + "EVENT,"
	topics = topics + "LOG,"

	//if cdr outtopic is enabled
	if types.OutTopic != "" {
		topics = topics + types.OutTopic + ","
	}

	//add a test topic
	topics = topics + "TEST"

	userInputs.Subject = strings.Split(topics, ",")

	userInputs.MaxFileSizeMB = types.CimConfigObj.CimConfig.Lmaas.MaxFileSizeInMB
	userInputs.MaxBackupFiles = types.CimConfigObj.CimConfig.Lmaas.MaxBackupFiles
	userInputs.MaxAge = types.CimConfigObj.CimConfig.Lmaas.MaxAge
	userInputs.BufSize = types.CimConfigObj.CimConfig.Lmaas.BufferSize
	userInputs.FlushTimeout = time.Duration(types.CimConfigObj.CimConfig.Lmaas.FlushTimeout)
	userInputs.PodID = types.PodID
	userInputs.Namespace = types.Namespace

	return userInputs, nil
}

//GetAllEnvironmentVariablesSet get the env values
func GetAllEnvironmentVariablesSet() {
	var err error
	//get all the env variables set
	types.Namespace = os.Getenv("K8S_NAMESPACE")
	types.PodID = os.Getenv("K8S_POD_ID")
	types.MsConfigVersion = os.Getenv("MS_CONFIG_REVISION")
	types.NfConfigVersion = os.Getenv("NF_CONFIG_REVISION")

	// read cim REST port value
	types.CIMRestPort = 6060
	if os.Getenv("CIM_REST_PORT") != "" {
		types.CIMRestPort, err = strconv.Atoi(os.Getenv("CIM_REST_PORT"))
		if err != nil {
			log.Println("Improper value provided for env variable CIM_REST_PORT. Continuing with default CIM REST port 6060.")
			types.CIMRestPort = 6060
		}
	}

	// read cim NATS port value
	types.CIMNatsPort = 4222
	if os.Getenv("CIM_NATS_PORT") != "" {
		types.CIMNatsPort, err = strconv.Atoi(os.Getenv("CIM_NATS_PORT"))
		if err != nil {
			log.Println("Improper value provided for env variable CIM_NATS_PORT. Continuing with default CIM NATS port 4222.")
			types.CIMNatsPort = 4222
		}
	}
}

func GenerateCorrelationId(min int, max int) (id int) {
	rand.Seed(time.Now().UnixNano())
	id = rand.Intn(max-min) + min
	return
}

func ReadConfigMap() {
	var content []byte
	_, err := os.Stat("/opt/conf/cim.json")
	fmt.Println("Logs directory info : ", err)
	if os.IsNotExist(err) {
		fmt.Println("Config File not found reading from default location")
		content, err = ioutil.ReadFile("/opt/bin/cim.json")
	} else {
		content, err = ioutil.ReadFile("/opt/conf/cim.json")
	}
	if err != nil {
		log.Println("Error in reading config map", err.Error())
	}
	log.Println("Initial config data", string(content[:]))
	data := types.CimConfigObj
	err = json.Unmarshal(content, &data)
	if err != nil {
		panic(err)
	}
	types.ConfigDetails = data.CimConfig.App
	types.CimConfigObj = data
	log.Println("Unmarshall data", types.CimConfigObj.CimConfig.Subjects)
	log.Println("Unmarshall data", types.CimConfigObj.CimConfig.App)
	log.Println("Unmarshall data", types.CimConfigObj.CimConfig.Lmaas)
	return
}

func ApplyJsonPatch(patch []byte) (err error) {
	log.Println("Recieved json patch", string(patch[:]))
	decode_patch, err := jsonpatch.DecodePatch(patch)
	if err != nil {
		log.Println("Json patch decode fail", err.Error())
		return
	}
	original, _ := json.Marshal(types.CimConfigObj)
	modified, err := decode_patch.Apply(original)
	if err != nil {
		log.Println("Json patch failed", err.Error())
		return
	}
	orig, _ := json.Marshal(types.CimConfigObj)
	log.Println("Json patch successfull Original Value", string(orig[:]))
	data := types.CimConfigObj
	json.Unmarshal(modified, &data)
	types.CimConfigObj = data
	types.ConfigDetails = data.CimConfig.App
	types.CimConfigObj = data
	types.Userinputs, _ = GetUserInputs( /*args*/ )
	mod, _ := json.Marshal(types.CimConfigObj)
	log.Println("Json Patch successfull Modified value", string(mod[:]))
	return
}

// GetMD5Uid returns the md5sum hash of an input string in the form of UUID, i.e.,
// in the format - xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx
func GetMD5Uid(text string) string {
	hash := GetMD5Hash(text)
	return strings.Join(
		[]string{
			hash[:8],
			hash[8:12],
			hash[12:16],
			hash[16:20],
			hash[20:],
		},
		"-",
	)
}

// GetMD5Hash returns the md5sum hash of an input string
func GetMD5Hash(text string) string {
	hasher := md5.New()
	hasher.Write([]byte(text))
	return hex.EncodeToString(hasher.Sum(nil))
}
