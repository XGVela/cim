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
	"bufio"
	"cim/logs"
	"cim/types"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

const DIR = "/opt/logs/"

var (
	fileOffsetMap map[string]int
	bkpFilesMap   map[string][]os.FileInfo
)

func init() {
	fileOffsetMap = make(map[string]int)
	bkpFilesMap = make(map[string][]os.FileInfo)
}

func sortFiles(files []os.FileInfo) []os.FileInfo {
	sort.Slice(files, func(i, j int) bool {
		return files[i].ModTime().Before(files[j].ModTime())
	})
	return files
}

func storeLogfilesBasedOnTime(path string, files []os.FileInfo) {
	for k, _ := range bkpFilesMap {
		for _, j := range files {
			if strings.Contains(j.Name(), k) {
				bkpFilesMap[k] = append(bkpFilesMap[k], j)
			}
		}
		bkpFilesMap[k] = sortFiles(bkpFilesMap[k])
	}

}

func deleteFile(file string) {
	var err = os.Remove(file)
	if err != nil {
		logs.LogConf.Error(err)
	} else {
		fmt.Println(file, "removed")
	}

}

func getFileOffset(fileName string) int64 {
	v, ok := fileOffsetMap[fileName]
	if ok {
		return int64(v)
	}
	return 0
}

func Readln(r *bufio.Reader) ([]byte, error) {
	var (
		isPrefix bool  = true
		err      error = nil
		line, ln []byte
	)
	for isPrefix && err == nil {
		line, isPrefix, err = r.ReadLine()
		ln = append(ln, line...)
	}
	return ln, err
}

func sendFileToLogger(connection interface{}, fileName string) (int, error) {
	file, err := os.Open(fileName)
	if err != nil {
		logs.LogConf.Error(err)
		return 0, err
	}
	r := bufio.NewReader(file)

	offset := getFileOffset(fileName)
	r.Discard(int(offset))
	totalBytesRead := int(offset)

	if offset != 0 {
		log.Println("file ", fileName, " is partially read")
	}
	s, e := Readln(r)
	for {

		if !LogConnExist {
			log.Println("Connectivity down, file ", fileName, " partially transmited")
			go WatchFilesRotation()
			return totalBytesRead, fmt.Errorf("Connection down")
		}
		s, e = Readln(r)
		if e != nil {
			break
		}
		totalBytesRead = totalBytesRead + len(s) + 1

		containerName, err := GetContainerName(fileName)
		if err != nil {
			logs.LogConf.Error("skipping file", err)
			fmt.Println("deleting file", fileName)
			delete(fileOffsetMap, fileName)
			deleteFile(fileName)
			return totalBytesRead, nil
		}
		var payload []byte
		if types.CimConfigObj.CimConfig.Lmaas.LoggingMode == "TCP" {
			if LogFormatJson {
				payload = CreateJsonHeader(s, containerName)
			} else {
				payload = append(s, byte('\n'))
			}
			_, err = connection.(net.Conn).Write(payload)

		} else {
			connection.(*KafkaPublisher).writeToKafka(s, containerName)
		}

	}
	defer file.Close()
	fmt.Println("deleting file", fileName)
	delete(fileOffsetMap, fileName)
	deleteFile(fileName)
	return totalBytesRead, nil
}

func GetContainerName(filename string) (string, error) {
	s := strings.Split(filename, "_")
	if len(s) < 3 {
		return "", fmt.Errorf("Unable to get container name")
	}
	containerName := strings.Split(s[2], "-")
	if len(containerName) < 1 {
		return "", fmt.Errorf("Unable to get container name")
	}
	return containerName[0], nil
}

func getAbsPath(fileName string) string {
	fileName = DIR + fileName
	return fileName
}

func cleanFileOffsetMap() {
	for k := range fileOffsetMap {
		_, err := os.Stat(getAbsPath(k))
		if os.IsNotExist(err) {
			delete(fileOffsetMap, k)
		}
	}
}

func watchFile(fileName string, wg *sync.WaitGroup) {
	defer wg.Done()
	var old_file_size int64
	for {
		if LogConnExist {
			return
		}
		new_size := GetFileSize(fileName)
		if new_size < old_file_size {
			log.Println(fileName, " rotated")
			offset, ok := fileOffsetMap[fileName]
			if ok && offset != 0 {
				fileCount := len(bkpFilesMap[fileName])
				rotatedFileName := bkpFilesMap[fileName][fileCount-2].Name()
				fileOffsetMap[rotatedFileName] = fileOffsetMap[fileName]
				fileOffsetMap[fileName] = 0
				logs.LogConf.Debug("File ", fileName, " is rotated, New file created is ", rotatedFileName)
			}
		}
		old_file_size = new_size
		time.Sleep(1 * time.Second)
	}
}

func GetFileSize(fileName string) int64 {
	fi, err := os.Stat(fileName)
	if err != nil {
		return 0
	}
	size := fi.Size()
	return size
}

func WatchFilesRotation() {
	var wg sync.WaitGroup
	for k, _ := range bkpFilesMap {
		wg.Add(1)
		filename := DIR + k + ".log"
		logs.LogConf.Info("Start Watching :", filename)
		go watchFile(filename, &wg)
	}
	wg.Wait()
}

func Retransmit(files []os.FileInfo) {
	var c interface{}
	var err error
	if types.CimConfigObj.CimConfig.Lmaas.LoggingMode == "TCP" {
		c, err = GetConnection()
	} else {
		c, err = IntialiseKafkaProducer()
	}

	if err != nil {
		logs.LogConf.Info("Error while retransmission", err)
		return
	}

	logs.LogConf.Info("Start sending stored logs to the Endpoints")
	storeLogfilesBasedOnTime(DIR, files)
	for k, v := range bkpFilesMap {
		for _, f := range v {
			log.Println("sending file", f.Name())
			n, err := sendFileToLogger(c, getAbsPath(f.Name()))
			if err == nil {
				bkpFilesMap[k] = bkpFilesMap[k][1:]
			} else {
				fileOffsetMap[getAbsPath(f.Name())] = n
			}
		}
	}
	if types.CimConfigObj.CimConfig.Lmaas.LoggingMode == "TCP" {
		c.(net.Conn).Close()
	} else {
		c.(*KafkaPublisher).Writer.Close()
	}
	cleanFileOffsetMap()
}

func GetFiles() []os.FileInfo {
	files, err := ioutil.ReadDir(DIR)
	var podFiles []os.FileInfo
	if err != nil {
		logs.LogConf.Error(err)
	}
	 
	for _, v := range files {
		if strings.Contains(v.Name(), types.PodID+"_"+types.Namespace) {
			podFiles = append(podFiles, v)
		}
	}
	return podFiles
}

func updateFileMap(files []os.FileInfo) {
	for _, v := range files {
		filename := v.Name()
		extension := filepath.Ext(filename)
		if extension == ".log" {
			name := filename[0 : len(filename)-len(extension)]
			bkpFilesMap[name] = nil
		}

	}
}

//  Watch Log directory if it contains logs
func WatchLogDirectory() {
	// If retrax is not enabled then log will writing to the file.
	if types.CommonInfraConfigObj.EnableRetx != "true" {
		return
	}
	logfileSubscription := NewNatsSubscription("LOG")
	time.Sleep(logfileSubscription.FlushTimeout + 10*time.Second)
	logfileSubscription.FlushBufferedLogsToFile()
	files := GetFiles()
	if files != nil {
		updateFileMap(files)
		Retransmit(files)
	}

}
