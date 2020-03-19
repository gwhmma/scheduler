package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/gorhill/cronexpr"
	"scheduler/common"
	"time"
)

// 任务调度
type Scheduler struct {
	JobEventChan    chan *JobEvent              // etcd的任务事件chan
	JobPlanMap      map[string]*JobSchedulePlan // 任务调度计划列表
	JobExecutingMap map[string]*JobExecuteInfo  // 正在执行的任务列表
	JobExeResChan   chan *JobExeResult          // 任务的执行结果
}

// 任务调度计划
type JobSchedulePlan struct {
	Job      *Job                 //任务信息
	CronExpr *cronexpr.Expression // 解析好的cron表达式
	NextTime time.Time            //任务下次的执行时间
}

// 任务执行状态
type JobExecuteInfo struct {
	Job        *Job               //任务信息
	PlanTime   time.Time          // 任务计划执行时间
	RealTime   time.Time          // 任务实际执行时间
	Ctx        context.Context    // 用于取消任务的上下文信息
	CancelFunc context.CancelFunc // 用于取消任务的cancel func
}

var Schedule *Scheduler

func InitScheduler() {
	Schedule = &Scheduler{
		JobEventChan:    make(chan *JobEvent, 1000),
		JobPlanMap:      make(map[string]*JobSchedulePlan),
		JobExecutingMap: make(map[string]*JobExecuteInfo),
		JobExeResChan:   make(chan *JobExeResult, 1000),
	}

	go Schedule.scheduleLoop()
}

//  任务调度协程
func (s *Scheduler) scheduleLoop() {
	// 初始化调度一次
	scheduleAfter := s.trySchedule()

	// 调度的延时定时器
	timer := time.NewTimer(scheduleAfter)

	for {
		select {
		case jobEvent := <-s.JobEventChan: //监听任务的变化
			// 对内存中维护的任务进行增删改查
			s.handleJobEvent(jobEvent)
		case <-timer.C:
		// 最近的任务到期了
		case jobExeRes := <-s.JobExeResChan: // 监听任务执行结果
			// 处理任务的执行结果
			s.handleJonExeRes(jobExeRes)
		}

		// 调度一次任务
		scheduleAfter = s.trySchedule()
		// 重置计时器
		timer.Reset(scheduleAfter)
	}
}

// 计算任务调度状态
// 遍历所有任务
// 过期的任务立即执行
// 计算最近的即将过期的任务的时间
func (s *Scheduler) trySchedule() time.Duration {
	now := time.Now()
	var nearTime *time.Time

	//如果当前还没有任务
	if len(s.JobPlanMap) == 0 {
		return time.Second
	}

	// 遍历任务李列表
	for _, plan := range s.JobPlanMap {
		// 如果当前有过期的任务 则立即执行
		if plan.NextTime.Before(now) || plan.NextTime.Equal(now) {
			// 构建任务执行状态信息
			s.tryStartJob(plan)
			// 在计算这个任务的下次执行时间
			plan.NextTime = plan.CronExpr.Next(now)
		}

		// 统计最近要过期任务的时间
		if nearTime == nil || plan.NextTime.Before(*nearTime) {
			nearTime = &plan.NextTime
		}
	}

	// 下次任务的调度时间间隔   下次调度的时间 = 最近要调度的时间 - 单前时间
	return (*nearTime).Sub(now)
}

func (s *Scheduler) pushJobEvent(event *JobEvent) {
	s.JobEventChan <- event
}

