package main

import (
	"flag"
	"fmt"
	"runtime"
	"scheduler/common"
	"scheduler/worker"
	"time"
)

func initEnv() {
	runtime.GOMAXPROCS(runtime.NumCPU())
}

var etcdConfig = flag.String("e", "conf/etcd.toml", "etcd配置文件路径")
var mongoConfig = flag.String("m", "conf/mongo.toml", "mongo配置文件路径")
var mqConfig = flag.String("mq", "conf/mq.toml", "mq配置文件路径")

func main() {
	flag.Parse()

	initEnv()

	// 初始化mq
	if err := common.InitMqCfg(*mqConfig); err != nil {
		return
	}

	// 初始化etcd
	if err := common.InitEtcdManager(*etcdConfig); err != nil {
		fmt.Println("初始化加载etcd配置出错 : ", err)
		return
	}

	if err := common.InitLock(); err !=nil {
		fmt.Println("Init lock 报错 : ", err)
		return

	}

	// 初始化MongoDB
	if err := common.InitLogSink(*mongoConfig); err != nil {
		fmt.Println("初始化加载MongoDB配置出错")
	}

	// 初始化 worker节点注册到etcd
	err := worker.InitRegister()
	if err != nil {
		return
	}

	// 初始化执行器
	worker.InitExecutor()

	// 初始化任务调度器
	worker.InitScheduler()

	// 初始化worker的任务管理器
	if err = worker.InitWorkJobManager(); err != nil {
		return
	}

	for {
		time.Sleep(10 * time.Second)
	}
}
