package main

import (
	"bufio"
	"encoding/json"
	"errors"
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

var serverIP string

func input(text string) string {
	reader := bufio.NewReader(os.Stdin)
	fmt.Print(text)
	response, err := reader.ReadString('\n')
	if err != nil {
		panic(err)
	}

	return strings.TrimSpace(response)
}

func getTerminalDimensions() (int, int) {
	fd := int(os.Stdout.Fd())
	terminalWidth, terminalHeight, err := term.GetSize(fd)
	if err != nil {
		panic(err)
	}
	return terminalWidth, terminalHeight
}

func getPadding(text string, offset int) string {
	terminalWidth, _ := getTerminalDimensions()
	return strings.Repeat(" ", (terminalWidth/2)-(len(text)/2)-offset)
}

func centeredText(text string, offset int) string {
	return getPadding(text, offset) + text
}

func sendData(payload string) (string, error) {
	conn, err := net.Dial("tcp", fmt.Sprintf("%s:9090", serverIP))
	if err != nil {
		return "", errors.New("lost connection")
	}

	defer func() {
		if closeErr := conn.Close(); closeErr != nil {
			panic(closeErr)
		}
	}()

	writer := bufio.NewWriter(conn)
	_, err = writer.WriteString(payload)
	if err != nil {
		return "", errors.New("failed to write playload")
	}

	err = writer.Flush()
	if err != nil {
		return "", errors.New("failed to flush writer")
	}

	netData, err := bufio.NewReader(conn).ReadString('\n')
	if err != nil {
		return "", errors.New("failed to read data")
	}

	return netData, nil
}

func chooseServer(serverMessage string) {
	for {
		printHeader([]string{serverMessage}, 1)

		serverIP = input("Enter a server IP (or exit to exit): ")
		if slices.Contains([]string{"exit", "EXIT", "e", "E"}, serverIP) {
			break
		}

		if getHealthCheck() {
			break
		} else {
			serverMessage = fmt.Sprintf("Could not connect to server %s", serverIP)
		}
	}
}

func getHealthCheck() bool {
	netData, err := sendData("{\"action\": \"healthCheck\"}\n")
	if err != nil {
		return false
	}

	type HealthDetails struct {
		Service string
		Success bool
	}

	var healthDetails HealthDetails
	var healthData map[string]any
	err = json.Unmarshal([]byte(netData), &healthData)
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

func printLogo(extraLines int) {
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
	terminalWidth, terminalHeight := getTerminalDimensions()
	if terminalWidth > len(titleLargeLines[0]) && terminalHeight > len(titleLargeLines)+extraLines {
		printTitle = titleLargeLines
	} else if terminalWidth > len(titleMediumLines[0]) && terminalHeight > len(titleMediumLines)+extraLines {
		printTitle = titleMediumLines
	} else if terminalWidth > len(titleSmallLines[0]) && terminalHeight > len(titleSmallLines)+extraLines {
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

func printHeader(messages []string, extraLines int) {
	extraLines += 1 + (len(messages) * 2)
	printLogo(extraLines)

	terminalWidth, _ := getTerminalDimensions()
	fmt.Println(strings.Repeat("─", terminalWidth))

	for _, message := range messages {
		if message != "" {
			fmt.Printf("%s\n", centeredText(message, 0))
			fmt.Println(strings.Repeat("─", terminalWidth))
		}
	}
}

func printMainMenu(message string) {
	terminalWidth, _ := getTerminalDimensions()

	printLogo(7)
	fmt.Println(strings.Repeat("─", terminalWidth))
	fmt.Printf("%s\n", centeredText(fmt.Sprintf("Current Server: %s", serverIP), 0))
	fmt.Println("─┬" + strings.Repeat("─", terminalWidth-2))

	menuOptions := []string{"Create a new room", "Join a room", "Exit"}
	for i, menuOption := range menuOptions {
		fmt.Printf("%d│%s\n", i+1, centeredText(menuOption, 2))
	}
	fmt.Println("─┴" + strings.Repeat("─", terminalWidth-2))

	if message != "" {
		fmt.Printf("%s\n", centeredText(message, 0))
		fmt.Println(strings.Repeat("─", terminalWidth))
	}
}

func printRoomMenu(isPrivate bool, numRounds int, handSize int, message string) {
	terminalWidth, _ := getTerminalDimensions()
	extraLines := 10
	if message != "" {
		extraLines += 2
	}
	printLogo(extraLines)

	fmt.Println(strings.Repeat("─", terminalWidth))
	fmt.Printf("%s\n", centeredText("Room Options", 0))
	fmt.Println("─┬" + strings.Repeat("─", terminalWidth-2))

	var publicOption string
	if isPrivate {
		publicOption = "Private"
	} else {
		publicOption = "Public"
	}

	fmt.Printf("1│%s\n", centeredText(fmt.Sprintf("Set Public/Private: %s", publicOption), 2))
	fmt.Printf("2│%s\n", centeredText(fmt.Sprintf("Change Number of Rounds: %d", numRounds), 2))
	fmt.Printf("3│%s\n", centeredText(fmt.Sprintf("Change Initial Hand Size: %d", handSize), 2))
	fmt.Println("─┼" + strings.Repeat("─", terminalWidth-2))
	fmt.Printf("C│%s\n", centeredText("Create room", 2))
	fmt.Printf("E│%s\n", centeredText("Return to main menu", 2))
	fmt.Println("─┴" + strings.Repeat("─", terminalWidth-2))

	if message != "" {
		fmt.Printf("%s\n", centeredText(message, 0))
		fmt.Println(strings.Repeat("─", terminalWidth))
	}
}

func roomOptions(isPrivate *bool, numRounds *int, handSize *int) bool {
	choice := ""
	errorMessage := ""

	for {
		printRoomMenu(*isPrivate, *numRounds, *handSize, errorMessage)

		choice = input("Choose a menu option: ")
		if !slices.Contains([]string{
			"1", "2", "3", "C", "E",
			"create", "CREATE", "exit", "EXIT",
			"public", "private", "rounds", "hand",
		}, choice) {
			errorMessage = "Invalid menu option selected."
			continue
		} else {
			errorMessage = ""
		}

		switch choice {
		case "C", "create", "CREATE":
			return true
		case "E", "exit", "EXIT":
			return false
		case "1", "public", "private":
			var privateChoice string
			messages := []string{"Set Public/Private"}

			for {
				printHeader(messages, 1)

				privateChoice = input("Choose Public or Private (public/private): ")
				if privateChoice != "public" && privateChoice != "private" {
					if len(messages) < 2 {
						messages = append(messages, "Error, please choose \"public\" or \"private\".")
					}
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
		case "2", "rounds":
			var inputRounds string
			messages := []string{"Set Number of Rounds"}

			for {
				printHeader(messages, 1)

				inputRounds = input("Enter the number of rounds to play: ")
				intRounds, err := strconv.Atoi(inputRounds)
				if err != nil {
					if len(messages) < 2 {
						messages = append(messages, "Error, please enter an integer.")
					} else {
						messages[1] = "Error, please enter an integer."
					}
					continue
				}

				if intRounds >= 1 && intRounds <= 10 {
					*numRounds = intRounds
					break
				} else {
					if len(messages) < 2 {
						messages = append(messages, "Error, please choose a number between 1 and 10.")
					} else {
						messages[1] = "Error, please choose a number between 1 and 10."
					}
					continue
				}
			}
		case "3", "hand":
			var inputHand string
			messages := []string{"Set Initial Hand Size"}

			for {
				printHeader(messages, 1)

				inputHand = input("Enter the number of cards to start the game with: ")
				intHand, err := strconv.Atoi(inputHand)
				if err != nil {
					if len(messages) < 2 {
						messages = append(messages, "Error, please enter an integer.")
					} else {
						messages[1] = "Error, please enter an integer."
					}
					continue
				}

				if intHand >= 1 && intHand <= 15 {
					*handSize = intHand
					break
				} else {
					if len(messages) < 2 {
						messages = append(messages, "Error, please choose a number between 1 and 15.")
					} else {
						messages[1] = "Error, please choose a number between 1 and 15."
					}
					continue
				}
			}
		}
	}
}

func createRoom(player *Player, isPrivate bool, numRounds int, handSize int) error {
	netData, err := sendData(fmt.Sprintf("{\"action\": \"createRoom\", \"isPrivate\": %t, \"numRounds\": %d, \"handSize\": %d}\n", isPrivate, numRounds, handSize))
	if err != nil {
		return err
	}

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
	return nil
}

func joinRoom(player *Player, roomCode string) (bool, error) {
	netData, err := sendData(fmt.Sprintf("{\"action\": \"joinRoom\", \"roomCode\": \"%s\"}\n", roomCode))
	if err != nil {
		return false, err
	}

	type JoinDetails struct {
		Success bool
		Player  Player
	}

	var joinDetails JoinDetails
	var joinData map[string]any
	err = json.Unmarshal([]byte(netData), &joinData)
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
		return false, nil
	} else {
		*player = joinDetails.Player

		player.RoomCode = roomCode
		return true, nil
	}
}

func (player Player) leaveRoom() error {
	_, err := sendData(fmt.Sprintf("{\"playerID\": \"%s\", \"roomCode\": \"%s\", \"action\": \"leaveRoom\"}\n", player.PlayerID, player.RoomCode))
	return err
}

func getRooms() (string, error) {
	netData, err := sendData("{\"action\": \"getRooms\"}\n")
	if err != nil {
		return "", err
	}

	type RoomsDetails struct {
		Rooms string
	}

	var roomsDetails RoomsDetails
	var roomsData map[string]any
	err = json.Unmarshal([]byte(netData), &roomsData)
	if err != nil {
		fmt.Println(netData)
		panic(err)
	}

	// Loads the JSON data into the struct using mapstructure
	err = mapstructure.Decode(roomsData, &roomsDetails)
	if err != nil {
		panic(err)
	}
	return roomsDetails.Rooms, nil
}

func (player Player) startGame() (string, error) {
	netData, err := sendData(fmt.Sprintf("{\"playerID\": \"%s\", \"roomCode\": \"%s\", \"action\": \"startGame\"}\n", player.PlayerID, player.RoomCode))
	if err != nil {
		return "", err
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

	// Loads the JSON data into the struct using mapstructure
	err = mapstructure.Decode(handData, &initialHandDetails)
	if err != nil {
		panic(err)
	}
	return initialHandDetails.InitialHand, nil
}

func (player Player) waitForGameStart() (string, error) {
	netData, err := sendData(fmt.Sprintf("{\"playerID\": \"%s\", \"roomCode\": \"%s\", \"action\": \"waitStart\"}\n", player.PlayerID, player.RoomCode))
	if err != nil {
		return "", err
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

	// Loads the JSON data into the struct using mapstructure
	err = mapstructure.Decode(handData, &initialHandDetails)
	if err != nil {
		panic(err)
	}
	return initialHandDetails.InitialHand, nil
}

func (player Player) requestTurn() (int, string, error) {
	netData, err := sendData(fmt.Sprintf("{\"roomCode\": \"%s\", \"action\": \"requestTurn\"}\n", player.RoomCode))
	if err != nil {
		return 0, "", err
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

	// Loads the JSON data into the struct using mapstructure
	err = mapstructure.Decode(turnData, &requestTurnDetails)
	if err != nil {
		panic(err)
	}

	fmt.Println(requestTurnDetails.PlayerNumber)
	return requestTurnDetails.PlayerNumber, requestTurnDetails.TopCard, nil
}

func (player Player) waitTurn() (bool, int, int, int, map[string]int, error) {
	netData, err := sendData(fmt.Sprintf("{\"roomCode\": \"%s\", \"action\": \"waitTurn\"}\n", player.RoomCode))
	if err != nil {
		return false, 0, 0, 0, make(map[string]int), err
	}

	type GameWonDetails struct {
		WonRound           bool
		WinningRoundPlayer int
		WinningGamePlayer  int
		Round              int
		Stats              map[string]int
	}

	var gameWonDetails GameWonDetails
	var gameData map[string]any
	err = json.Unmarshal([]byte(netData), &gameData)
	if err != nil {
		fmt.Println(netData)
		panic(err)
	}

	// Loads the JSON data into the struct using mapstructure
	err = mapstructure.Decode(gameData, &gameWonDetails)
	if err != nil {
		panic(err)
	}

	return gameWonDetails.WonRound, gameWonDetails.WinningRoundPlayer, gameWonDetails.WinningGamePlayer, gameWonDetails.Round, gameWonDetails.Stats, nil
}

func (player Player) playCard(playedCard string) (bool, string, bool, int, int, map[string]int, error) {
	netData, err := sendData(fmt.Sprintf("{\"playerID\": \"%s\", \"roomCode\": \"%s\", \"action\": \"playCard\", \"card\": \"%s\"}\n", player.PlayerID, player.RoomCode, playedCard))
	if err != nil {
		return false, "", false, -1, 0, make(map[string]int), err
	}

	type RuleCheckResults struct {
		RulesPassed bool
		CurrentHand string
		WonRound    bool
		Winner      int
		Round       int
		Stats       map[string]int
	}

	var ruleCheckResults RuleCheckResults
	var ruleData map[string]any
	err = json.Unmarshal([]byte(netData), &ruleData)
	if err != nil {
		fmt.Println(netData)
		panic(err)
	}

	// Loads the JSON data into the struct using mapstructure
	err = mapstructure.Decode(ruleData, &ruleCheckResults)
	if err != nil {
		panic(err)
	}

	return ruleCheckResults.RulesPassed, ruleCheckResults.CurrentHand, ruleCheckResults.WonRound, ruleCheckResults.Winner, ruleCheckResults.Round, ruleCheckResults.Stats, nil
}

func (player Player) addRule(newRule string) (bool, error) {
	netData, err := sendData(fmt.Sprintf("{\"playerID\": \"%s\", \"roomCode\": \"%s\", \"action\": \"addRule\", \"newRule\": \"%s\"}\n", player.PlayerID, player.RoomCode, newRule))
	if err != nil {
		return false, err
	}

	type AddRuleResults struct {
		Success bool
	}

	var addRuleResults AddRuleResults
	var ruleData map[string]any
	err = json.Unmarshal([]byte(netData), &ruleData)
	if err != nil {
		fmt.Println(netData)
		panic(err)
	}

	// Loads the JSON data into the struct using mapstructure
	err = mapstructure.Decode(ruleData, &addRuleResults)
	if err != nil {
		panic(err)
	}

	return addRuleResults.Success, nil
}

func (player Player) getStats() (map[string]int, error) {
	netData, err := sendData(fmt.Sprintf("{\"roomCode\": \"%s\", \"action\": \"getStats\"}\n", player.RoomCode))
	if err != nil {
		return make(map[string]int), err
	}

	type Stats struct {
		Stats map[string]int
	}

	var stats Stats
	var statsData map[string]any
	err = json.Unmarshal([]byte(netData), &statsData)
	if err != nil {
		fmt.Println(netData)
		panic(err)
	}

	// Loads the JSON data into the struct using mapstructure
	err = mapstructure.Decode(statsData, &stats)
	if err != nil {
		panic(err)
	}

	return stats.Stats, nil
}

func printTurn(player Player, handList string, adjustedPlayerNumber int, turnTopCard string, message string) {
	extraLines := 8
	if message != "" {
		extraLines += 2
	}
	printHeader([]string{}, extraLines)

	terminalWidth, _ := getTerminalDimensions()
	if adjustedPlayerNumber-1 == player.PlayerNumber {
		fmt.Println(centeredText("YOUR TURN", 0))
	} else {
		fmt.Println(centeredText(fmt.Sprintf("Player %d's turn", adjustedPlayerNumber), 0))
	}
	fmt.Println(strings.Repeat("─", terminalWidth))
	fmt.Println(centeredText("Current Card:", 0))
	fmt.Println(centeredText(turnTopCard, 0))
	fmt.Println(centeredText("Your Hand:", 0))
	fmt.Println(centeredText(handList, 0))
	fmt.Println(strings.Repeat("─", terminalWidth))

	if message != "" {
		fmt.Println(centeredText(message, 0))
		fmt.Println(strings.Repeat("─", terminalWidth))
	}
}

func printStats(stats map[string]int) {
	terminalWidth, _ := getTerminalDimensions()
	for playerNumber, numWins := range stats {
		intPlayerNumber, err := strconv.Atoi(playerNumber)
		if err != nil {
			panic(err)
		}

		var winText string
		if numWins == 1 {
			winText = "win"
		} else {
			winText = "wins"
		}
		fmt.Println(centeredText(fmt.Sprintf("Player %d - %d %s", intPlayerNumber+1, numWins, winText), 0))
	}
	fmt.Println(strings.Repeat("─", terminalWidth))
}

func main() {
	var player Player

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	go func() {
		for sig := range c {
			if sig.String() == "interrupt" {
				_ = player.leaveRoom()
				os.Exit(0)
			}
		}
	}()

	var handList string
	var serverMessage string
	var lostConnection bool
	for {
		if serverIP == "" {
			chooseServer(serverMessage)
			serverMessage = ""
			lostConnection = false
		}

		if slices.Contains([]string{"exit", "EXIT", "e", "E"}, serverIP) {
			return
		}

		printMainMenu("")
		choice := input("Choose a menu option: ")
		for {
			if !slices.Contains([]string{
				"1", "2", "3",
				"create", "CREATE", "c", "C",
				"join", "JOIN", "j", "J",
				"exit", "EXIT", "e", "E",
			}, choice) {
				printMainMenu("Error, please choose a valid menu option.")
				choice = input("Choose a menu option: ")
			} else {
				break
			}
		}

		switch choice {
		case "1", "create", "CREATE", "c", "C":
			isPrivate := false
			numRounds := 5
			handSize := 7

			doCreate := roomOptions(&isPrivate, &numRounds, &handSize)

			if doCreate {
				err := createRoom(&player, isPrivate, numRounds, handSize)
				if err != nil {
					serverMessage = "Lost connection to server."
					serverIP = ""
					continue
				}
			} else {
				continue
			}
		case "2", "join", "JOIN", "j", "J":
			errorMessage := ""
			exitBreak := false
			re := regexp.MustCompile(`[A-Z]{4}`)

			for {
				roomsList, err := getRooms()
				if err != nil {
					lostConnection = true
					exitBreak = true
					break
				}

				messages := []string{fmt.Sprintf("Open Rooms: %s", roomsList)}
				if errorMessage != "" {
					messages = append(messages, errorMessage)
				}
				printHeader(messages, 1)

				roomCode := input("Enter a room code: ")
				match := re.MatchString(roomCode)
				if slices.Contains([]string{"exit", "EXIT", "e", "E"}, roomCode) {
					exitBreak = true
					break
				} else if !match {
					errorMessage = "Error, please choose a valid room code."
					continue
				}

				joinedSuccessfully, err := joinRoom(&player, roomCode)
				if err != nil {
					lostConnection = true
					exitBreak = true
					break
				}

				if joinedSuccessfully {
					break
				} else {
					errorMessage = "Error, please choose a valid room code."
				}
			}

			if lostConnection {
				serverMessage = "Lost connection to server."
				serverIP = ""
			}

			if exitBreak {
				continue
			}
		case "3", "exit", "EXIT", "e", "E":
			serverIP = ""
			continue
		}

		if player.CanStartGame {
			for {
				message := fmt.Sprintf("Room Code: %s", player.RoomCode)
				printHeader([]string{message}, 1)

				choice = input("Press Y to start the game: ") // Should include an option to back out. Can I also display number of users joined?
				if choice == "Y" {
					newHandList, err := player.startGame()
					if err != nil {
						lostConnection = true
					} else {
						handList = newHandList
					}
					break
				}
			}
			if lostConnection {
				serverMessage = "Lost connection to server."
				serverIP = ""
				continue
			}
		} else {
			printHeader([]string{"Waiting for game to start..."}, 0)
			newHandList, err := player.waitForGameStart()
			if err != nil {
				serverMessage = "Lost connection to server."
				serverIP = ""
				continue
			} else {
				handList = newHandList
			}
		}

		gameMessage := ""
		for {
			// Request turn
			turnPlayerNumber, turnTopCard, err := player.requestTurn()
			if err != nil {
				lostConnection = true
				break
			}

			// If my player number equals the returned turn player, take turn. Else, wait for turn and display current hand/top card
			printTurn(player, handList, turnPlayerNumber+1, turnTopCard, gameMessage)
			gameMessage = ""
			if player.PlayerNumber == turnPlayerNumber {
				var playedCard string
				for {
					playedCard = input("Choose a card to play (or 'draw' to draw): ")
					if !slices.Contains(strings.Split(handList, ", "), playedCard) && playedCard != "draw" && playedCard != "leave" {
						printTurn(player, handList, turnPlayerNumber+1, turnTopCard, "Error: The card you entered is not in hand.")
					} else {
						break
					}
				}

				if playedCard == "leave" {
					err := player.leaveRoom()
					if err != nil {
						lostConnection = true
					}
					break
				}

				playSucceeded, currentHand, wonRound, winner, round, stats, err := player.playCard(playedCard)
				if err != nil {
					lostConnection = true
					break
				}

				if winner != -1 {
					if winner == player.PlayerNumber {
						printHeader([]string{"You won the game!"}, len(stats)+2)
					} else {
						printHeader([]string{fmt.Sprintf("Game Over. Player %d wins!", winner+1)}, len(stats)+2)
					}
					printStats(stats)
					_ = input("Press Y to leave the game: ")
					break
				}

				if wonRound {
					stats, err := player.getStats()
					if err != nil {
						lostConnection = true
						break
					}
					printHeader([]string{fmt.Sprintf("You won Round %d!", round)}, len(stats)+2)
					printStats(stats)

					newRule := input("As your reward, describe a new rule to add to the game: ")
					if newRule == "leave" {
						err := player.leaveRoom()
						if err != nil {
							lostConnection = true
						}
						break
					}

					addedRule, err := player.addRule(newRule)
					if err != nil {
						lostConnection = true
						break
					}

					if addedRule {
						gameMessage = "Rule added successfully."
					} else {
						gameMessage = "Your rule could not be added."
					}

					newHandList, err := player.startGame()
					if err != nil {
						lostConnection = true
					} else {
						handList = newHandList
					}

					continue
				}

				if playSucceeded {
					handList = currentHand
					gameMessage = ""
				} else {
					handList = currentHand
					gameMessage = "You broke a rule and incurred a penalty."
				}

				// send card to play. can validate user actually has card on server side now cause it's handled over there
				// once card has actually been played and is valid, respond back with new handlist for next turn
			} else {
				terminalWidth, _ := getTerminalDimensions()
				fmt.Println(centeredText("Waiting for turn...", 0))
				fmt.Println(strings.Repeat("─", terminalWidth))

				wonRound, roundWinner, winner, round, stats, err := player.waitTurn()
				if err != nil {
					lostConnection = true
					break
				}

				if winner != -1 {
					if winner == player.PlayerNumber {
						printHeader([]string{"You won the game!"}, len(stats)+2)
					} else {
						printHeader([]string{fmt.Sprintf("Game Over. Player %d wins!", winner+1)}, len(stats)+2)
					}
					printStats(stats)
					_ = input("Press Y to leave the game: ")
					break
				}

				if wonRound {
					stats, err := player.getStats()
					if err != nil {
						lostConnection = true
						break
					}
					terminalWidth, _ := getTerminalDimensions()
					message := fmt.Sprintf("Player %d won Round %d.", roundWinner, round)
					printHeader([]string{message}, len(stats)+2)
					printStats(stats)

					fmt.Println(centeredText(fmt.Sprintf("Waiting for player %d to add a new rule...", roundWinner), 0))
					fmt.Println(strings.Repeat("─", terminalWidth))

					newHandList, err := player.waitForGameStart()
					if err != nil {
						lostConnection = true
						break
					} else {
						handList = newHandList
					}

					continue
				}
			}
		}
		if lostConnection {
			serverMessage = "Lost connection to server."
			serverIP = ""
		}
	}
}