// 处理任务事件
func (s *Scheduler) handleJobEvent(event *JobEvent) {
	switch event.eventType {
	case common.JOB_EVENT_SAVE: // 任务保存事件
	    // 如果修改了一个正在执行的任务  那么就先把这个任务停止
		if exe, exist := s.JobExecutingMap[event.job.Name]; exist {
			exe.CancelFunc()
			delete(s.JobExecutingMap, event.job.Name)
		}

		// 构造一个任务事件 
		plan, err := s.buildSchedulePlan(event.job)
		if err != nil {
			return
		}
		// 将这个新的任务事件保存到内存中
		s.JobPlanMap[plan.Job.Name] = plan
	case common.JOB_EVENT_DELETE: // 任务删除事件
		// 将该任务从内存中删除
		if _, exist := s.JobPlanMap[event.job.Name]; exist {
			delete(s.JobPlanMap, event.job.Name)
		}
		if exe, exist := s.JobExecutingMap[event.job.Name]; exist {
			exe.CancelFunc()
			delete(s.JobExecutingMap, event.job.Name)
		}
	case common.JOB_EVENT_KILL: // 任务杀死事件
		// 处理任务杀死事件
		// 取消command执行  首先判断该任务是否在执行
		if exe, exist := s.JobExecutingMap[event.job.Name]; exist {
			// 触发command杀死shell子进程  任务退出
			exe.CancelFunc()

			ip, _ := common.GetLocalIP()
			alerts := &common.AlertsInfo{}
			alerts.Worker = ip
			alerts.AlertType = 3
			alerts.ErrorInfo = "任务被强制杀死"
			alerts.Time = time.Now().Format(common.TIME_FORMAT)
			// 发送这个告警消息
			body, _ := json.Marshal(alerts)
			common.Send(body)

			delete(s.JobExecutingMap, event.job.Name)
		}
	}
}

func (s *Scheduler) buildSchedulePlan(job *Job) (*JobSchedulePlan, error) {
	expr, err := cronexpr.Parse(job.CronExpr)
	if err != nil {
		return nil, err
	}

	return &JobSchedulePlan{
		Job:      job,
		CronExpr: expr,
		NextTime: expr.Next(time.Now()),
	}, nil
}

// 尝试启动一个任务
func (s *Scheduler) tryStartJob(plan *JobSchedulePlan) {
	// 先查看这个任务是否在执行
	if _, executing := s.JobExecutingMap[plan.Job.Name]; executing {
		fmt.Println("job is executing...")
		return
	}

	// 构建任务执行状态信息
	exeInfo := s.buildJobExecuteInfo(plan)
	// 保存任务的执行状态 (正在执行)
	s.JobExecutingMap[plan.Job.Name] = exeInfo

	//执行任务
	fmt.Println("执行任务 : ", exeInfo.Job.Name)
	Exe.ExecuteJob(exeInfo)
}

func (s *Scheduler) buildJobExecuteInfo(plan *JobSchedulePlan) *JobExecuteInfo {
	exeInfo := &JobExecuteInfo{
		Job:      plan.Job,
		PlanTime: plan.NextTime,
		RealTime: time.Now(),
	}

	exeInfo.Ctx, exeInfo.CancelFunc = context.WithCancel(context.TODO())
	return exeInfo
}

func (s *Scheduler) pushJobExeRes(res *JobExeResult) {
	s.JobExeResChan <- res
}

func (s *Scheduler) handleJonExeRes(res *JobExeResult) {
	// 从任务执行表中删除这个任务
	delete(s.JobExecutingMap, res.exeInfo.Job.Name)

	if res.err != common.ERR_LOCK_ALREADY_REQUIRED {
		// 生成日志 保存日志
		log := &common.JobLog{
			JobName:      res.exeInfo.Job.Name,
			Command:      res.exeInfo.Job.Command,
			OutPut:       string(res.outPut),
			PlanTime:     res.exeInfo.PlanTime.UnixNano() / 1000000,
			ScheduleTime: res.exeInfo.RealTime.UnixNano() / 1000000,
			StartTime:    res.startTime.UnixNano() / 1000000,
			EndTime:      res.endTime.UnixNano() / 1000000,
		}

		if res.err != nil {
			log.Error = res.err.Error()
		}

		if res.err != nil && res.err != common.ERR_LOCK_ALREADY_REQUIRED {
			ip, _ := common.GetLocalIP()
			alerts := &common.AlertsInfo{}
			alerts.Worker = ip
			alerts.AlertType = 2
			alerts.ErrorInfo = "任务执行失败"
			alerts.Time = time.Now().Format(common.TIME_FORMAT)
			// 发送这个告警消息
			body, _ := json.Marshal(alerts)
			common.Send(body)
		}

		common.Sink.Append(log)
	}
}
