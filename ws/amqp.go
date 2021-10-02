package main

import (
	"log"

	"github.com/streadway/amqp"
)

const amqpWSLogin string = "guest"
const amqpWSPassword string = "guest"
const amqpWSHost string = "127.0.0.1"
const amqpWSPort string = "5672"

const (
	amqpPath = "amqp://" + amqpWSLogin + ":" + amqpWSPassword + "@" + amqpWSHost + ":" + amqpWSPort
)

func (hub *Hub) receive() {
	conn, err := amqp.Dial(amqpPath)
	if err != nil {
		log.Fatalf("Failed to connect to RabbitMQ: %s", err)
	}
	defer conn.Close()

	ch, err := conn.Channel()
	if err != nil {
		log.Fatalf("Failed to open a channel: %s", err)
	}
	defer ch.Close()

	q, err := ch.QueueDeclare(
		"default",
		false,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		log.Fatalf("Failed to declare a queue: %s", err)
	}

	msgs, err := ch.Consume(
		q.Name,
		"",
		true,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		log.Fatalf("Failed to register a consumer: %s", err)
	}

	forever := make(chan struct{})

	go func() {
		for d := range msgs {
			hub.broadcast <- d.Body
			log.Printf("Received a message: %s", d.Body)
		}
	}()

	log.Printf(" [*] Waiting for messages. To exit press CTRL+C")
	<-forever
}
