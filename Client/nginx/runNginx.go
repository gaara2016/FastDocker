package main

import (
	"bufio"
	"bytes"
	"fmt"
	"github.com/liyue201/gostl/ds/queue"
	"golang.org/x/crypto/ssh"
	"io"
	"log"
	"net"
	"os"
	"strings"
	"sync"
)

/*
	从终端获取输入，根据输入做如下几件事情：
		1. 配置docker代理
		2. 启动nginx
		3. 取消docker代理
*/
const SERVER_LIST = "serverList"
const USER = "root"
const PASSWD = "vt@operat99"
const NGINXPATH = "/usr/local/nginx/sbin/nginx"
const DOCKERPROXYPATH = "/etc/systemd/system/docker.service.d"
const DCOKERPROXY = "cat > /etc/systemd/system/docker.service.d/https-proxy.conf << EOF\n[Service]\nEnvironment=\"HTTP_PROXY=http://localhost:443/\" \"HTTPS_PROXY=https://localhost:443/\" \"NO_PROXY=localhost,127.0.0.1,docker-registry.example.com,\"\nEOF\n"
const DOCKERPROXYFILE = "/etc/systemd/system/docker.service.d/https-proxy.conf"
const RESTARTDOCKER = "systemctl daemon-reload && systemctl restart docker"
const CHECKFASTDOCKER = "netstat -unltp | grep fastdocker"
const REGISTRY = "10.10.108.60"
const PORT = 5000
var wg sync.WaitGroup

func main() {
	info := `	*******************************************
	1: disable docker proxy config 
	2: enable docker proxy config
	3: start nginx
	4: stop nginx
	5: restart nginx
	6: reload nginx congif
	7: check nginx status 
	8. restart docker
	9. check docker proxy
	10. check fastdocker status
	11. remove result file
	12. cat copy_targz.sh
	13. cat docker_pull_hub.sh
    14. check dragonfly config
	*******************************************`
	log.Printf("根据提示输入相应数字：\n")
	fmt.Println(info)

	// 1. 从配置文件读取IP并放入队列
	file, err := os.Open(SERVER_LIST)
	if err != nil {
		log.Println(err)
		os.Create(SERVER_LIST)
	}
	defer file.Close()

	reader := bufio.NewReader(file)
	queue := queue.New()
	for {
		line, err := reader.ReadString('\n')
		// 去掉换行回车符
		line = strings.TrimSuffix(line, "\n")
		line = strings.TrimSuffix(line, "\r")
		if err != nil || err == io.EOF {
			//最后一行
			if len(line) > 0 {
				queue.Push(line)
			}
			break
		}
		queue.Push(line)
	}
	fmt.Printf("请选择你要执行的操作: \n")
	// 2. 从队列里面取出IP
	var input string
	fmt.Scanf("%s", &input)

	for !queue.Empty() {
		ip := (queue.Pop()).(string)
		// 3. 根据IP执行启动命令  多线程
		wg.Add(1)
		switch input {
		case "1":
			go disableDockerProxy(USER, PASSWD, ip)
		case "2":
			go enableDockerProxy(USER, PASSWD, ip)
		case "3":
			go runNingx(USER, PASSWD, ip)
		case "4":
			go stopNginx(USER, PASSWD, ip)
		case "5":
			go restartNginx(USER, PASSWD, ip)
		case "6":
			go reloadNginx(USER, PASSWD, ip)
		case "7":
			go checkNginxStatus(USER, PASSWD, ip)
		case "8":
			go restartDocker(USER, PASSWD, ip)
		case "9":
			go checkDcoerkProxy(USER, PASSWD, ip)
		case "10":
			go checkFastDockerStatus(USER, PASSWD, ip)
		case "11":
			go removeResult(USER, PASSWD, ip)
		case "12":
			go catTarGz(USER, PASSWD, ip)
		case "13":
			go catDockerPullHub(USER, PASSWD, ip)
		case "14":
			go checkDragonflyConfig(USER, PASSWD, ip)
		}
	}
	wg.Wait()
}

