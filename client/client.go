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
}

func (player Player) playCard(playedCard string) []any {
	conn, err := net.Dial("tcp", "localhost:9090")
	if err != nil {
		panic(err)
	}
	defer conn.Close()

	writer := bufio.NewWriter(conn)
	_, err = writer.WriteString(fmt.Sprintf("{\"playerID\": \"%s\", \"playCard\": \"%s\"}\n", player.PlayerID, playedCard))
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

	return []any{ruleCheckResults.RulesPassed, ruleCheckResults.CurrentHand}
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

func startGame(player Player) string {
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
	_, err = writer.WriteString(fmt.Sprintf("{\"playerID\": \"%s\", \"action\": \"waitingStart\"}\n", player.PlayerID))
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

func requestTurn() []string {
	conn, err := net.Dial("tcp", "localhost:9090")
	if err != nil {
		panic(err)
	}
	defer conn.Close()

	writer := bufio.NewWriter(conn)
	_, err = writer.WriteString("{\"action\": \"requestTurn\"}\n")
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

	return []string{strconv.Itoa(requestTurnDetails.PlayerNumber), requestTurnDetails.TopCard}
}

func waitTurn() {
	conn, err := net.Dial("tcp", "localhost:9090")
	if err != nil {
		panic(err)
	}
	defer conn.Close()

	writer := bufio.NewWriter(conn)
	_, err = writer.WriteString("{\"action\": \"waitTurn\"}\n")
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
	var handList string

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
				handList = startGame(player)
			}
		} else {
			fmt.Println("Waiting for game to start...")
			handList = waitForGameStart(player)
		}

		for {
			// Request turn
			turnDetails := requestTurn()
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
				waitTurn()
			}

			// request turn
			// if playerID = my ID, branch into playing a card
			// otherwise, send an action: waitTurn
			// after turn played, server will respond to both playTurn (or whatever) waitTurn with a game status: win, lose, or continue
			//  - if lose, break and say who won
			//  - if win, break and show congrats or something
			//  - if continue, have everyone send another request turn and go from there
		}
		// On game end, send winning playe prompt to request a new rule

	} else {
		fmt.Println("Goodbye")
	}
}
