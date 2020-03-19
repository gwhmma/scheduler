package main

import (
	"flag"
	"fmt"
	"github.com/astaxie/beego"
	"runtime"
	"scheduler/common"
	_ "scheduler/router"
)

func initEnv() {
	runtime.GOMAXPROCS(runtime.NumCPU())
}

var etcdConfig = flag.String("e", "conf/etcd.toml", "etcd配置文件路径")
var mongoConfig = flag.String("m", "conf/mongo.toml", "mongo配置文件路径")
var alertConfig = flag.String("a", "conf/alert.toml", "alert配置文件路径")
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

	if err := common.InitLeader(); err != nil {
		// 抢锁失败成为follower
		fmt.Println("成为follower")
		go common.WatchMaster(*alertConfig)
	} else {
		// 抢锁成功成为leader
		// 负责告警操作
		fmt.Println("成为leader")
		go common.WatchMaster(*alertConfig)
		if err := common.InitAlert(*alertConfig); err != nil {
			fmt.Println("初始化alert出错 : ", err)
			return
		}
	}


	// 初始化MongoDB
	if err := common.InitLogSink(*mongoConfig); err != nil {
		fmt.Println("初始化加载MongoDB配置出错")
	}

	beego.Run()
}
