package main

import (
	"bufio"
	"fmt"
	"github.com/liyue201/gostl/ds/queue"
	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
	"log"
	"net"
	"os"
	"path"
	"sync"
	"time"
)
const LOCALPATH = "D:\\ICT\\Yun_OS\\4人组\\code\\go\\FastDocker\\group_nolimit\\time_10"
const REMOTEDIR = "/root/strategy5/result"
const SERVER_LIST = "serverList"
const PASSWD = "vt@operat99"
const USER = "root"

var wg sync.WaitGroup

/*
	从各个主机上下载 /root/strategy5/result 文件 到本地并重命名为result_IP
*/
func main() {
	// 1. 从配置文件读取IP并放入队列
	file, err := os.Open(SERVER_LIST)
	if err != nil {
		log.Println(err)
		os.Create(SERVER_LIST)
	}
	defer file.Close()
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
		go func() {
			Download(REMOTEDIR,LOCALPATH,ip)
			log.Printf("download from %s completed.\n", ip)
			wg.Done()
		}()
	}
	wg.Wait()
}
// 用来测试的远程文件路径 和 本地文件夹
// remoteFilePath = "/path/to/remote/path/test.txt"
// localDir = "/local/dir"
func  Download(remoteFilePath, localDir ,ip string){
	var (
		err        error
		sftpClient *sftp.Client
	)

	// 这里换成实际的 SSH 连接的 用户名，密码，主机名或IP，SSH端口
	sftpClient, err = connect(USER, PASSWD,ip, 22)
	if err != nil {
		log.Fatal(err)
	}
	defer sftpClient.Close()

	srcFile, err := sftpClient.Open(remoteFilePath)
	if err != nil {
		log.Println(err)
		return
	}
	defer srcFile.Close()

	var localFileName = path.Base(remoteFilePath) + "_" + ip
	dstFile, err := os.Create(path.Join(localDir, localFileName))
	if err != nil {
		log.Fatal(err)
	}
	defer dstFile.Close()

	if _, err = srcFile.WriteTo(dstFile); err != nil {
		log.Fatal(err)
	}

	fmt.Println("copy file from remote server finished!")
}

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
