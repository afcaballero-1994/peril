package main

import (
	"fmt"
	"log"

	"github.com/bootdotdev/learn-pub-sub-starter/internal/gamelogic"
	"github.com/bootdotdev/learn-pub-sub-starter/internal/pubsub"
	"github.com/bootdotdev/learn-pub-sub-starter/internal/routing"
	"github.com/rabbitmq/amqp091-go"
)

func writesLog(gl routing.GameLog) pubsub.Acktype {
	defer fmt.Printf("> ")
	err := gamelogic.WriteLog(gl)
	if err != nil {
		return pubsub.NackRequeue
	}
	return pubsub.Ack
}

func main() {
	const connectionString string = "amqp://guest:guest@localhost:5672/"
	cotion, err := amqp091.Dial(connectionString)
	if err != nil {
		log.Fatalf("Could no create connection: %v", err)
	}
	defer cotion.Close()
	fmt.Println("Starting Peril server...")
	log.Println("Connection successful")
	chamq, err := cotion.Channel()
	if err != nil {
		log.Fatalf("Error creating channel: %v", err)
	}

	gamelogic.PrintServerHelp()

	err = pubsub.SubscribeGob(cotion,
		routing.ExchangePerilTopic,
		routing.GameLogSlug,
		routing.GameLogSlug + ".*", pubsub.Durable,
		writesLog)
	if err != nil {
		log.Fatalf("could not start consuming logs: %v\n", err)
	}
loop:
	for {
		inuser := gamelogic.GetInput()
		command := inuser[0]
		switch command {
		case "pause":
			log.Printf("Sending pause message\n")
			err = pubsub.PublishJSON(chamq, routing.ExchangePerilDirect,
				routing.PauseKey, routing.PlayingState{
					IsPaused: true,
				})
		case "resume":
			log.Printf("Sending resume message\n")
			err = pubsub.PublishJSON(chamq, routing.ExchangePerilDirect,
				routing.PauseKey, routing.PlayingState{
					IsPaused: false,
				})
		case "quit":
			log.Printf("Exiting server loop")
			break loop
		default:
			log.Println("Unknown Command")
		}

	}

	log.Println("\n Shutting down Connection")
}
