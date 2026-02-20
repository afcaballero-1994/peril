package pubsub

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/rabbitmq/amqp091-go"
)

func PublishJSON[T any](ch *amqp091.Channel, exchange, key string, val T) error {
	jsonData, err := json.Marshal(val)
	if err != nil {
		return err
	}
	err = ch.PublishWithContext(
		context.Background(), exchange,
		key, false, false, amqp091.Publishing{
			ContentType: "application/json",
			Body:        jsonData},
	)
	if err != nil {
		return err
	}
	return nil
}

type SimpleQueueType int
type Acktype int

const (
	Durable SimpleQueueType = iota
	Transient
)

const (
	Ack Acktype = iota
	NackRequeue
	NackDiscard
)

var QueueTypeName = map[SimpleQueueType]string{
	Durable:   "durable",
	Transient: "transient",
}

func DeclareAndBind(
	conn *amqp091.Connection,
	exchange, queueName, key string,
	queueType SimpleQueueType,
) (*amqp091.Channel, amqp091.Queue, error) {
	ch, err := conn.Channel()
	if err != nil {
		return &amqp091.Channel{}, amqp091.Queue{}, err
	}
	
	var dur bool = false
	var autoDelete bool = false
	var exclusive bool = false
	switch queueType {
	case Durable:
		dur = true
	case Transient:
		autoDelete = true
		exclusive = true
	}
	tq := amqp091.Table{}
	tq["x-dead-letter-exchange"] = "peril_dlx"
	qu, err := ch.QueueDeclare(queueName, dur, autoDelete,
		exclusive, false, tq)
	if err != nil {
		return nil, amqp091.Queue{}, fmt.Errorf("could not declare queu: %v", err)
	}

	err = ch.QueueBind(qu.Name, key, exchange, false, nil)
	fmt.Println("name:", qu.Name)

	if err != nil {
		return nil, amqp091.Queue{}, fmt.Errorf("could not bind queu: %v", err)
	}
	return ch, qu, nil
}

func SubscribeJSON[T any](conn *amqp091.Connection, exchange, queueName, key string,
	queueType SimpleQueueType, handler func(T) Acktype) error {
	ch, queue, err := DeclareAndBind(conn, exchange, queueName, key, queueType)
	if err != nil {
		return err
	}
	cc, err := ch.Consume(queue.Name, "", false, false, false, false, nil)
	if err != nil {
		return err
	}
	go func() {
		defer ch.Close()
		for a := range cc {
			var data T
			err = json.Unmarshal(a.Body, &data)
			if err != nil {
				fmt.Printf("Could not unmarshal message: %v\n", err)
			}
			switch handler(data) {
			case Ack:
				a.Ack(false)
				fmt.Println("Ack")
			case NackDiscard:
				a.Nack(false, false)
				fmt.Println("nackdiscard")
			case NackRequeue:
				a.Nack(false, true)
				fmt.Println("nacrequeue")
			}
		}
	}()
	return nil
}
