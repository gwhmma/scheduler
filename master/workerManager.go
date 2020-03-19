package master

import (
	"context"
	"github.com/coreos/etcd/clientv3"
	"scheduler/common"
)

// 返回注册到etcd的所有worker
func WorkerList() (*[]string, error) {
	var rs []string

	getResp, err := common.ETCD.KV.Get(context.TODO(), common.JOB_WORKER_DIR, clientv3.WithPrefix())
	if err != nil {
		return &rs, err
	}

	if len(getResp.Kvs) > 0 {
		for _, v := range getResp.Kvs {
			rs = append(rs, common.ExtractName(string(v.Key), common.JOB_WORKER_DIR))
		}
	}

	return &rs, nil
}
