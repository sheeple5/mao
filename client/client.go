package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"os/signal"
	"regexp"
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

var (
	player   Player
	serverIP string
)

func getTerminalWidth() int {
	fd := int(os.Stdout.Fd())
	terminalWidth, _, err := term.GetSize(fd)
	if err != nil {
		panic(err)
	}
	return terminalWidth
}

func chooseServer() string {
	message := ""

	for {
		printHeader(message)

		fmt.Print("Enter a server IP (or exit to exit): ")
		fmt.Scan(&serverIP)

		if serverIP == "exit" {
			return serverIP
		}

		if getHealthCheck() {
			return serverIP
		} else {
			message = fmt.Sprintf("Could not connect to server %s", serverIP)
		}
	}
}

func sendData(payload string) string {
	conn, err := net.Dial("tcp", fmt.Sprintf("%s:9090", serverIP))
	if err != nil && payload == "{\"action\": \"healthCheck\"}\n" {
		return "{\"success\": false}"
	} else if err != nil {
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

func getHealthCheck() bool {
	netData := sendData("{\"action\": \"healthCheck\"}\n")
	type HealthDetails struct {
		Service string
		Success bool
	}

	var healthDetails HealthDetails
	var healthData map[string]any
	err := json.Unmarshal([]byte(netData), &healthData)
	if err != nil {
		fmt.Println(netData)
		panic(err)
	}

	// Loads the JSON data into the struct using mapstructure
	err = mapstructure.Decode(healthData, &healthDetails)
	if err != nil {
		panic(err)
	}

	if healthDetails.Service == "Mao Game" && healthDetails.Success {
		return true
	} else {
		return false
	}
}

func printRoomMenu(isPrivate bool, numRounds int, handSize int, message string) {
	terminalWidth := getTerminalWidth()
	printHeader("Room Options")

	var publicOption string
	if isPrivate {
		publicOption = "Private"
	} else {
		publicOption = "Public"
	}
	fmt.Printf("1. Set Public/Private: %s\n", publicOption)
	fmt.Printf("2. Change Number of Rounds: %d\n", numRounds)
	fmt.Printf("3. Change Initial Hand Size: %d\n", handSize)
	fmt.Println("--------------------")
	fmt.Println("C. Create room")
	fmt.Println("E. Return to main menu")
	fmt.Println(strings.Repeat("─", terminalWidth))

	if message != "" {
		fmt.Printf("%s%s\n", strings.Repeat(" ", (terminalWidth/2)-(len(message)/2)), message)
		fmt.Println(strings.Repeat("─", terminalWidth))
	}
}

func roomOptions(isPrivate *bool, numRounds *int, handSize *int) bool {
	choice := ""
	errorMessage := ""

	for {
		printRoomMenu(*isPrivate, *numRounds, *handSize, errorMessage)
		fmt.Printf("Choose a menu option: ")
		fmt.Scan(&choice)

		if !slices.Contains([]string{"1", "2", "3", "C", "E"}, choice) {
			errorMessage = "Invalid menu option selected."
			continue
		} else {
			errorMessage = ""
		}

		switch choice {
		case "C":
			return true
		case "E":
			return false
		case "1":
			var privateChoice string

			for {
				printHeader("Set Public/Private")
				fmt.Print("Choose Public or Private: ")
				fmt.Scan(&privateChoice)

				if privateChoice != "public" && privateChoice != "private" {
					continue
				}

				switch privateChoice {
				case "public":
					*isPrivate = false
				case "private":
					*isPrivate = true
				}
				break
			}
		case "2":
			var inputRounds string

			for {
				printHeader("Set Number of Rounds")
				fmt.Print("Enter the number of rounds to play: ")
				fmt.Scan(&inputRounds)

				intRounds, err := strconv.Atoi(inputRounds)
				if err != nil {
					continue
				}

				if intRounds >= 1 && intRounds <= 10 {
					*numRounds = intRounds
					break
				} else {
					continue
				}
			}
		case "3":
			var inputHand string

			for {
				printHeader("Set Initial Hand Size")
				fmt.Print("Enter the number of cards to start the game with: ")
				fmt.Scan(&inputHand)

				intHand, err := strconv.Atoi(inputHand)
				if err != nil {
					continue
				}

				if intHand >= 1 && intHand <= 15 {
					*handSize = intHand
					break
				} else {
					continue
				}
			}
		}

	}
}

func createRoom(player *Player, isPrivate bool, numRounds int, handSize int) {
	netData := sendData(fmt.Sprintf("{\"action\": \"createRoom\", \"isPrivate\": %t, \"numRounds\": %d, \"handSize\": %d}\n", isPrivate, numRounds, handSize))

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

func joinRoom(player *Player, roomCode string) bool {
	netData := sendData(fmt.Sprintf("{\"action\": \"joinRoom\", \"roomCode\": \"%s\"}\n", roomCode))
	type JoinDetails struct {
		Success bool
		Player  Player
	}

	var joinDetails JoinDetails
	var joinData map[string]any
	err := json.Unmarshal([]byte(netData), &joinData)
	if err != nil {
		fmt.Println(netData)
		panic(err)
	}

	// Loads the JSON data into the player struct using mapstructure
	err = mapstructure.Decode(joinData, &joinDetails)
	if err != nil {
		panic(err)
	}

	if !joinDetails.Success {
		return false
	} else {
		*player = joinDetails.Player

		player.RoomCode = roomCode
		return true
	}
}

func leaveRoom(player Player) {
	_ = sendData(fmt.Sprintf("{\"playerID\": \"%s\", \"roomCode\": \"%s\", \"action\": \"leaveRoom\"}\n", player.PlayerID, player.RoomCode))
}

func getRooms() string {
	netData := sendData("{\"action\": \"getRooms\"}\n")
	type RoomsDetails struct {
		Rooms string
	}

	var roomsDetails RoomsDetails
	var roomsData map[string]any
	err := json.Unmarshal([]byte(netData), &roomsData)
	if err != nil {
		fmt.Println(netData)
		panic(err)
	}

	// Loads the JSON data into the struct using mapstructure
	err = mapstructure.Decode(roomsData, &roomsDetails)
	if err != nil {
		panic(err)
	}
	return roomsDetails.Rooms
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

func waitTurn(player Player) []any {
	netData := sendData(fmt.Sprintf("{\"roomCode\": \"%s\", \"action\": \"waitTurn\"}\n", player.RoomCode))
	type GameWonDetails struct {
		WonRound           bool
		WinningRoundPlayer string
		WinningGamePlayer  string
		Round              int
		Stats              map[string]int
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

	return []any{gameWonDetails.WonRound, gameWonDetails.WinningRoundPlayer, gameWonDetails.WinningGamePlayer, gameWonDetails.Round, gameWonDetails.Stats}
}

func (player Player) playCard(playedCard string) []any {
	netData := sendData(fmt.Sprintf("{\"playerID\": \"%s\", \"roomCode\": \"%s\", \"action\": \"playCard\", \"card\": \"%s\"}\n", player.PlayerID, player.RoomCode, playedCard))
	type RuleCheckResults struct {
		RulesPassed bool
		CurrentHand string
		WonRound    bool
		Winner      string
		Round       int
		Stats       map[string]int
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

	return []any{ruleCheckResults.RulesPassed, ruleCheckResults.CurrentHand, ruleCheckResults.WonRound, ruleCheckResults.Winner, ruleCheckResults.Round, ruleCheckResults.Stats}
}

func addRule(player Player, newRule string) bool {
	netData := sendData(fmt.Sprintf("{\"playerID\": \"%s\", \"roomCode\": \"%s\", \"action\": \"addRule\", \"newRule\": \"%s\"}\n", player.PlayerID, player.RoomCode, newRule))
	type AddRuleResults struct {
		Success bool
	}

	var addRuleResults AddRuleResults
	var ruleData map[string]any
	err := json.Unmarshal([]byte(netData), &ruleData)
	if err != nil {
		fmt.Println(netData)
		panic(err)
	}

	// Loads the JSON data into the struct using mapstructure
	err = mapstructure.Decode(ruleData, &addRuleResults)
	if err != nil {
		panic(err)
	}

	return addRuleResults.Success
}

func getStats(player Player) map[string]int {
	netData := sendData(fmt.Sprintf("{\"playerID\": \"%s\", \"roomCode\": \"%s\", \"action\": \"getStats\"}\n", player.PlayerID, player.RoomCode))
	type Stats struct {
		Stats map[string]int
	}

	var stats Stats
	var statsData map[string]any
	err := json.Unmarshal([]byte(netData), &statsData)
	if err != nil {
		fmt.Println(netData)
		panic(err)
	}

	// Loads the JSON data into the struct using mapstructure
	err = mapstructure.Decode(statsData, &stats)
	if err != nil {
		panic(err)
	}

	return stats.Stats
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
	terminalWidth := getTerminalWidth()
	serverBanner := fmt.Sprintf("Current Server: %s", serverIP)
	serverPadding := strings.Repeat(" ", (terminalWidth/2)-(len(serverBanner)/2))

	printLogo()
	fmt.Println(strings.Repeat("─", terminalWidth))
	fmt.Printf("%s%s\n", serverPadding, serverBanner)
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
	fmt.Printf("%s%s\n", strings.Repeat(" ", (terminalWidth/2)-1), turnTopCard)
	fmt.Printf("%sYour Hand:\n", strings.Repeat(" ", (terminalWidth/2)-5))
	fmt.Printf("%s%s\n", strings.Repeat(" ", (terminalWidth/2)-(len(handList)/2)), handList)
	fmt.Println(strings.Repeat("─", terminalWidth))

	if message != "" {
		fmt.Printf("%s%s\n", strings.Repeat(" ", (terminalWidth/2)-(len(message)/2)), message)
		fmt.Println(strings.Repeat("─", terminalWidth))
	}
}

func printStats(stats map[string]int) {
	terminalWidth := getTerminalWidth()
	for playerNumber, numWins := range stats {
		intPlayerNumber, err := strconv.Atoi(playerNumber)
		if err != nil {
			panic(err)
		}

		fmt.Printf("Player %d: %d wins\n", intPlayerNumber+1, numWins)
	}
	fmt.Println(strings.Repeat("─", terminalWidth))
}

func main() {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	go func() {
		for sig := range c {
			if sig.String() == "interrupt" {
				leaveRoom(player)
				os.Exit(0)
			}
		}
	}()

	var handList string

	for {
		if serverIP == "" {
			chooseServer()
		}

		if serverIP == "exit" {
			return
		}

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
			isPrivate := false
			numRounds := 5
			handSize := 7

			doCreate := roomOptions(&isPrivate, &numRounds, &handSize)

			if doCreate {
				createRoom(&player, isPrivate, numRounds, handSize)
			} else {
				continue
			}
		case "2":
			exitBreak := false
			for {
				roomsList := getRooms()
				printHeader(fmt.Sprintf("Open Rooms: %s", roomsList))

				var roomCode string
				fmt.Print("Enter a room code: ")
				fmt.Scan(&roomCode)

				match, err := regexp.MatchString(`[A-Z]{4}`, roomCode)
				if err != nil {
					panic(err)
				}

				if roomCode == "exit" {
					exitBreak = true
					break
				} else if !match {
					continue
				}
				joinedSuccessfully := joinRoom(&player, roomCode)

				if joinedSuccessfully {
					break
				}
			}

			if exitBreak {
				continue
			}
		case "3":
			serverIP = ""
			continue
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

		gameMessage := ""
		for {
			// Request turn
			turnDetails := requestTurn(player)
			turnPlayerNumber := turnDetails[0]
			turnTopCard := turnDetails[1]

			adjustedTurnPlayerNumber, _ := strconv.Atoi(turnPlayerNumber)
			adjustedTurnPlayerNumber += 1
			// If my player number equals the returned turn player, take turn. Else, wait for turn and display current hand/top card

			printTurn(player, handList, adjustedTurnPlayerNumber, turnTopCard, gameMessage)
			if strconv.Itoa(player.PlayerNumber) == turnPlayerNumber {
				var playedCard string
				for {
					fmt.Print("Choose a card to play (or 'draw' to draw): ")
					fmt.Scan(&playedCard)

					if !slices.Contains(strings.Split(handList, ", "), playedCard) && playedCard != "draw" && playedCard != "leave" {
						printTurn(player, handList, adjustedTurnPlayerNumber, turnTopCard, "Error: The card you entered is not in hand.")
					} else {
						break
					}
				}

				if playedCard == "leave" {
					leaveRoom(player)
					break
				}

				playResults := player.playCard(playedCard)
				if winner, ok := playResults[3].(string); ok {
					if winner != "" {
						if stats, ok := playResults[5].(map[string]int); ok {
							if winner == strconv.Itoa(player.PlayerNumber) {
								printHeader("You won the game!")
							} else {
								adjustedWinner, err := strconv.Atoi(winner)
								if err != nil {
									panic(err)
								}
								adjustedWinner += 1
								printHeader(fmt.Sprintf("Game Over. Player %d wins!", adjustedWinner))
							}
							printStats(stats)
							var confirm string
							fmt.Print("Press Y to leave the game: ")
							fmt.Scan(&confirm)
							break
						}
					}
				}

				if wonRound, ok := playResults[2].(bool); ok {
					if wonRound {
						if round, ok := playResults[4].(int); ok {
							printHeader(fmt.Sprintf("You won Round %d!", round))
							printStats(getStats(player))

							reader := bufio.NewReader(os.Stdin)
							fmt.Print("As your reward, describe a new rule to add to the game: ")
							newRule, _ := reader.ReadString('\n')
							newRule = strings.TrimSpace(newRule)

							if newRule == "leave" {
								leaveRoom(player)
								break
							}

							if addRule(player, newRule) {
								gameMessage = "Rule added successfully."
							} else {
								gameMessage = "Your rule could not be added."
							}
							handList = startGame(player)
							continue
						}
					}
				}

				if currentHand, ok := playResults[1].(string); ok {
					if playSucceeded, ok := playResults[0].(bool); ok {
						if playSucceeded {
							handList = currentHand
							gameMessage = ""
						} else {
							handList = currentHand
							gameMessage = "You broke a rule and incurred a penalty."
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
				playResults := waitTurn(player)

				if winner, ok := playResults[2].(string); ok {
					if winner != "" {
						if stats, ok := playResults[4].(map[string]int); ok {
							if winner == strconv.Itoa(player.PlayerNumber) {
								printHeader("You won the game!")
							} else {
								adjustedWinner, err := strconv.Atoi(winner)
								if err != nil {
									panic(err)
								}
								adjustedWinner += 1
								printHeader(fmt.Sprintf("Game Over. Player %d wins!", adjustedWinner))
							}
							printStats(stats)
							var confirm string
							fmt.Print("Press Y to leave the game: ")
							fmt.Scan(&confirm)
							break
						}
					}
				}

				if wonRound, ok := playResults[0].(bool); ok {
					if wonRound {
						if winningPlayer, ok := playResults[1].(string); ok {
							if round, ok := playResults[3].(int); ok {
								message := fmt.Sprintf("Player %s won Round %d.", winningPlayer, round)
								printHeader(message)
								printStats(getStats(player))

								fmt.Printf("Waiting for player %s to add a new rule...\n", winningPlayer)
								handList = waitForGameStart(player)
								continue
							}
						}
					}
				}
			}
		}
	}
}
