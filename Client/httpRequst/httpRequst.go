package main

import (
	"bufio"
	"bytes"
	"container/list"
	"fmt"
	"golang.org/x/crypto/ssh"
	"log"
	"net"
	"os"
	"strconv"
	"sync"
	"time"
)

/*
	1. 从配置文件读入IP和镜像名并放入队列
	2. 从队列里面取出IP和镜像明
	3. 根据IP和镜像名字执行docker pull命令  多线程
*/
const SERVER_LIST = "serverList"
const IMAGE_LIST = "imageList"
const PASSWD = "vt@operat99"
const USER = "root"
const COMMAND = "curl http://%s:%d/%s"
const RESULT = "result"
const PORT = 15902
var wg sync.WaitGroup
var lock sync.Mutex

type customQueue struct {
	queue *list.List
}

func (c *customQueue) Enqueue(value string) {
	c.queue.PushBack(value)
}

func (c *customQueue) Dequeue() error {
	if c.queue.Len() > 0 {
		ele := c.queue.Front()
		c.queue.Remove(ele)
	}
	return fmt.Errorf("Pop Error: Queue is empty")
}

func (c *customQueue) Front() (string, error) {
	if c.queue.Len() > 0 {
		if val, ok := c.queue.Front().Value.(string); ok {
			return val, nil
		}
		return "", fmt.Errorf("Peep Error: Queue Datatype is incorrect")
	}
	return "", fmt.Errorf("Peep Error: Queue is empty.")
}

func (c *customQueue) Size() int {
	return c.queue.Len()
}

func (c *customQueue) Empty() bool {
	return c.queue.Len() == 0
}

func runSsh(user, password, ip, imageName, command string,numbers int) {

	var stdOut, stdErr bytes.Buffer

	session, err := SSHConnect(user, password, ip, 22)
	if err != nil {
		log.Fatal(err)
		return
	}
	defer session.Close()

	session.Stdout = &stdOut
	session.Stderr = &stdErr
	err1 := session.Run(command)
	if err1 != nil {
		log.Println(err1)
	}
	log.Printf("%s pulling %s\n", ip, imageName)
	wg.Done()
	//result := stdOut.String()
	//
	//fmt.Printf("%s", result)
}

func SSHConnect(user, password, host string, port int) (*ssh.Session, error) {
	var (
		auth         []ssh.AuthMethod
		addr         string
		clientConfig *ssh.ClientConfig
		client       *ssh.Client
		session      *ssh.Session
		err          error
	)
	// get auth method
	auth = make([]ssh.AuthMethod, 0)
	auth = append(auth, ssh.Password(password))

	hostKeyCallbk := func(hostname string, remote net.Addr, key ssh.PublicKey) error {
		return nil
	}

	clientConfig = &ssh.ClientConfig{
		User: user,
		Auth: auth,
		// Timeout:             30 * time.Second,
		HostKeyCallback: hostKeyCallbk,
	}
	// connet to ssh
	addr = fmt.Sprintf("%s:%d", host, port)
	if client, err = ssh.Dial("tcp", addr, clientConfig); err != nil {
		return nil, err
	}

	// create session
	if session, err = client.NewSession(); err != nil {
		return nil, err
	}

	return session, nil
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
	var content string
	// 1. 从配置文件读取IP和镜像名字并放入队列
	fileServer, err := os.Open(SERVER_LIST)
	if err != nil {
		log.Println(err)
		os.Create(SERVER_LIST)
	}
	defer fileServer.Close()
	fileImage, err := os.Open(IMAGE_LIST)
	if err != nil {
		log.Println(err)
		os.Create(IMAGE_LIST)
	}
	defer fileImage.Close()

	readerIP := bufio.NewScanner(fileServer)
	readerImage := bufio.NewScanner(fileImage)
	queueIP := &customQueue{
		queue: list.New(),
	}
	var images []string
	var ipNumbers int
	for readerIP.Scan(){
		line := readerIP.Text()
		queueIP.Enqueue(line)
	}

	ipNumbers = queueIP.Size()
	// 读取镜像名入切片
	for readerImage.Scan(){
		line := readerImage.Text()
		images = append(images, line)
	}

	// 2. 从队列里面取出IP和镜像名
	start := time.Now()
	//size := queueIP.Size()
	for queueIP.Size() > 0 {
		var ip string
		ip, _ = queueIP.Front()
		queueIP.Dequeue()
		for _, v := range images {
			imageName := v
			// 3. 根据IP和镜像名执行pull命令  多线程
			wg.Add(1)
			cmd := fmt.Sprintf(COMMAND,ip,PORT,imageName)
			go runSsh(USER, PASSWD, ip, imageName, cmd,ipNumbers)
		}
	}
	wg.Wait()
	elapsed := time.Since(start)
	content = images[0] + ":" + elapsed.String() + " " + strconv.Itoa(ipNumbers)
	fmt.Println(elapsed.String())
	writeFile(RESULT,content)
}
