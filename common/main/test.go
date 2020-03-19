package main

import (
	"fmt"
	"github.com/streadway/amqp"
	"time"
)

func main()  {
	// 连接到mq
	conn, err := amqp.Dial("amqp://guest:guest@127.0.0.1:5672")
	defer conn.Close()
	if err != nil {
		return
	}

	// 创建channel
	ch, err := conn.Channel()
	defer ch.Close()
	if err != nil {
		return
	}

	queue, err := ch.QueueDeclare("alert", false, false, false, false, nil)
	if err != nil {
		return
	}

	// 消费数据
	msgs, err := ch.Consume(queue.Name, "", true, false, false, false, nil)
	if err != nil {
		return
	}

	go func() {
		for  {
			select {
			case m := <-msgs:
				fmt.Println("ch : ", string(m.Body))
			}
		}
	}()


	time.Sleep(time.Minute)
}

