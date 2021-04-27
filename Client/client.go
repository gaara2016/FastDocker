package main

import (
	"bufio"
	"fmt"
	"github.com/gorilla/mux"
	"io"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"os"
	"sync"
	"time"
)
var lock sync.Mutex
const SERVER_SOCKET string = "10.10.108.60:15900"
const HTTP_PORT string = "13001"
const REGISTRY_BLOB_PATH_PREFIX = "http://10.10.108.85:8088%s"

// 4MB的文件缓冲区
const BUF_LEN = 4 * 1024 * 1024
const RESULT = "result"
//var storageServer = []string{"10.10.108.10:8088", "10.10.108.14:8088", "10.10.108.40:8088","10.10.108.41:8088",
//								"10.10.108.42:8088","10.10.108.43:8088", "10.10.108.44:8088", "10.10.108.85:8088"}
// storage servers
var storageServer = []string{"10.10.108.40:8088","10.10.108.41:8088", "10.10.108.42:8088","10.10.108.43:8088", "10.10.108.44:8088"}

// 接收nginx发来的blob请求
func httpServer() {
	rtr := mux.NewRouter()
	rtr.HandleFunc("/v2/{image:[a-zA-Z0-9-]+}/blobs/{file:[:.a-zA-z0-9]+}", func(writer http.ResponseWriter, request *http.Request) {
		start := time.Now()
		params := mux.Vars(request)
		fileName := params["file"]
		imageName := params["image"]
		log.Printf("You are requesting %v-%v\n", imageName, fileName)
		partUrl := handle(imageName, fileName)
		pullFromFastDFS(writer, partUrl, fileName, imageName)
		elapsed := time.Since(start)
		content := imageName + ":" + elapsed.String()
		fmt.Println(elapsed.String())
		writeFile(RESULT,content)
	}).Methods("GET")
	http.Handle("/", rtr)
	log.Println("HTTP Server started, listening :" + HTTP_PORT + "...")
	_ = http.ListenAndServe(":"+HTTP_PORT, nil)
}

// 处理文件下载请求 返回0表示出错
func handle(image string, blob string) (url string) {
	// 向 director 转发请求
	response, err := http.Get(fmt.Sprintf("http://"+SERVER_SOCKET+"/v2/%v/blobs/%v", image, blob))
	if err != nil {
		log.Println("Cannot connect to server, ", err)
		return ""
	}
	log.Printf("Connected to %v\n", response.Request.URL)
	// 等待 director 返回结果
	bytes, err := ioutil.ReadAll(response.Body)
	if err != nil {
		log.Println("Cannot read response.Body,", err)
		return ""
	}
	url = string(bytes) // 样例：/group1/default/20200925/10/04/2/main.go
	log.Printf("Receive %s from director\n", url)
	return
}

func pullFromFastDFS(writer http.ResponseWriter, partURL string, blob string, image string) {
	// 多节点间基于层的并发
	index := rand.Intn(10000) % len(storageServer)
	fullURL := "http://" + storageServer[index] + partURL
	//fullURL := fmt.Sprintf(REGISTRY_BLOB_PATH_PREFIX, partURL)
	log.Printf("Pulling from registry %s...\n", fullURL)
	response, err := http.Get(fullURL)
	if err != nil {
		log.Printf("Failed to connect to %s, %v\n", fullURL, err)
		return
	}
	download(writer, response.Body, blob)
}

// 从reader中读取数据并保存到本地文件, 同时向writer返回字节流
func download(writer http.ResponseWriter, reader io.ReadCloser, blob string) {
	// 开始从 reader 中读取流数据
	buffer := [BUF_LEN]byte{}
	for true {
		// 1 读取
		n, err1 := reader.Read(buffer[:])
		// 2 发送给客户端
		_, err2 := writer.Write(buffer[:n])
		// 3 异常处理 (golang 要求后检查错误: https://pkg.go.dev/io#Reader)
		if err1 != nil {
			if err1 == io.EOF {
				log.Printf("[C201] Blob %s is downloaded successfully.\n", blob)
				break
			} else {
				log.Println("[C202] Cannot download file, ", err1)
				return
			}
		}
		if err2 != nil {
			log.Printf("[C204] Cannot send to client, %v\n", err2)
			return
		}
	}
}

func writeFile(filename,content string) error {
	// 判断文件是否存在
	if _, ok := os.Stat(filename) ; ok != nil {
		if _, er := os.Create(filename) ; er != nil {
			log.Print(er)
			return er
		}
	}
	f, err := os.OpenFile(filename, os.O_APPEND | os.O_WRONLY, 0644)
	if err != nil {
		log.Printf("%v\n", err)
		return err
	}
	defer f.Close()
	lock.Lock()
	wr := bufio.NewWriter(f)
	wr.WriteString(content)
	wr.WriteString("\n")
	wr.Flush()
	lock.Unlock()
	return nil
}

func main() {
	rand.Seed(time.Now().Unix())
	httpServer()
}
