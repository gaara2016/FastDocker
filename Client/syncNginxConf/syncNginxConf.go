package main

import (
	"bufio"
	"fmt"
	"github.com/liyue201/gostl/ds/queue"
	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
	"io/ioutil"
	"log"
	"net"
	"os"
	"path"
	"sync"
	"time"
)

/*
 将策略5客户端程序分发到各个节点

*/
const LOCALPATH = "D:\\ICT\\Yun_OS\\4人组\\code\\go\\FastDocker\\nginx\\/nginx.conf"
const REMOTEDIR = "/usr/local/nginx/conf/"
const SERVER_LIST = "serverList"
const PASSWD = "vt@operat99"
const USER = "root"

var wg sync.WaitGroup

func connect(user, password, host string, port int) (*sftp.Client, error) {
	var (
		auth         []ssh.AuthMethod
		addr         string
		clientConfig *ssh.ClientConfig
		sshClient    *ssh.Client
		sftpClient   *sftp.Client
		err          error
	)
	// get auth method
	auth = make([]ssh.AuthMethod, 0)
	auth = append(auth, ssh.Password(password))

	clientConfig = &ssh.ClientConfig{
		User:    user,
		Auth:    auth,
		Timeout: 30 * time.Second,
		HostKeyCallback: func(hostname string, remote net.Addr, key ssh.PublicKey) error {
			return nil
		},
	}

	// connet to ssh
	addr = fmt.Sprintf("%s:%d", host, port)

	if sshClient, err = ssh.Dial("tcp", addr, clientConfig); err != nil {
		return nil, err
	}

	// create sftp client
	if sftpClient, err = sftp.NewClient(sshClient); err != nil {
		return nil, err
	}

	return sftpClient, nil
}

/* 	注意分发文件 需要转义符把文件名转义下
	示例：  "D:\\ICT\\Yun_OS\\4人组\\code\\go\\FastDocker\\/nginx.conf"
*/
func uploadFile(sftpClient *sftp.Client, localFilePath string, remotePath string) {
	srcFile, err := os.Open(localFilePath)
	if err != nil {
		fmt.Println("os.Open error : ", localFilePath)
		log.Fatal(err)

	}
	defer srcFile.Close()

	var remoteFileName = path.Base(localFilePath)
	dstFile, err := sftpClient.Create(path.Join(remotePath, remoteFileName))
	if err != nil {
		fmt.Println("sftpClient.Create error : ", path.Join(remotePath, remoteFileName))
		log.Println(err)

	}
	defer dstFile.Close()

	ff, err := ioutil.ReadAll(srcFile)
	if err != nil {
		fmt.Println("ReadAll error : ", localFilePath)
		log.Fatal(err)

	}
	dstFile.Write(ff)
	fmt.Println(localFilePath + "  copy file to remote server finished!")
}

func uploadDirectory(sftpClient *sftp.Client, localPath string, remotePath string) {
	localFiles, err := ioutil.ReadDir(localPath)
	if err != nil {
		log.Fatal("read dir list fail ", err)
	}

	for _, backupDir := range localFiles {
		localFilePath := path.Join(localPath, backupDir.Name())
		remoteFilePath := path.Join(remotePath, backupDir.Name())
		if backupDir.IsDir() {
			sftpClient.Mkdir(remoteFilePath)
			uploadDirectory(sftpClient, localFilePath, remoteFilePath)
		} else {
			uploadFile(sftpClient, path.Join(localPath, backupDir.Name()), remotePath)
		}
	}

	fmt.Println("copy directory to remote server finished!")
}

func main() {
	// 1. 从配置文件读取IP并放入队列
	file, err := os.Open(SERVER_LIST)
	if err != nil {
		log.Println(err)
	}
	defer file.Close()

	start := time.Now()
	scanner := bufio.NewScanner(file)
	queue := queue.New()
	for scanner.Scan(){
		line := scanner.Text()
		queue.Push(line)
	}
	// 2. 从队列里面取出IP
	for !queue.Empty() {
		var ip string
		ip = (queue.Pop()).(string)
		// 3. 根据IP执行分发操作  多线程
		wg.Add(1)
		var (
			err        error
			sftpClient *sftp.Client
		)
		go func() {
			sftpClient, err = connect(USER, PASSWD, ip, 22)
			if err != nil {
				log.Fatal(err)
			}
			defer sftpClient.Close()
			uploadFile(sftpClient, LOCALPATH, REMOTEDIR)
			log.Printf("copy to %s completed.\n", ip)
			wg.Done()
		}()
	}
	wg.Wait()

	elapsed := time.Since(start)
	fmt.Println("elapsed time : ", elapsed)
}
