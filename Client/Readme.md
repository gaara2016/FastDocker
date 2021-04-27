# 各个文件功能说明
## imageList 
待分发的镜像
## serverList
待分发的节点IP
## start.go 
启动客户端程序
## client.go
拦截 `docker daemon` 的 `http` 请求，将请求重定向到分布式文件系统，并从分布式文件系统下载文件，最后将镜像导入daemon
## distribute/distributeFile.go
将编译好的 `client.go`文件从本地分发到各个节点，并修改文件可执行权限
## nginx/runNginx.go
1. `enable/disable` docker代理
2. `start/stop/restart/reload` nginx
3. 查询nginx状态
## pull/pull.go
向各个节点发送`docker pull`命令
## tag/tag.go
将imageList中的镜像打上tag再push到私有仓库，同时检查是否push成功