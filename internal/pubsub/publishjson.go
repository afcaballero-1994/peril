package pubsub

import (
	"context"
	"encoding/json"

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

const (
	Durable SimpleQueueType = iota
	Transient
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
	qu, err := ch.QueueDeclare(queueName, dur, autoDelete,
		exclusive, false, nil)
	err = ch.QueueBind(queueName, key, exchange, false, nil)
	if err != nil {
		return &amqp091.Channel{}, amqp091.Queue{}, err
	}
	return ch, qu, nil
}
