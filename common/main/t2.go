package main

import "scheduler/common"

func main()  {
	common.InitMqCfg("conf/mq.toml")
	b := []byte("he he 1111")
	common.Send(b)
}
