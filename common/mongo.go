package common

import (
	"github.com/BurntSushi/toml"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"golang.org/x/net/context"
	"time"
)

type MongoCfg struct {
	MongoAddrs []string `toml:"mongoAddr"`
	Timeout    int64    `toml:"timeout"`
}

type Mongo struct {
	Client     *mongo.Client
	Collection *mongo.Collection
}

type JobFilter struct {
	JobName string `bson:"jobName"`
}

type JobSort struct {
	Sort int64 `bson:"sort"`
}

var MongoDB *Mongo

func loadMongoCfg(path string) (*MongoCfg, error) {
	cfg := &MongoCfg{}

	if _, err := toml.DecodeFile(path, cfg); err != nil {
		return cfg, err
	}
	return cfg, nil
}

func MongoConn(path string) (*Mongo, error) {
	m := &Mongo{}

	// 加载MongoDB配置文件
	cfg, err := loadMongoCfg(path)
	if err != nil {
		return m, err
	}

	ctx, _ := context.WithTimeout(context.Background(), time.Duration(cfg.Timeout)*time.Millisecond)
	client, err := mongo.Connect(ctx, options.Client().ApplyURI(cfg.MongoAddrs[0]))
	if err != nil {
		return m, err
	}

	m.Client = client
	m.Collection = client.Database("cron").Collection("log")

	MongoDB = &Mongo{
		Client:     client,
		Collection: client.Database("cron").Collection("log"),
	}

	return m, nil
}
