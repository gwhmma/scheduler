package common

import (
	"context"
	"fmt"
	"github.com/coreos/etcd/clientv3"
	"github.com/coreos/etcd/mvcc/mvccpb"
)

type Lock struct {
	ctx        context.Context
	cancelFunc context.CancelFunc
	locked     bool
}

type MasterLock struct {
	ip      string
	etcd    *EtcdManager
	lock    *Lock
	leaseID clientv3.LeaseID
}

type JobLock struct {
	jobName string
	etcd    *EtcdManager
	lock    *Lock
	leaseID clientv3.LeaseID
}

var Locks *Lock
var Master *MasterLock

// 选举leader
func InitLeader() error {
	// 初始化锁失败
	if err := InitLock(); err != nil {
		return err
	}

	Master = &MasterLock{}
	if err := Master.initMasterLock(); err != nil {
		return err
	}

	if err := Master.tryLockMaster(); err != nil {
		return err
	}

	return nil
}

func InitLock() error {
	// 创建ctx 用于取消操作  取消自动续租
	ctx, cancelFunc := context.WithCancel(context.TODO())

	Locks = &Lock{
		ctx:        ctx,
		cancelFunc: cancelFunc,
	}

	return nil
}

// 初始化master锁
func (m *MasterLock) initMasterLock() error {
	// 得到本机ip
	ip, err := GetLocalIP()
	if err != nil {
		return err
	}

	m.ip = ip
	m.etcd = ETCD
	m.lock = Locks

	return nil
}

// 抢leader/master 分布式锁 成功的成为leader
func (m *MasterLock) tryLockMaster() error {
	// 创建租约  默认5s
	leaseGrant, err := ETCD.Lease.Grant(context.TODO(), 5)
	if err != nil {
		return err
	}

	leaseId := leaseGrant.ID

	// 自动续租
	leaseAliveChan, err := m.etcd.Lease.KeepAlive(m.lock.ctx, leaseId)
	if err != nil {
		// 取消自动续租
		m.lock.cancelFunc()
		// 释放租约
		m.etcd.Lease.Revoke(context.TODO(), leaseId)
		return err
	}

	// 处理自动续租的协程
	go func() {
		for {
			select {
			case keepChan := <-leaseAliveChan:
				if keepChan == nil {
					m.lock.locked = false
					// 往告警模块发送一个消息让其停止告警
					StopAlerts()
					goto END
				}
			}
		}
	END:
	}()

	// 创建txn事务
	txn := m.etcd.KV.Txn(context.TODO())
	lockKey := MASTER_LOCK_DIR + m.ip

	//事务抢锁
	txn.If(clientv3.Compare(clientv3.CreateRevision(lockKey), "=", 0)).
		Then(clientv3.OpPut(lockKey, "", clientv3.WithLease(leaseId))).Else(clientv3.OpGet(lockKey))

	// 提交事务
	txnResp, err := txn.Commit()
	if err != nil {
		// 取消自动续租
		m.lock.cancelFunc()
		// 释放租约
		m.etcd.Lease.Revoke(context.TODO(), leaseId)
		return err
	}

	// 判断抢锁是否成功
	if !txnResp.Succeeded {
		// 锁被占用
		// 取消自动续租
		m.lock.cancelFunc()
		// 释放租约
		m.etcd.Lease.Revoke(context.TODO(), leaseId)
		return ERR_LOCK_ALREADY_REQUIRED
	}

	m.lock.locked = true
	m.leaseID = leaseId

	return nil
}

// unlock
func (m *MasterLock) unlockMaster() {
	if m.lock.locked {
		m.lock.cancelFunc()
		m.etcd.Lease.Revoke(context.TODO(), m.leaseID)
	}
}

// 初始化任务执行锁
func CreateJobLock(jobName string) *JobLock {
	return &JobLock{
		jobName: jobName,
		etcd:    ETCD,
		lock:    Locks,
	}
}

// 尝试上锁
func (j *JobLock) TryLockJob() error {
	// 创建租约
	leaseGrant, err := j.etcd.Lease.Grant(context.TODO(), 5)
	if err != nil {
		return err
	}

	leaseId := leaseGrant.ID

	// 自动续租
	leaseKeepChan, err := j.etcd.Lease.KeepAlive(j.lock.ctx, leaseId)
	if err != nil {
		j.lock.cancelFunc()
		j.etcd.Lease.Revoke(context.TODO(), leaseId)
		return err
	}

	// 处理自动续租应答
	go func() {
		for {
			select {
			case keepChan := <-leaseKeepChan:
				if keepChan == nil {
					goto END
				}
			}
		END:
		}
	}()

	// 创建事务
	txn := j.etcd.KV.Txn(context.TODO())
	lockKey := JOB_LOCK_KEY + j.jobName

	// 事务抢锁
	txn.If(clientv3.Compare(clientv3.CreateRevision(lockKey), "=", 0)).
		Then(clientv3.OpPut(lockKey, "", clientv3.WithLease(leaseId))).Else(clientv3.OpGet(lockKey))

	// 提交事务
	txnResp, err := txn.Commit()
	if err != nil {
		j.lock.cancelFunc()
		j.etcd.Lease.Revoke(context.TODO(), leaseId)
		return err
	}

	// 抢锁成功返回   失败释放租约
	if !txnResp.Succeeded {
		j.lock.cancelFunc()
		j.etcd.Lease.Revoke(context.TODO(), leaseId)
		return ERR_LOCK_ALREADY_REQUIRED
	}

	j.leaseID = leaseId
	j.lock.locked = true

	return nil
}

func (j *JobLock) UnLock() {
	if j.lock.locked {
		j.lock.cancelFunc()
		j.etcd.Lease.Revoke(context.TODO(), j.leaseID)
	}
}

// 监听leader是否挂了 如果挂了马上参加选举
func WatchMaster(path string) {
	for {
		watcher := clientv3.NewWatcher(ETCD.Client)
		watchChan := watcher.Watch(context.TODO(), MASTER_LOCK_DIR, clientv3.WithPrefix())

		for watch := range watchChan {
			for _, event := range watch.Events {
				switch event.Type {
				case mvccpb.DELETE:
					// 如果master挂了 那么之前注册的key就会过期 delete掉
					// 重新开始一次选举
					if err := InitLeader(); err == nil {
						// 抢锁成功 成为master
						Master.lock.locked = true
						if err := InitAlert(path); err != nil {
							fmt.Println("重新选举后初始化alert出错")
						}
					}
				}
			}
		}
	}
}
