package common

import (
	"github.com/BurntSushi/toml"
	"github.com/coreos/etcd/clientv3"
	"time"
)

type EtcdCfg struct {
	EndPoints []string `toml:"etcdEndPoints"`
	Timeout   int64    `toml:"etcdDialTimeout"`
}

type EtcdManager struct {
	Client *clientv3.Client
	KV     clientv3.KV
	Lease  clientv3.Lease
}

var ETCD *EtcdManager

// 从toml配置文件里加载etcd的配置信息
func loadEtcdCfg(path string) (*EtcdCfg, error) {
	cfg := &EtcdCfg{}

	if _, err := toml.DecodeFile(path, cfg); err != nil {
		return cfg, err
	}
	return cfg, nil
}

// 初始化etcdManager 得到etcd的client kv  lease
func InitEtcdManager(path string) error {
	cfg, err := loadEtcdCfg(path)
	if err != nil {
		return err
	}

	// 初始化配置
	etcdCfg := clientv3.Config{
		Endpoints:   cfg.EndPoints,
		DialTimeout: time.Duration(cfg.Timeout) * time.Millisecond,
	}

	// 建立连接 得到client
	client, err := clientv3.New(etcdCfg)
	if err != nil {
		return err
	}

	// 得到kv和lease
	kv := clientv3.NewKV(client)
	lease := clientv3.NewLease(client)

	ETCD = &EtcdManager{
		Client: client,
		KV:     kv,
		Lease:  lease,
	}

	return nil
}
