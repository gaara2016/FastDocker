package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
)

/*
	分析nginx的拦截日志，分析每个镜像的manifest文件，得出每个镜像需要的blobs
*/

const LOG = "forward_proxy.access.log"
const LOG_RESULT = "logResult.log"
const REGISTRY_PORT = "http://10.10.108.60:5000/v2/"
const BLOBSSHA = "/blobs/sha256:"

func main() {
	var blobs = make(map[string][]string, 50)
	file, err := os.Open(LOG)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		var (
			imageName    string
			blobID       string
			registryPort = "GET %s"
		)
		registryPort = fmt.Sprintf(registryPort, REGISTRY_PORT)
		// 过滤掉无用请求行
		if strings.Contains(line, registryPort) && strings.Contains(line, BLOBSSHA) {
			// 过滤掉 127.0.0.1 - - [17/Oct/2020:12:49:08 +0800] "
			line = strings.Split(line, "] \"")[1]
			// 过滤掉 GET
			line = strings.TrimLeft(line, "GET ")
			// 去掉 URI之后的部分
			line = strings.Split(line, " HTTP/1.1")[0]
			// 只留下镜像名和blobs ID部分
			line = strings.TrimPrefix(line, REGISTRY_PORT)
			tmp := strings.Split(line, "/")
			imageName = tmp[0]
			blobID = tmp[2]
			_, ok := blobs[imageName]
			if ok == false {
				blobs[imageName] = make([]string, 0, 20)
				//blobs[imageName] = append(blobs[imageName],blobID)
			}
			blobs[imageName] = append(blobs[imageName], blobID)
		}
	}
	// 去除同一个镜像中相同blob
	for key, value := range blobs {
		newSlice := removeDuplicate(value)
		blobs[key] = newSlice
	}
	// 将结果写入文件
	logWriter, er := os.OpenFile(LOG_RESULT, os.O_TRUNC|os.O_CREATE, 0644)
	if er != nil {
		log.Fatal(er)
	}
	defer logWriter.Close()
	writer := bufio.NewWriter(logWriter)
	for key, value := range blobs {
		fmt.Println(key, value)
		// 将string切片转成以逗号分隔的字符串
		str := strings.Replace(strings.Trim(fmt.Sprint(value), "[]"), " ", ",", -1)
		writer.WriteString(key + ": " + str + " " + strconv.Itoa(len(value)) + "\n")
		writer.Flush()
	}
}

// 数组/切片 去重
func removeDuplicate(arr []string) []string {
	resArr := make([]string, 0)
	tmpMap := make(map[string]interface{})
	for _, val := range arr {
		//判断主键为val的map是否存在
		if _, ok := tmpMap[val]; !ok {
			resArr = append(resArr, val)
			tmpMap[val] = nil
		}
	}

	return resArr
}
