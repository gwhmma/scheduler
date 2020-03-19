package common

import (
	"fmt"
	"github.com/BurntSushi/toml"
	"github.com/streadway/amqp"
)

type Mq struct {
	Url   string `toml:"url"`
	Queue string `toml:"queue"`
}

var mqcfg *Mq

func InitMqCfg(path string) error {
	mq := &Mq{}
	_, err := toml.DecodeFile(path, mq)

	mqcfg = &Mq{
		Url:   mq.Url,
		Queue: mq.Queue,
	}
	return err
}

func Send(body []byte) error {
	// 连接到mq
	conn, err := amqp.Dial(mqcfg.Url)
	defer conn.Close()
	if err != nil {
		return err
	}

	// 创建channel
	ch, err := conn.Channel()
	defer ch.Close()
	if err != nil {
		return err
	}
	queue, err := ch.QueueDeclare(mqcfg.Queue, false, false, false, false, nil)
	if err != nil {
		return err
	}

	// 发送数据
	err = ch.Publish("", queue.Name, false, false, amqp.Publishing{
		ContentType: "",
		Body:        body,
	})

	return err
}

func Receive() (<-chan amqp.Delivery, error) {
	// 连接到mq
	fmt.Println("mqcfg : ", mqcfg)
	conn, err := amqp.Dial(mqcfg.Url)
	defer conn.Close()
	if err != nil {
		return nil, err
	}

	// 创建channel
	ch, err := conn.Channel()
	defer ch.Close()
	if err != nil {
		return nil, err
	}

	queue, err := ch.QueueDeclare(mqcfg.Queue, false, false, false, false, nil)
	if err != nil {
		return nil, err
	}

	// 消费数据
	msgs, err := ch.Consume(queue.Name, "", true, false, false, false, nil)
	if err != nil {
		return nil, err
	}

	return msgs, nil
}
