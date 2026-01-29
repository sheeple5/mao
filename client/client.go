package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net"
	"slices"
	"strconv"
	"strings"

	"github.com/go-viper/mapstructure/v2"
)

type Player struct {
	PlayerID     string
	PlayerNumber int
	CanStartGame bool
	RoomCode     string
}

func (player Player) playCard(playedCard string) []any {
	conn, err := net.Dial("tcp", "localhost:9090")
	if err != nil {
		panic(err)
	}
	defer conn.Close()

	writer := bufio.NewWriter(conn)
	_, err = writer.WriteString(fmt.Sprintf("{\"playerID\": \"%s\", \"roomCode\": \"%s\", \"playCard\": \"%s\"}\n", player.PlayerID, player.RoomCode, playedCard))
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

	type RuleCheckResults struct {
		RulesPassed bool
		CurrentHand string
		WonGame     bool
	}
	// Converts the received JSON data and converts it into a map
	var ruleCheckResults RuleCheckResults
	var ruleData map[string]any
	err = json.Unmarshal([]byte(netData), &ruleData)
	if err != nil {
		fmt.Println(netData)
		panic(err)
	}

	// Loads the JSON data into the player struct using mapstructure
	err = mapstructure.Decode(ruleData, &ruleCheckResults)
	if err != nil {
		panic(err)
	}

	return []any{ruleCheckResults.RulesPassed, ruleCheckResults.CurrentHand, ruleCheckResults.WonGame}
}

