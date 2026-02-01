package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"slices"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/go-viper/mapstructure/v2"
	"golang.org/x/term"
)

type Player struct {
	PlayerID     string
	PlayerNumber int
	CanStartGame bool
	RoomCode     string
}

func getTerminalWidth() int {
	fd := int(os.Stdout.Fd())
	terminalWidth, _, err := term.GetSize(fd)
	if err != nil {
		panic(err)
	}
	return terminalWidth
}

func sendData(payload string) string {
	conn, err := net.Dial("tcp", "localhost:9090")
	if err != nil {
		panic(err)
	}

	defer func() {
		if closeErr := conn.Close(); closeErr != nil {
			panic(closeErr)
		}
	}()

	writer := bufio.NewWriter(conn)
	_, err = writer.WriteString(payload)
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

	return netData
}

func createGame(player *Player) {
	netData := sendData(("{\"action\": \"createRoom\"}\n"))

	var playerData map[string]any
	err := json.Unmarshal([]byte(netData), &playerData)
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
	netData := sendData(fmt.Sprintf("{\"action\": \"joinRoom\", \"roomCode\": \"%s\"}\n", roomCode))

	var playerData map[string]any
	err := json.Unmarshal([]byte(netData), &playerData)
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
	netData := sendData(fmt.Sprintf("{\"playerID\": \"%s\", \"roomCode\": \"%s\", \"action\": \"startGame\"}\n", player.PlayerID, player.RoomCode))
	type InitialHandDetails struct {
		InitialHand string
	}

	var initialHandDetails InitialHandDetails
	var handData map[string]any
	err := json.Unmarshal([]byte(netData), &handData)
	if err != nil {
		fmt.Println(netData)
		panic(err)
	}

	// Loads the JSON data into the struct using mapstructure
	err = mapstructure.Decode(handData, &initialHandDetails)
	if err != nil {
		panic(err)
	}
	return initialHandDetails.InitialHand
}

func waitForGameStart(player Player) string {
	netData := sendData(fmt.Sprintf("{\"playerID\": \"%s\", \"roomCode\": \"%s\", \"action\": \"waitingStart\"}\n", player.PlayerID, player.RoomCode))
	type InitialHandDetails struct {
		InitialHand string
	}

	var initialHandDetails InitialHandDetails
	var handData map[string]any
	err := json.Unmarshal([]byte(netData), &handData)
	if err != nil {
		fmt.Println(netData)
		panic(err)
	}

	// Loads the JSON data into the struct using mapstructure
	err = mapstructure.Decode(handData, &initialHandDetails)
	if err != nil {
		panic(err)
	}
	return initialHandDetails.InitialHand
}

func requestTurn(player Player) []string {
	netData := sendData(fmt.Sprintf("{\"roomCode\": \"%s\", \"action\": \"requestTurn\"}\n", player.RoomCode))
	type RequestTurnDetails struct {
		PlayerNumber int
		TopCard      string
	}

	var requestTurnDetails RequestTurnDetails
	var turnData map[string]any
	err := json.Unmarshal([]byte(netData), &turnData)
	if err != nil {
		fmt.Println(netData)
		panic(err)
	}

	// Loads the JSON data into the struct using mapstructure
	err = mapstructure.Decode(turnData, &requestTurnDetails)
	if err != nil {
		panic(err)
	}

	fmt.Println(requestTurnDetails.PlayerNumber)
	return []string{strconv.Itoa(requestTurnDetails.PlayerNumber), requestTurnDetails.TopCard}
}

