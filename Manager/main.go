package main

import (
	"bufio"
	"errors"
	"fmt"
	"github.com/astaxie/beego/httplib"
	"github.com/gorilla/mux"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

const HTTP_PORT = ":15900"

// 镜像仓库文件路径前缀
const PRE_PATH = "/data/registry/docker/registry/v2/blobs/sha256"
const STORAGE = "10.10.108.85"
const GROUP = "group1"
const PORT = 8088
const FILENAME = "imageInfo"
const HOTBACKUP = "hotBackup"

var url = make(map[string][]string, 50)

// http 服务器
func fileServer() {
	rtr := mux.NewRouter()
	//rtr.HandleFunc("/request/{image}/{file:[:.a-zA-z0-9]+}", func(writer http.ResponseWriter, request *http.Request)
	rtr.HandleFunc("/v2/{image}/blobs/{file:[:.a-zA-z0-9]+}", func(writer http.ResponseWriter, request *http.Request) {
		params := mux.Vars(request)
		// 请求镜像名
		image := params["image"]
		// blobs
		fileName := params["file"]

		if ok := strings.Contains(fileName, "sha256:"); ok == false {
			log.Println("file name is wrong")
		}
		fileName = fileName[7:]

		log.Printf("Get a new connection from %v, requesting %v", request.RemoteAddr, fileName)
		// 检查镜像是否已上传
		ret := searchURL(image, fileName)
		// 已上传，直接返回路劲
		if ret != "" {
			writer.Write([]byte(ret))
			log.Printf("Have been distributed the blob %v of image %v", fileName, image)
		} else { // 未上传，则上传再返回
			// 开始热备
			start := time.Now()
			filePath, err := findFile(PRE_PATH, fileName)
			if err == nil {
				oldName := filePath + "/data"
				newName := filePath + "/sha256:" + fileName

				if _, ok := os.Stat(newName); ok != nil {
					if ok := os.Rename(oldName, newName); ok != nil {
						log.Println(ok)
					}
				}

				path := upload(STORAGE, GROUP, newName, PORT)
				// 热备结束
				elapsed := time.Since(start)
				content := image + ": " + elapsed.String()
				writeFile(HOTBACKUP, content)
				log.Printf("Hot backup %v completed  cost: %v", fileName, elapsed.String())
				if path != "" {
					saveURL(path, image)
					//fmt.Println(url)
					writer.Write([]byte(path))
					log.Printf("Have been distributed the blob %v of image %v", fileName, image)
					//writer.Write([]byte(" "))
				}

			} else {
				log.Println("Registry has not this image")
				writer.Write([]byte("Registry has not this image"))
				return
			}
		}
		toFile(FILENAME)
	}).Methods("GET")
	http.Handle("/", rtr)
	log.Println("Listening...")
	_ = http.ListenAndServe(HTTP_PORT, nil)
}

func writeFile(filename, content string) error {
	// 判断文件是否存在
	if _, ok := os.Stat(filename); ok != nil {
		if _, er := os.Create(filename); er != nil {
			log.Print(er)
			return er
		}
	}
	f, err := os.OpenFile(filename, os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		log.Printf("%v\n", err)
		return err
	}
	defer f.Close()
	wr := bufio.NewWriter(f)
	wr.WriteString(content)
	wr.WriteString("\n")
	wr.Flush()
	return nil
}

// 检查文件是否存在
func findFile(prePath, filename string) (string, error) {
	if prePath == "" || filename == "" {
		log.Print("path or file name is nil \n")
		return "", errors.New("path or file name is nil")
	}
	substr := filename[0:2]
	tmp := prePath + "/" + substr
	_, err := os.Stat(tmp)
	if err != nil {
		log.Println("该文件或目录不存在", err)
		return "", err
	}

	tmp = tmp + "/" + filename
	_, ok := os.Stat(tmp)
	if ok != nil {
		log.Println("该文件或目录不存在", err)
		return "", ok
	}

	return tmp, ok
}

// 将 filenmame 上传到 FastDFS?storageServer 的 group1
// storageServer : FastDFS存储服务器地址
// filename : 待上传的文件名
func upload(storageServer, group, filename string, port int) (path string) {
	var obj interface{}
	url := "http://" + storageServer + ":" + strconv.Itoa(port) + "/" + group + "/upload"
	//fmt.Println(url)
	req := httplib.Post(url)
	req.PostFile("file", filename)
	req.Param("output", "json")
	req.Param("scene", "")
	req.Param("path", "")
	req.ToJSON(&obj)
	fmt.Println(obj)

	if v, ok := obj.(map[string]interface{}); ok && len(v) > 0 {
		for key, value := range v { // 遍历map
			v, ok := value.(string)                      // 类型断言
			if strings.Compare(key, "path") == 0 && ok { // key是path 且值是 string类型
				path = v
				//fmt.Println(path)
			}
		}
	} else {
		path = ""
	}
	//fmt.Println("upload path :", path)
	return
}

// 将FastDFS返回路径写入map
func saveURL(path, imageName string) {
	_, ok := url[imageName]
	if ok != true {
		url[imageName] = make([]string, 0, 50)
	}
	url[imageName] = append(url[imageName], path)
}

// 将FastDFS返回路径写入磁盘文件
func toFile(filename string) error {
	// 判断文件是否存在
	if _, ok := os.Stat(filename); ok != nil {
		if _, er := os.Create(filename); er != nil {
			log.Print(er)
			return er
		}
	}
	f, err := os.OpenFile(filename, os.O_TRUNC|os.O_WRONLY, 0644)
	if err != nil {
		log.Printf("%v\n", err)
		return err
	}
	defer f.Close()
	wr := bufio.NewWriter(f)

	for key, value := range url {
		wr.WriteString(key + ": ")
		for index, path := range value {
			wr.WriteString(path)
			if index < len(value)-1 {
				wr.WriteString(",")
			}
		}
		wr.WriteString("\n")
		wr.Flush()
	}
	wr.Flush()

	return nil
}

//从磁盘文件读取路径并写入map
func writeMap(filename string) {
	file, err := os.Open(filename)
	if err != nil {
		log.Println(err)
		os.Create(filename)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()

		put := func(parse string) {
			var key string
			var value []string
			index := strings.Index(parse, ":")
			key = parse[:index]
			tmp := parse[index+2:]
			value = strings.Split(tmp, ",")
			url[key] = make([]string, 0, len(value))
			url[key] = value
		}

		put(line)
	}
}

func searchURL(imageName, filename string) (path string) {
	if imageName == "" || filename == "" {
		log.Println("镜像名有误或请求文件不存在")
		return
	}
	if _, ok := url[imageName]; ok {
		for _, v := range url[imageName] {
			if strings.Contains(v, filename) {
				path = v
				return path
			}
		}
	}
	return
}

func main() {
	writeMap(FILENAME)
	//for key,value := range url {
	//	fmt.Println(key,value)
	//}
	fileServer()
}
