package main

import (
	"bufio"
	"github.com/gorilla/mux"
	"log"
	"net/http"
	"os"
	"strings"
)

const HTTP_PORT = ":15901"
const FILENAME = "imageInfo"

var url = make(map[string]string, 50)

/*
	1. 读取imageInfo文件中信息，将其存入map中  key是镜像名  value是镜像在分布式文件系统中的路径
	2. 等待client发来http请求，根据请求镜像名从map中返回路径
*/

// http 服务器
func fileServer() {
	rtr := mux.NewRouter()
	rtr.HandleFunc("/{image}", func(writer http.ResponseWriter, request *http.Request) {
		params := mux.Vars(request)
		// 请求镜像名
		image := params["image"]
		log.Printf("Get a new connection from %v, requesting %v", request.RemoteAddr, image)
		// 检查镜像是否已上传
		ret := searchURL(image)
		// 已上传，直接返回路劲
		if ret != "" {
			writer.Write([]byte(ret))
			log.Printf("Have been distributed image %v", image)
		}
	}).Methods("GET")
	http.Handle("/", rtr)
	log.Println("Listening...")
	_ = http.ListenAndServe(HTTP_PORT, nil)
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
			var value string
			tmp := strings.Split(parse, ":")
			key = tmp[0]
			value = tmp[1]
			url[key] = value
		}
		put(line)
	}
}

func searchURL(imageName string) string {
	if imageName == "" {
		log.Println("请求镜像名为空")
		return ""
	}
	if v, ok := url[imageName]; ok {
		return v
	}
	return ""
}

func main() {
	writeMap(FILENAME)
	fileServer()
}
