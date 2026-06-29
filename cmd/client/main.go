package main

import (
	"fmt"
	"log"
	"time"

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
	fmt.Println("Peril game client connected to rabbit")

	publishCh, err := conn.Channel()
	if err != nil {
		log.Fatalf("could not create channel: %v", err)
	}
	uname, err := gamelogic.ClientWelcome()
	if err != nil {
		log.Fatalf("Could not get username: %v", err)
	}

	gs := gamelogic.NewGameState(uname)
	err = pubsub.SubscribeJSON(
		conn,
		string(routing.ExchangePerilTopic),
		string(routing.ArmyMovesPrefix)+"."+gs.GetUsername(),
		string(routing.ArmyMovesPrefix)+".*",
		pubsub.Transient,
		handlerMove(gs, publishCh),
	)

	err = pubsub.SubscribeJSON(conn, routing.ExchangePerilDirect,
		routing.PauseKey+"."+gs.GetUsername(),
		routing.PauseKey, pubsub.Transient, handlerPause(gs))
	if err != nil {
		log.Fatalf("could not subscribe to pause: %v", err)
	}
	err = pubsub.SubscribeJSON(
		conn,
		routing.ExchangePerilTopic,
		"war",
		routing.WarRecognitionsPrefix+".*",
		pubsub.Durable,
		handlerWar(gs, publishCh),
	)

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
			mv, err := gs.CommandMove(inuser)
			if err != nil {
				fmt.Println(err)
				continue
			}
			err = pubsub.PublishJSON(
				publishCh,
				string(routing.ExchangePerilTopic),
				routing.ArmyMovesPrefix+"."+mv.Player.Username,
				mv,
			)
			if err != nil {
				log.Printf("Could not publish move: %v", err)
				continue
			}
			log.Printf("Move %v units to %s\n", len(mv.Units), mv.ToLocation)
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

func handlerPause(gs *gamelogic.GameState) func(routing.PlayingState) pubsub.Acktype {
	return func(ps routing.PlayingState) pubsub.Acktype {
		defer fmt.Print("> ")
		gs.HandlePause(ps)
		return pubsub.Ack
	}
}

func handlerMove(gs *gamelogic.GameState, ch *amqp091.Channel) func(gamelogic.ArmyMove) pubsub.Acktype {
	return func(move gamelogic.ArmyMove) pubsub.Acktype {
		defer fmt.Print("> ")
		out := gs.HandleMove(move)
		switch out {
		case gamelogic.MoveOutComeSafe:
			return pubsub.Ack
		case gamelogic.MoveOutcomeMakeWar:
			err := pubsub.PublishJSON(
				ch,
				routing.ExchangePerilTopic,
				routing.WarRecognitionsPrefix+ "." +gs.GetUsername(),
				gamelogic.RecognitionOfWar{
					Attacker: move.Player,
					Defender: gs.GetPlayerSnap(),
				},
			)
			if err != nil {
				return pubsub.NackRequeue
			} else {
				return pubsub.Ack
			}
			
		case gamelogic.MoveOutcomeSamePlayer:
			return pubsub.NackDiscard
		default:
			return pubsub.NackDiscard
		}
	}
}

func handlerWar(gs *gamelogic.GameState, ch *amqp091.Channel) func(gamelogic.RecognitionOfWar) pubsub.Acktype {

	return func(rw gamelogic.RecognitionOfWar) pubsub.Acktype {
		defer fmt.Print("> ")
		out, winner, loser := gs.HandleWar(rw)
		var message string
		switch out {
		case gamelogic.WarOutcomeNotInvolved:
			return pubsub.NackRequeue
		case gamelogic.WarOutcomeNoUnits:
			return pubsub.NackDiscard
		case gamelogic.WarOutcomeOpponentWon:
			message = fmt.Sprintf("{%s} won a war against {%s}\n", winner, loser)
			err := printGob(
				ch,
				message,
				gs.GetUsername(),
			)
			if err != nil {
				return pubsub.NackRequeue
			} else {
				return pubsub.Ack
			}
			
		case gamelogic.WarOutcomeYouWon:
			message = fmt.Sprintf("{%s} won a war against {%s}\n", winner, loser)
			err := printGob(
				ch,
				message,
				gs.GetUsername(),
			)
			if err != nil {
				return pubsub.NackRequeue
			} else {
				return pubsub.Ack
			}
		case gamelogic.WarOutcomeDraw:
			message = fmt.Sprintf("A war between {%s} and {%s} resulted in a draw\n", winner, loser)
			err := printGob(
				ch,
				message,
				gs.GetUsername(),
			)
			if err != nil {
				return pubsub.NackRequeue
			} else {
				return pubsub.Ack
			}
		default:
			fmt.Println("Error: no valid outcome")
			return pubsub.NackDiscard
		}
	}
}

func printGob(ch *amqp091.Channel, message, user string) error{
	data := routing.GameLog{
		CurrentTime: time.Now(),
		Message: message,
		Username: user,
	}
	log.Printf(message)
	err := pubsub.PublishGob(ch,
		routing.ExchangePerilTopic,
		routing.GameLogSlug + "." + user,
		data)
	return err
}
