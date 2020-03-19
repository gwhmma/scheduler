package worker

import (
	"context"
	"fmt"
	"github.com/coreos/etcd/clientv3"
	"scheduler/common"
	"time"
)

// 初始化worker注册到etcd
func InitRegister() error {
	ip, err := common.GetLocalIP()
	if err != nil {
		return err
	}

	registerKey := fmt.Sprintf("%s%s", common.JOB_WORKER_DIR, ip)

	go keepOnline(registerKey)

	return nil
}

// 自动注册到etcd的 /cron/worker1/ip目录下  并自动续租
func keepOnline(registerKey string) {
	for {
		//创建租约
		leaseGrant, err := common.ETCD.Lease.Grant(context.TODO(), 10)
		if err != nil {
			continue
		}

		// 自动续租
		keepChan, err := common.ETCD.Lease.KeepAlive(context.TODO(), leaseGrant.ID)
		if err != nil {
			continue
		}

		// 注册到etcd
		ctx, cancelFunc := context.WithCancel(context.TODO())
		_, err = common.ETCD.KV.Put(ctx, registerKey, "", clientv3.WithLease(leaseGrant.ID))
		if err != nil {
			cancelFunc()
			time.Sleep(time.Second)
			continue
		}

		// 处理续租应答
		for {
			select {
			case keep := <-keepChan:
				if keep == nil {
					goto RETRY
				}
			}
		}

	RETRY:
		time.Sleep(time.Second)
	}

}