func createGame(player *Player) {
	conn, err := net.Dial("tcp", "localhost:9090")
	if err != nil {
		panic(err)
	}
	defer conn.Close()

	writer := bufio.NewWriter(conn)
	_, err = writer.WriteString("{\"action\": \"createRoom\"}\n")
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

func joinGame(player *Player, roomCode string) {
	conn, err := net.Dial("tcp", "localhost:9090")
	if err != nil {
		panic(err)
	}
	defer conn.Close()

	writer := bufio.NewWriter(conn)
	_, err = writer.WriteString(fmt.Sprintf("{\"joinRoom\": \"%s\"}\n", roomCode))
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
	player.RoomCode = roomCode
}

func startGame(player Player) string {
	conn, err := net.Dial("tcp", "localhost:9090")
	if err != nil {
		panic(err)
	}
	defer conn.Close()

	writer := bufio.NewWriter(conn)
	_, err = writer.WriteString(fmt.Sprintf("{\"playerID\": \"%s\", \"roomCode\": \"%s\", \"action\": \"startGame\"}\n", player.PlayerID, player.RoomCode))
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

	type InitialHandDetails struct {
		InitialHand string
	}
	var initialHandDetails InitialHandDetails
	var handData map[string]any
	err = json.Unmarshal([]byte(netData), &handData)
	if err != nil {
		fmt.Println(netData)
		panic(err)
	}

	// Loads the JSON data into the drawAction struct using mapstructure
	err = mapstructure.Decode(handData, &initialHandDetails)
	if err != nil {
		panic(err)
	}
	return initialHandDetails.InitialHand
}

func waitForGameStart(player Player) string {
	conn, err := net.Dial("tcp", "localhost:9090")
	if err != nil {
		panic(err)
	}
	defer conn.Close()

	writer := bufio.NewWriter(conn)
	fmt.Println(player.RoomCode)
	_, err = writer.WriteString(fmt.Sprintf("{\"playerID\": \"%s\", \"roomCode\": \"%s\", \"action\": \"waitingStart\"}\n", player.PlayerID, player.RoomCode))
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

	type InitialHandDetails struct {
		InitialHand string
	}
	var initialHandDetails InitialHandDetails
	var handData map[string]any
	err = json.Unmarshal([]byte(netData), &handData)
	if err != nil {
		fmt.Println(netData)
		panic(err)
	}

	// Loads the JSON data into the drawAction struct using mapstructure
	err = mapstructure.Decode(handData, &initialHandDetails)
	if err != nil {
		panic(err)
	}
	return initialHandDetails.InitialHand
}

func requestTurn(player Player) []string {
	conn, err := net.Dial("tcp", "localhost:9090")
	if err != nil {
		panic(err)
	}
	defer conn.Close()

	writer := bufio.NewWriter(conn)
	_, err = writer.WriteString(fmt.Sprintf("{\"roomCode\": \"%s\", \"action\": \"requestTurn\"}\n", player.RoomCode))
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

	type RequestTurnDetails struct {
		PlayerNumber int
		TopCard      string
	}
	var requestTurnDetails RequestTurnDetails
	var turnData map[string]any
	err = json.Unmarshal([]byte(netData), &turnData)
	if err != nil {
		fmt.Println(netData)
		panic(err)
	}

	// Loads the JSON data into the drawAction struct using mapstructure
	err = mapstructure.Decode(turnData, &requestTurnDetails)
	if err != nil {
		panic(err)
	}

	fmt.Println(requestTurnDetails.PlayerNumber)
	return []string{strconv.Itoa(requestTurnDetails.PlayerNumber), requestTurnDetails.TopCard}
}

func waitTurn(player Player) string {
	conn, err := net.Dial("tcp", "localhost:9090")
	if err != nil {
		panic(err)
	}
	defer conn.Close()

	writer := bufio.NewWriter(conn)
	_, err = writer.WriteString(fmt.Sprintf("{\"roomCode\": \"%s\", \"action\": \"waitTurn\"}\n", player.RoomCode))
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

	type GameWonDetails struct {
		WonGame       bool
		WinningPlayer string
	}
	var gameWonDetails GameWonDetails
	var gameData map[string]any
	err = json.Unmarshal([]byte(netData), &gameData)
	if err != nil {
		fmt.Println(netData)
		panic(err)
	}

	// Loads the JSON data into the drawAction struct using mapstructure
	err = mapstructure.Decode(gameData, &gameWonDetails)
	if err != nil {
		panic(err)
	}

	if gameWonDetails.WonGame {
		return gameWonDetails.WinningPlayer
	} else {
		return ""
	}
}

func main() {
	var player Player
	var handList string

	var choice string
	fmt.Print("Press C to create the game or J to join the game: ")
	fmt.Scan(&choice)

	if choice == "C" {
		createGame(&player)
	} else if choice == "J" {
		var roomCode string
		fmt.Print("Enter a room code: ")
		fmt.Scan(&roomCode)
		joinGame(&player, roomCode)
	}

	fmt.Println(player)

	if player.CanStartGame {
		fmt.Print("Press Y to start the game: ")
		fmt.Scan(&choice)

		if choice == "Y" {
			handList = startGame(player)
		}
	} else {
		fmt.Println("Waiting for game to start...")
		handList = waitForGameStart(player)
	}

	for {
		// Request turn
		turnDetails := requestTurn(player)
		turnPlayerNumber := turnDetails[0]
		turnTopCard := turnDetails[1]

		adjustedTurnPlayerNumber, _ := strconv.Atoi(turnPlayerNumber)
		adjustedTurnPlayerNumber += 1
		// If my player number equals the returned turn player, take turn. Else, wait for turn and display current hand/top card
		if strconv.Itoa(player.PlayerNumber) == turnPlayerNumber {
			var playedCard string
			fmt.Printf("Player %d's turn...\n", adjustedTurnPlayerNumber)
			fmt.Printf("Current Card: %s\n", turnTopCard)
			fmt.Printf("My Hand: %s\n", handList)

			for {
				fmt.Print("Choose a card to play (or 'draw' to draw): ")
				fmt.Scan(&playedCard)

				if !slices.Contains(strings.Split(handList, ", "), playedCard) && playedCard != "draw" {
					fmt.Println("Error: The card you entered is not in hand.")
				} else {
					break
				}
			}

			playResults := player.playCard(playedCard)
			if wonGame, ok := playResults[2].(bool); ok {
				if wonGame {
					fmt.Println("Congratulations, you win!")
					break
				}
			}

			if currentHand, ok := playResults[1].(string); ok {
				if playSucceeded, ok := playResults[0].(bool); ok {
					if playSucceeded {
						fmt.Println("VALID")
						fmt.Println(currentHand)
						handList = currentHand
					} else {
						fmt.Println("INVALID")
						fmt.Println(currentHand)
						handList = currentHand
					}
				}
			}
			// send card to play. can validate user actually has card on server side now cause it's handled over there
			// once card has actually been played and is valid, respond back with new handlist for next turn
		} else {
			fmt.Printf("Player %d's turn...\n", adjustedTurnPlayerNumber)
			fmt.Printf("Current Card: %s\n", turnTopCard)
			fmt.Printf("My Hand: %s\n", handList)
			fmt.Println("Waiting for turn...")
			winningPlayer := waitTurn(player)

			if winningPlayer != "" {
				fmt.Printf("Game over! Player %s wins.", winningPlayer)
				break
			}
		}
	}
}