func waitTurn(player Player) string {
	netData := sendData(fmt.Sprintf("{\"roomCode\": \"%s\", \"action\": \"waitTurn\"}\n", player.RoomCode))
	type GameWonDetails struct {
		WonGame       bool
		WinningPlayer string
	}

	var gameWonDetails GameWonDetails
	var gameData map[string]any
	err := json.Unmarshal([]byte(netData), &gameData)
	if err != nil {
		fmt.Println(netData)
		panic(err)
	}

	// Loads the JSON data into the struct using mapstructure
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

func (player Player) playCard(playedCard string) []any {
	netData := sendData(fmt.Sprintf("{\"playerID\": \"%s\", \"roomCode\": \"%s\", \"action\": \"playCard\", \"card\": \"%s\"}\n", player.PlayerID, player.RoomCode, playedCard))
	type RuleCheckResults struct {
		RulesPassed bool
		CurrentHand string
		WonGame     bool
	}

	var ruleCheckResults RuleCheckResults
	var ruleData map[string]any
	err := json.Unmarshal([]byte(netData), &ruleData)
	if err != nil {
		fmt.Println(netData)
		panic(err)
	}

	// Loads the JSON data into the struct using mapstructure
	err = mapstructure.Decode(ruleData, &ruleCheckResults)
	if err != nil {
		panic(err)
	}

	return []any{ruleCheckResults.RulesPassed, ruleCheckResults.CurrentHand, ruleCheckResults.WonGame}
}

func printLogo() { // Should also introduce height variability as well
	titleLarge := `          _____                    _____                   _______         
         /\    \                  /\    \                 /::\    \        
        /::\____\                /::\    \               /::::\    \       
       /::::|   |               /::::\    \             /::::::\    \      
      /:::::|   |              /::::::\    \           /::::::::\    \     
     /::::::|   |             /:::/\:::\    \         /:::/~~\:::\    \    
    /:::/|::|   |            /:::/__\:::\    \       /:::/    \:::\    \   
   /:::/ |::|   |           /::::\   \:::\    \     /:::/    / \:::\    \  
  /:::/  |::|___|______    /::::::\   \:::\    \   /:::/____/   \:::\____\ 
 /:::/   |::::::::\    \  /:::/\:::\   \:::\    \ |:::|    |     |:::|    |
/:::/    |:::::::::\____\/:::/  \:::\   \:::\____\|:::|____|     |:::|    |
\::/    / ~~~~~/:::/    /\::/    \:::\  /:::/    / \:::\    \   /:::/    / 
 \/____/      /:::/    /  \/____/ \:::\/:::/    /   \:::\    \ /:::/    /  
             /:::/    /            \::::::/    /     \:::\    /:::/    /   
            /:::/    /              \::::/    /       \:::\__/:::/    /    
           /:::/    /               /:::/    /         \::::::::/    /     
          /:::/    /               /:::/    /           \::::::/    /      
         /:::/    /               /:::/    /             \::::/    /       
        /:::/    /               /:::/    /               \::/____/        
        \::/    /                \::/    /                 ~~              
         \/____/                  \/____/                                  
                                                                           `

	titleMedium := `                                                      
     ______  _______         _____           _____    
    |      \/       \    ___|\    \     ____|\    \   
   /          /\     \  /    /\    \   /     /\    \  
  /     /\   / /\     ||    |  |    | /     /  \    \ 
 /     /\ \_/ / /    /||    |__|    ||     |    |    |
|     |  \|_|/ /    / ||    .--.    ||     |    |    |
|     |       |    |  ||    |  |    ||\     \  /    /|
|\____\       |____|  /|____|  |____|| \_____\/____/ |
| |    |      |    | / |    |  |    | \ |    ||    | /
 \|____|      |____|/  |____|  |____|  \|____||____|/ 
    \(          )/       \(      )/       \(    )/    
     '          '         '      '         '    '     
                                                      `

	titleSmall := ` _____ ______   ________  ________     
|\   _ \  _   \|\   __  \|\   __  \    
\ \  \\\__\ \  \ \  \|\  \ \  \|\  \   
 \ \  \\|__| \  \ \   __  \ \  \\\  \  
  \ \  \    \ \  \ \  \ \  \ \  \\\  \ 
   \ \__\    \ \__\ \__\ \__\ \_______\
    \|__|     \|__|\|__|\|__|\|_______|
                                       `

	titleTiny := `░█▄█░█▀█░█▀█
░█░█░█▀█░█░█
░▀░▀░▀░▀░▀▀▀`

	titleLargeLines := strings.Split(titleLarge, "\n")
	titleMediumLines := strings.Split(titleMedium, "\n")
	titleSmallLines := strings.Split(titleSmall, "\n")
	titleTinyLines := strings.Split(titleTiny, "\n")

	var printTitle []string
	terminalWidth := getTerminalWidth()
	if terminalWidth > len(titleLargeLines[0]) {
		printTitle = titleLargeLines
	} else if terminalWidth > len(titleMediumLines[0]) {
		printTitle = titleMediumLines
	} else if terminalWidth > len(titleSmallLines[0]) {
		printTitle = titleSmallLines
	} else {
		printTitle = titleTinyLines
	}

	fmt.Print("\033[H\033[2J")
	for _, line := range printTitle {
		titlePadding := strings.Repeat(" ", (terminalWidth/2)-(utf8.RuneCountInString(line)/2))
		fmt.Printf("%s%s\n", titlePadding, line)
	}
}

func printHeader(message string) {
	printLogo()

	terminalWidth := getTerminalWidth()
	fmt.Println(strings.Repeat("─", terminalWidth))

	if message != "" {
		fmt.Printf("%s%s\n", strings.Repeat(" ", (terminalWidth/2)-(len(message)/2)), message)
		fmt.Println(strings.Repeat("─", terminalWidth))
	}
}

func printMainMenu() {
	printLogo()

	terminalWidth := getTerminalWidth()
	fmt.Println("─┬" + strings.Repeat("─", terminalWidth-2))

	menuOptions := []string{"Create a new room", "Join a room", "Exit"}
	for i, menuOption := range menuOptions {
		titlePadding := strings.Repeat(" ", (terminalWidth/2)-(utf8.RuneCountInString(menuOption)/2)-2)
		fmt.Printf("%d│%s%s\n", i+1, titlePadding, menuOption)
	}
	fmt.Println("─┴" + strings.Repeat("─", terminalWidth-2))
}

func printTurn(player Player, handList string, adjustedPlayerNumber int, turnTopCard string, message string) {
	printHeader("")

	terminalWidth := getTerminalWidth()
	if adjustedPlayerNumber-1 == player.PlayerNumber {
		fmt.Printf("%sYOUR TURN\n", strings.Repeat(" ", (terminalWidth/2)-4))
	} else {
		playerTitle := fmt.Sprintf("Player %d's turn", adjustedPlayerNumber)
		fmt.Printf("%s%s\n", strings.Repeat(" ", (terminalWidth/2)-(len(playerTitle)/2)), playerTitle)
	}
	fmt.Println(strings.Repeat("─", terminalWidth))
	fmt.Printf("%sCurrent Card:\n", strings.Repeat(" ", (terminalWidth/2)-6))
	fmt.Printf("%s%s:\n", strings.Repeat(" ", (terminalWidth/2)-1), turnTopCard)
	fmt.Printf("%sYour Hand:\n", strings.Repeat(" ", (terminalWidth/2)-5))
	fmt.Printf("%s%s\n", strings.Repeat(" ", (terminalWidth/2)-(len(handList)/2)), handList)
	fmt.Println(strings.Repeat("─", terminalWidth))

	if message != "" {
		fmt.Printf("%s%s\n", strings.Repeat(" ", (terminalWidth/2)-(len(message)/2)), message)
		fmt.Println(strings.Repeat("─", terminalWidth))
	}
}

func main() {
	var player Player
	var handList string

	printMainMenu()
	var choice string
	fmt.Print("Choose a menu option: ")
	for {
		fmt.Scan(&choice)

		if !slices.Contains([]string{"1", "2", "3"}, choice) {
			printMainMenu()
			fmt.Print("Error - please choose a valid menu option: ")
		} else {
			break
		}
	}

	switch choice {
	case "1":
		createGame(&player)
	case "2":
		printHeader("") // Should print available rooms eventually

		var roomCode string
		fmt.Print("Enter a room code: ")
		fmt.Scan(&roomCode)
		joinGame(&player, roomCode)
	case "3":
		return
	}

	if player.CanStartGame {
		for {
			message := fmt.Sprintf("Room Code: %s", player.RoomCode)
			printHeader(message)
			fmt.Print("Press Y to start the game: ") // Should include an option to back out. Can I also display number of users joined?
			fmt.Scan(&choice)

			if choice == "Y" {
				handList = startGame(player)
				break
			}
		}
	} else {
		printHeader("Waiting for game to start...")
		handList = waitForGameStart(player)
	}

	penaltyMessage := ""
	for {
		// Request turn
		turnDetails := requestTurn(player)
		turnPlayerNumber := turnDetails[0]
		turnTopCard := turnDetails[1]

		adjustedTurnPlayerNumber, _ := strconv.Atoi(turnPlayerNumber)
		adjustedTurnPlayerNumber += 1
		// If my player number equals the returned turn player, take turn. Else, wait for turn and display current hand/top card

		printTurn(player, handList, adjustedTurnPlayerNumber, turnTopCard, penaltyMessage)
		if strconv.Itoa(player.PlayerNumber) == turnPlayerNumber {
			var playedCard string
			for {
				fmt.Print("Choose a card to play (or 'draw' to draw): ")
				fmt.Scan(&playedCard)

				if !slices.Contains(strings.Split(handList, ", "), playedCard) && playedCard != "draw" {
					printTurn(player, handList, adjustedTurnPlayerNumber, turnTopCard, "Error: The card you entered is not in hand.")
				} else {
					break
				}
			}

			playResults := player.playCard(playedCard)
			if wonGame, ok := playResults[2].(bool); ok {
				if wonGame {
					printHeader("Congratulations, you win!")
					break
				}
			}

			if currentHand, ok := playResults[1].(string); ok {
				if playSucceeded, ok := playResults[0].(bool); ok {
					if playSucceeded {
						handList = currentHand
						penaltyMessage = ""
					} else {
						handList = currentHand
						penaltyMessage = "You broke a rule and incurred a penalty."
					}
				}
			}
			// send card to play. can validate user actually has card on server side now cause it's handled over there
			// once card has actually been played and is valid, respond back with new handlist for next turn
		} else {
			message := "Waiting for turn..."
			terminalWidth := getTerminalWidth()
			fmt.Printf("%s%s\n", strings.Repeat(" ", (terminalWidth/2)-(len(message)/2)), message)
			fmt.Println(strings.Repeat("─", terminalWidth))
			winningPlayer := waitTurn(player)

			if winningPlayer != "" {
				message := fmt.Sprintf("Game over! Player %s wins.", winningPlayer)
				printHeader(message)
				break
			}
		}
	}
}
