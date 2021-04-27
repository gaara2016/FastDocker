package main

import (
	"bufio"
	"errors"
	"fmt"
	"github.com/astaxie/beego/httplib"
	"log"
	"os"
	"path"
	"strconv"
	"strings"
)

const HTTP_PORT = ":15900"

// 镜像仓库文件路径前缀
const PRE_PATH = "/root/experimentalImage"
const STORAGE = "10.10.108.14"
const GROUP = "group1"
const PORT = 8088
const IMAGE_INFO = "imageInfo"
const IMAGES_LIST = "imagesList"

var url = make(map[string][]string, 50)

/*将镜像（tar包）上传到DFS*/
func uploadImage(storageServer, group, filename string, port int) string {
	err := findFile(PRE_PATH, filename)
	if err != nil {
		return ""
	}
	name := path.Join(PRE_PATH, filename)
	ret := upload(storageServer, group, name, port)
	return ret
}

// 检查文件是否存在
func findFile(prePath, filename string) error {
	if prePath == "" || filename == "" {
		log.Print("path or file name is nil \n")
		return errors.New("path or file name is nil")
	}

	_, err := os.Lstat(path.Join(prePath, filename))
	if err != nil {
		fmt.Println("test")
		log.Println(err)
		return err
	}
	return nil
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

func main() {

	var imageMap = make(map[string]string, 20)

	file, err := os.OpenFile(IMAGES_LIST, os.O_CREATE, 0644)
	if err != nil {
		log.Println(err)
	}
	defer file.Close()

	var images = make([]string, 0, 20)
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		fileName := scanner.Text()
		images = append(images, fileName)
	}
	// 上传镜像
	for _, value := range images {
		uploadPath := uploadImage(STORAGE, GROUP, value+".tar", PORT)
		if uploadPath == "" {
			log.Println("path is nil")
			break
		}
		imageMap[value] = uploadPath
	}
	// 将路径写到磁盘  写文件一定要加 os.O_RDWR  ***
	fileWriter, ok := os.OpenFile(IMAGE_INFO, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0644)
	if ok != nil {
		log.Println(ok)
	}
	defer fileWriter.Close()

	writer := bufio.NewWriter(fileWriter)
	for key, value := range imageMap {
		content := key + ":" + value + "\n"
		_, err := writer.WriteString(content)
		if err != nil {
			log.Println(err)
		}
		writer.Flush()
	}
}
