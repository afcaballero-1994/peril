package main

import (
	"fmt"
	"log"

	"github.com/bootdotdev/learn-pub-sub-starter/internal/gamelogic"
	"github.com/bootdotdev/learn-pub-sub-starter/internal/pubsub"
	"github.com/bootdotdev/learn-pub-sub-starter/internal/routing"
	"github.com/rabbitmq/amqp091-go"
)

func main() {
	fmt.Println("Starting Peril client...")
	const connectionString string = "amqp://guest:guest@localhost:5672/"
	conn, err := amqp091.Dial(connectionString)
	if err != nil {
		log.Fatalf("Could not connect to rabbit: %v", err)
	}
	defer conn.Close()
	uname, err := gamelogic.ClientWelcome()
	if err != nil {
		log.Fatalf("Could not get username: %v", err)
	}
	_, _, err = pubsub.DeclareAndBind(conn, routing.ExchangePerilDirect,
		routing.PauseKey+"."+uname, routing.PauseKey, pubsub.Transient)
	if err != nil {
		log.Fatalf("Could not bind connection: %v", err)
	}

	gs:= gamelogic.NewGameState(uname)
	for {
		inuser := gamelogic.GetInput()
		command := inuser[0]

		switch command {
		case "spawn":
			err = gs.CommandSpawn(inuser)
			if err != nil {
				fmt.Println(err)
				continue
			}
		case "move":
			_, err = gs.CommandMove(inuser)
			if err != nil {
				fmt.Println(err)
				continue
			}
		case "status":
			gs.CommandStatus()
		case "help":
			gamelogic.PrintClientHelp()
		case "spam":
			log.Println("Spamming not allowed yet")
		case "quit":
			gamelogic.PrintQuit()
			return
		default:
			fmt.Println("Command unknown")
		}
	}
}
