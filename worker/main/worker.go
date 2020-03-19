package worker

func InitWorker() error {
	// 初始化 worker节点注册到etcd
	err := InitRegister()
	if err != nil {
		return err
	}

	// 初始化执行器
	InitExecutor()

	// 初始化任务调度器
	InitScheduler()

	// 初始化worker的任务管理器
	err = InitWorkJobManager()

	// 监听leader是否挂了 如果挂了马上参加选举


	return err
}
