package master

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/coreos/etcd/clientv3"
	"go.mongodb.org/mongo-driver/mongo/options"
	"scheduler/common"
)

type Job struct {
	Name     string `json:"name"`      // 任务名
	Command  string `json:"command"`   // shell命令
	CronExpr string `json:"cron_expr"` //cron表达式
	Timeout  int64  `json:"timeout"`   // 任务执行的超时时间 秒
}

//保存任务到etcd
func (j *Job) SaveJob() (*Job, error) {
	job := &Job{}
	// 得到任务在etcd的保存目录
	jobKey := fmt.Sprintf("%s%s", common.JOB_SAVE_DIR, j.Name)

	// 对任务进行json序列化
	jobValue, err := json.Marshal(j)
	if err != nil {
		return job, err
	}

	// 保存到etcd
	putResp, err := common.ETCD.KV.Put(context.TODO(), jobKey, string(jobValue), clientv3.WithPrevKV())
	if err != nil {
		return job, err
	}

	// 如果是更新操作则返回原来的job
	if putResp.PrevKv != nil {
		json.Unmarshal(putResp.PrevKv.Value, job)
	}
	return job, nil
}

// 删除一个任务
func (j *Job) DeleteJob() (*Job, error) {
	job := &Job{}
	// 得到任务在etcd的保存目录
	jobKey := fmt.Sprintf("%s%s", common.JOB_SAVE_DIR, j.Name)

	delResp, err := common.ETCD.KV.Delete(context.TODO(), jobKey, clientv3.WithPrevKV())
	if err != nil {
		return job, err
	}

	// 解析原来的任务 并返回
	if len(delResp.PrevKvs) > 0 {
		json.Unmarshal(delResp.PrevKvs[0].Value, job)
	}
	return job, nil
}

// kill一个正在运行的任务
func (j *Job) KillJob() error {
	// 在kill目录put一个值  让worker监听这个目录
	delKey := fmt.Sprintf("%s%s", common.JOB_DELETE_DIR, j.Name)

	// 创建一个租约让key自动过期
	leaseGrant, err := common.ETCD.Lease.Grant(context.TODO(), 1)
	if err != nil {
		return err
	}

	if _, err := common.ETCD.KV.Put(context.TODO(), delKey, "", clientv3.WithLease(leaseGrant.ID)); err != nil {
		return err
	}
	return nil
}

// 返回所有的任务
func (j *Job) JobList() (jobs []*Job, err error) {
	getResp, err := common.ETCD.KV.Get(context.TODO(), common.JOB_SAVE_DIR, clientv3.WithPrefix())
	if err != nil {
		return
	}

	if len(getResp.Kvs) > 0 {
		for _, v := range getResp.Kvs {
			job := &Job{}
			if err := json.Unmarshal(v.Value, job); err != nil {
				continue
			}
			jobs = append(jobs, job)
		}
	}
	return
}

// 返回任务的执行日志
func JobLogs(log *common.Log) (*[]common.JobLog, error) {
	var logs = make([]common.JobLog, 0)

	// 过滤条件
	filter := common.JobFilter{JobName: log.JobName}
	// 排序规则
	sort := common.JobSort{Sort: -1}
	skip := log.Skip
	limit := log.Limit

	cursor, err := common.MongoDB.Collection.Find(context.TODO(), filter, &options.FindOptions{Skip: &skip, Limit: &limit, Sort: sort})
	defer cursor.Close(context.TODO())
	if err != nil {
		return &logs, err
	}

	for cursor.Next(context.TODO()) {
		jl := common.JobLog{}
		if err := cursor.Decode(&jl); err != nil {
			continue
		}
		logs = append(logs, jl)
	}

	return &logs, nil
}
