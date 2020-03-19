package worker

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"os/exec"
	"scheduler/common"
	"time"
)

type Executor struct {
}

//任务执行结果
type JobExeResult struct {
	exeInfo   *JobExecuteInfo
	startTime time.Time
	endTime   time.Time
	outPut    []byte
	err       error
}

var Exe *Executor

// 初始化执行器
func InitExecutor() {
	Exe = &Executor{}
}

func (e *Executor) ExecuteJob(info *JobExecuteInfo) {
	go func() {
		exeRes := &JobExeResult{exeInfo: info, outPut: make([]byte, 0)}
		// 为了消除不同机器时间的差异  导致的抢锁失败 在抢锁之前先随机睡眠一段时间
		time.Sleep(time.Millisecond * time.Duration(rand.Intn(1000)))

		// 首先获取锁
		// 初始化锁
		start := time.Now()
		jobLock := common.CreateJobLock(info.Job.Name)
		err := jobLock.TryLockJob()
		defer jobLock.UnLock()

		// 获取锁失败
		if err != nil {
			exeRes.err = err
			exeRes.startTime = start
			exeRes.endTime = time.Now()
		} else {
			// 抢锁成功 执行任务
			// 如果设置了超时时间 那么需要对任务的执行时间进行控制
			//timer := time.After(time.Duration(info.Job.Timeout) * time.Second)
			timer := time.AfterFunc(time.Duration(info.Job.Timeout)*time.Second,
				func() {
					fmt.Println("任务", info.Job.Name, " 执行超时 : ", time.Now())
					info.CancelFunc()
					ip, _ := common.GetLocalIP()
					alerts := &common.AlertsInfo{}
					alerts.Worker = ip
					alerts.AlertType = 1
					alerts.ErrorInfo = "任务执行超时"
					alerts.Time = time.Now().Format(common.TIME_FORMAT)
					// 发送这个告警消息
					body, _ := json.Marshal(alerts)
					common.Send(body)
				})

			cmd := exec.CommandContext(info.Ctx, "/bin/bash", "-c", info.Job.Command)
			// 执行命令 并捕获错误
			res, err := cmd.CombinedOutput()
			timer.Stop()
			exeRes.outPut = res
			exeRes.err = err
			exeRes.startTime = start
			exeRes.endTime = time.Now()
			fmt.Println(info.Job.Name, " 执行结果 : ", string(exeRes.outPut))
		}

		// 任务执行结束后 把该条记录推给scheduler  并从执行表里删除这条记录
		Schedule.pushJobExeRes(exeRes)
	}()
}
