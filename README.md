# file-sync

## introduce
文件同步工具：使用http协议传输文件，将已同步文件暂存在内存中，并持久化到fileHash.txt文件中，做增量同步。

## warn
目前同步支持 windows -> windows linux -> linux 

server端同步路径与client一致

同步持久化文件保存在启动目录，在不同目录启动文件会导致增量同步异常

## install
```go
git clone https://github.com/Madou-Shinni/file-sync.git
```

## 快速开始

### server
默认监听8881端口

```go
go run main.go
```

### client
注意：你需要修改配置文件`config.yml`，使用你需要同步的server的路由前缀
```yml
# 路由前缀
url-prefix: http://192.168.110.173:8881
```

```shell
GLOBAL OPTIONS:
   --src value, -S value  需要同步文件的目录
   --help, -h             show help

go run main.go --src D:/go-project/frisbee-officer-backend-GVA/server/uploads/file/
```