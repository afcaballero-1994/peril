package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"

	"github.com/rabbitmq/amqp091-go"
)

func main() {
	const connectionString string = "amqp://guest:guest@localhost:5672/"
	cotion, err := amqp091.Dial(connectionString)
	if err != nil {
		log.Fatalf("Could no create connection: %v", err)
	}
	defer cotion.Close()
	fmt.Println("Starting Peril server...")
	log.Println("Connection successful")

	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, os.Interrupt)
	<-signalChan
	log.Println("\n Shutting down Connection")
}
