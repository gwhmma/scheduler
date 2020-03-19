package worker

import (
	"context"
	"encoding/json"
	"github.com/coreos/etcd/clientv3"
	"github.com/coreos/etcd/mvcc/mvccpb"
	"scheduler/common"
)

type Job struct {
	Name     string `json:"name"`      // 任务名
	Command  string `json:"command"`   // shell命令
	CronExpr string `json:"cron_expr"` //cron表达式
	Timeout  int64  `json:"timeout"`   // 任务执行的超时时间 秒
}

type EtcdManager struct {
	ETCD    *common.EtcdManager
	Watcher clientv3.Watcher
}

// put delete
type JobEvent struct {
	eventType int64
	job       *Job
}

var WorkEtcdManager *EtcdManager

// worker1 启动后从etcd获取任务列表  并实时监听任务的变化
func InitWorkJobManager() error {
	watcher := clientv3.NewWatcher(common.ETCD.Client)
	WorkEtcdManager = &EtcdManager{
		ETCD:    common.ETCD,
		Watcher: watcher,
	}

	// 从etcd获取任务列表 实时监听任务的变化
	WorkEtcdManager.watchJobs()
	WorkEtcdManager.watchKiller()

	return nil
}

// 监听任务时间变化  新增任务 修改任务  删除任务
func (w *EtcdManager) watchJobs() error {
	// 首先从etcd的jobs目录获取全量的任务列表
	getResp, err := w.ETCD.KV.Get(context.TODO(), common.JOB_SAVE_DIR, clientv3.WithPrefix())
	if err != nil {
		return err
	}

	// 得到当前的所有任务
	job := &Job{}
	for _, v := range getResp.Kvs {
		if err := json.Unmarshal(v.Value, job); err != nil {
			continue
		}
		// 构建一个任务事件
		jobEvent := buildJobEvent(common.JOB_EVENT_SAVE, job)

		// 将这个任务退给调度器
		Schedule.pushJobEvent(jobEvent)
	}

	// 从当前的reversion向后监听事件变化
	// 监听协程
	go func() {
		// 从get时刻的版本向后监听变化
		watchStartRevision := getResp.Header.Revision + 1
		// 监听JOB_SAVE_DIR 的变化
		watchChan := w.Watcher.Watch(context.TODO(), common.JOB_SAVE_DIR, clientv3.WithPrefix(), clientv3.WithRev(watchStartRevision))

		// 处理监听事件
		var jobEvent *JobEvent
		for watchResp := range watchChan {
			for _, event := range watchResp.Events {
				switch event.Type {
				case mvccpb.PUT:
					// 新建了任务 或者有任务被修改了
					job := &Job{}
					if err := json.Unmarshal(event.Kv.Value, job); err != nil {
						continue
					}
					// 构建一个jobEvent
					jobEvent = buildJobEvent(common.JOB_EVENT_SAVE, job)
				case mvccpb.DELETE:
					// 任务被删除了
					job := &Job{Name: common.ExtractName(string(event.Kv.Key), common.JOB_DELETE_DIR)}
					jobEvent = buildJobEvent(common.JOB_EVENT_DELETE, job)
				}

				// 将这个任务退给调度器
				Schedule.pushJobEvent(jobEvent)
			}
		}
	}()

	return nil
}

// 监听杀死任务事件
func (w *EtcdManager) watchKiller() {
	//监听 JOB_KILL_DIR目录的变化
	go func() {
		watchChan := w.Watcher.Watch(context.TODO(), common.JOB_KILL_DIR, clientv3.WithPrefix())

		for watchResp := range watchChan {
			for _, event := range watchResp.Events {
				switch event.Type {
				case mvccpb.PUT:
					// 有新的kill任务被提交
					job := &Job{Name: common.ExtractName(string(event.Kv.Key), common.JOB_KILL_DIR)}
					jobEvent := buildJobEvent(common.JOB_EVENT_KILL, job)
					// 将这个任务退给调度器
					Schedule.pushJobEvent(jobEvent)
				case mvccpb.DELETE:
					// 任务自动过期 被删除
				}
			}
		}
	}()
}

func buildJobEvent(eventType int64, job *Job) *JobEvent {
	return &JobEvent{
		eventType: eventType,
		job:       job,
	}
}
