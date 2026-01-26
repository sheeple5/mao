package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net"

	"github.com/go-viper/mapstructure/v2"
)

type Player struct {
	PlayerID     string
	PlayerNumber int
	CanStartGame bool
	IsTurn       bool
}

func getPlayer(player *Player) {
	conn, err := net.Dial("tcp", "localhost:9090")
	if err != nil {
		panic(err)
	}
	defer conn.Close()

	writer := bufio.NewWriter(conn)
	_, err = writer.WriteString("{\"joinGame\": \"ABCD\"}\n")
	if err != nil {
		panic(err)
	}

	err = writer.Flush()
	if err != nil {
		panic(err)
	}

	netData, err := bufio.NewReader(conn).ReadString('\n')
	if err != nil {
		panic(err)
	}

	// Converts the received JSON data and converts it into a map
	var playerData map[string]any
	err = json.Unmarshal([]byte(netData), &playerData)
	if err != nil {
		fmt.Println(netData)
		panic(err)
	}

	// Loads the JSON data into the player struct using mapstructure
	err = mapstructure.Decode(playerData, &player)
	if err != nil {
		panic(err)
	}
}

func startGame(player Player) {
	conn, err := net.Dial("tcp", "localhost:9090")
	if err != nil {
		panic(err)
	}
	defer conn.Close()

	writer := bufio.NewWriter(conn)
	_, err = writer.WriteString(fmt.Sprintf("{\"playerID\": \"%s\", \"action\": \"startGame\"}\n", player.PlayerID))
	if err != nil {
		panic(err)
	}

	err = writer.Flush()
	if err != nil {
		panic(err)
	}

	netData, err := bufio.NewReader(conn).ReadString('\n')
	if err != nil {
		panic(err)
	}
	fmt.Println(netData)
}

func waitForGameStart() {
	conn, err := net.Dial("tcp", "localhost:9090")
	if err != nil {
		panic(err)
	}
	defer conn.Close()

	writer := bufio.NewWriter(conn)
	_, err = writer.WriteString("{\"action\": \"waitingStart\"}\n")
	if err != nil {
		panic(err)
	}

	err = writer.Flush()
	if err != nil {
		panic(err)
	}

	netData, err := bufio.NewReader(conn).ReadString('\n')
	if err != nil {
		panic(err)
	}
	fmt.Println(netData)
}

func drawHand(player *Player) {
	conn, err := net.Dial("tcp", "localhost:9090")
	if err != nil {
		panic(err)
	}
	defer conn.Close()

	writer := bufio.NewWriter(conn)
	_, err = writer.WriteString(fmt.Sprintf("{\"playerID\": \"%s\", \"action\": \"drawHand\"}\n", player.PlayerID))
	if err != nil {
		panic(err)
	}

	err = writer.Flush()
	if err != nil {
		panic(err)
	}

	netData, err := bufio.NewReader(conn).ReadString('\n')
	if err != nil {
		panic(err)
	}
	fmt.Println(netData)
}

func main() {
	var player Player

	var choice string
	fmt.Print("Press Y to join the game: ")
	fmt.Scan(&choice)

	if choice == "Y" {
		getPlayer(&player)
		fmt.Println(player)

		if player.CanStartGame {
			fmt.Print("Press Y to start the game: ")
			fmt.Scan(&choice)

			if choice == "Y" {
				startGame(player)
			}
		} else {
			fmt.Println("Waiting for game to start...")
			waitForGameStart()
		}

		drawHand(&player)

	} else {
		fmt.Println("Goodbye")
	}
}