func checkDragonflyConfig(user, pwd, ip string) error {
	err := runSsh(user, pwd, ip, "sh bindwidth_dstat.sh")  // sh bindwidth_dstat.sh   sh deploy_dfclient.sh  rm -f dstat_net*
	if err != nil {
		return err
	}
	return nil
}

func catDockerPullHub(user, pwd, ip string) error {
	err := runSsh(user, pwd, ip, "docker images | grep 10.10.108.60:5000/ | awk -F' ' '{print $1}' ") // cat /root/docker_pull_hub.sh
	if err != nil {
		return err
	}
	return nil
}

func catTarGz(user, pwd, ip string) error {
	err := runSsh(user, pwd, ip, "docker rmi $(docker images | grep \"none\" | awk '{print $3}')")
	if err != nil {
		return err
	}
	return nil
}

func removeResult(user, pwd, ip string) error {
	err := runSsh(user, pwd, ip, "sh config-docker-proxy.sh") //
	if err != nil {
		return err
	}
	return nil
}

func checkFastDockerStatus(user, pwd, ip string) error {
	err := runSsh(user, pwd, ip, "sh dragonfly.sh") // CHECKFASTDOCKER
	if err != nil {
		return err
	}
	return nil
}

func checkDcoerkProxy(user, pwd, ip string) error {
	cmd := "docker pull %s:%d/centos"
	cmd = fmt.Sprintf(cmd,REGISTRY,PORT)
	err := runSsh(user, pwd, ip,cmd)
	if err != nil {
		return err
	}
	return nil
}

func restartDocker(user, pwd, ip string)  error {
	err := runSsh(user, pwd, ip, RESTARTDOCKER)
	if err != nil {
		return err
	}
	return nil
}

func enableDockerProxy(user, pwd, ip string) error {
	command := "mkdir -p " + DOCKERPROXYPATH + " && " + DCOKERPROXY
	//fmt.Println(command)
	err := runSsh(user, pwd, ip, command)
	if err != nil {
		return err
	}
	command = RESTARTDOCKER
	err = runSsh(user, pwd, ip, command)
	if err != nil {
		log.Println(err)
		return err
	}
	return nil
}

func disableDockerProxy(user, pwd, ip string) error {
	command := "rm -f " + DOCKERPROXYFILE
	err := runSsh(user, pwd, ip, command)
	if err != nil {
		log.Println(err)
		return err
	}
	command = RESTARTDOCKER
	err = runSsh(user, pwd, ip, command)
	if err != nil {
		log.Println(err)
		return err
	}
	return nil
}

func runNingx(user, pwd, ip string) error {
	command := NGINXPATH
	err := runSsh(user, pwd, ip, command)
	if err != nil {
		log.Println(err)
		return err
	}
	return nil
}

func stopNginx(user, pwd, ip string) error {
	command := NGINXPATH + " -s stop"
	err := runSsh(user, pwd, ip, command)
	if err != nil {
		log.Println(err)
		return err
	}
	return nil
}

func restartNginx(user, pwd, ip string) error {
	command := NGINXPATH + " -s restart"
	err := runSsh(user, pwd, ip, command)
	if err != nil {
		log.Println(err)
		return err
	}
	return nil
}

func reloadNginx(user, pwd, ip string) error {
	command := NGINXPATH + " -s reload"
	err := runSsh(user, pwd, ip, command)
	if err != nil {
		log.Println(err)
		return err
	}
	return nil
}

func checkNginxStatus(user, pwd, ip string) error {
	command := "netstat -unltp | grep nginx"
	err := runSsh(user, pwd, ip, command)
	if err != nil {
		log.Println(err)
		return err
	}
	return nil
}

func runSsh(user, password, ip, command string) error {

	var stdOut, stdErr bytes.Buffer

	session, err := SSHConnect(user, password, ip, 22)
	if err != nil {
		log.Print(err)
		return err
	}
	defer session.Close()

	session.Stdout = &stdOut
	session.Stderr = &stdErr

	session.Run(command)

	result := stdOut.String()
	if result != "" {
		fmt.Printf("%v\n",ip)
		fmt.Printf("%v",result)
	}
	wg.Done()
	return nil
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
