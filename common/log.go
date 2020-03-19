package common

import (
	"context"
	"time"
)

type JobLog struct {
	JobName      string `json:"jobName" bson:"jobName"`           // 任务名
	Command      string `json:"command" bson:"command"`           // 执行的命令
	OutPut       string `json:"outPut" bson:"outPut"`             // 任务执行的输出
	Error        string `json:"error" bson:"error"`               // 执行的错误信息
	PlanTime     int64  `json:"planTime" bson:"planTime"`         // 计划执行时间
	ScheduleTime int64  `json:"scheduleTime" bson:"scheduleTime"` // 调度时间
	StartTime    int64  `json:"startTime" bson:"startTime"`       // 时间开始时间
	EndTime      int64  `json:"endTime" bson:"endTime"`           // 执行完成时间
}

type Log struct {
	JobName string `json:"jobName"`
	Limit   int64  `json:"limit"`
	Skip    int64  `json:"skip"`
}

type LogSink struct {
	*Mongo
	LogChan        chan *JobLog
	AutoCommitChan chan *LogBatch
}

type LogBatch struct {
	logs []interface{}
}

var Sink *LogSink

func InitLogSink(path string) error {
	mc, err := MongoConn(path)
	if err != nil {
		return err
	}

	Sink = &LogSink{
		Mongo:          mc,
		LogChan:        make(chan *JobLog, 1000),
		AutoCommitChan: make(chan *LogBatch, 1000),
	}

	// 初始化日志存储协程
	go Sink.writeLoop()

	return nil
}

func (l *LogSink) writeLoop() {
	var batch *LogBatch
	var commitTimer *time.Timer

	for {
		select {
		case log := <-l.LogChan:
			// 写入MongoDB 每次写入需要进过网络 耗时  所以按批次写入
			if batch == nil {
				batch = &LogBatch{}
				// 让这个batch超时自动提交
				commitTimer = time.AfterFunc(time.Second,
					func(batch *LogBatch) func() {
						return func() {
							l.AutoCommitChan <- batch
						}
					}(batch))
			}

			batch.logs = append(batch.logs, log)
			// 如果批次里的日志已经到达一定数目 那么就直接提交这个批次
			if len(batch.logs) >= 100 {
				l.Collection.InsertMany(context.TODO(), batch.logs)
				batch = nil
				// 取消定时器
				commitTimer.Stop()
			}
		case timeOutBatch := <-l.AutoCommitChan:
			// 过期的批次
			// 将批次写入
			// 判断该批次是否为当前批次
			if timeOutBatch != batch {
				// 如果不是当前的批次 那么说明这个超时批次已经被提交过了  跳过它
				continue
			}

			l.Collection.InsertMany(context.TODO(), timeOutBatch.logs)
			batch = nil
		}
	}
}

func (l *LogSink) Append(log *JobLog) {
	// chan 有可能因为日志太多而阻塞  阻塞就直接丢弃日志
	select {
	case l.LogChan <- log:
	default:
	}
}
