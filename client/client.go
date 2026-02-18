package main

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"os"
	"os/signal"
	"slices"
	"sort"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/go-viper/mapstructure/v2"
	"golang.org/x/term"
)

// Player struct that holds player information.
type Player struct {
	PlayerID     string
	PlayerNumber int
	Hand         string
	CanStartGame bool
	RoomCode     string
}

// Server struct that holds server information.
type Server struct {
	IP             string
	Message        string
	LostConnection bool
}

// Server is a global variable to avoid having to pass it in every function.
var server Server

// Helper function to request and return user input
func input(text string) string {
	reader := bufio.NewReader(os.Stdin)
	fmt.Print(text)
	response, err := reader.ReadString('\n')
	if err != nil {
		panic(err)
	}

	return strings.TrimSpace(response)
}

// Helper function that gets the current terminal dimensions. Helps for printing UI.
func getTerminalDimensions() (int, int) {
	fd := int(os.Stdout.Fd())
	terminalWidth, terminalHeight, err := term.GetSize(fd)
	if err != nil {
		panic(err)
	}
	return terminalWidth, terminalHeight
}

// Helper function that determines how much padding text needs in order for it to be centered.
// Includes an offset parameter for bumping text left if there is left aligned text.
func getPadding(text string, offset int) string {
	terminalWidth, _ := getTerminalDimensions()
	return strings.Repeat(" ", (terminalWidth/2)-(len(text)/2)-offset)
}

// Helper function that utilizes getPadding to return a text prepended with enough padding to center it on the screen.
func centeredText(text string, offset int) string {
	return getPadding(text, offset) + text
}

// Generic function for sending data to the server and receiving a response.
func sendData(payload string) (string, error) {
	conn, err := net.Dial("tcp", fmt.Sprintf("%s:9090", server.IP))
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

// First screen for selecting the server used for the game.
func chooseServer() {
	for {
		printHeader([]string{server.Message}, 1)

		server.IP = input("Enter a server IP (or exit to exit): ")
		if slices.Contains([]string{"exit", "EXIT", "e", "E"}, server.IP) {
			// If exit is chosen, return. In main, server.IP is checked for exit to quit the program.
			return
		}

		// After choosing an IP, runs a health check to see if the chosen server actually hosts Mao.
		if getHealthCheck() {
			server.Message = ""
			server.LostConnection = false
			return
		} else {
			server.Message = fmt.Sprintf("Could not connect to server %s", server.IP)
		}
	}
}

// Requests a health check from the server to determine if it is actually hosting Mao.
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

// Prints and centers the correctly sized Mao logo depending on the current terminal size.
// extraLines is passed to help assess the number of vertical lines on the screen, which will between
// the height of the logo along with however many other lines the function utilizing this adds.
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

// Prints the header, which is essentially the logo along with any number of messages.
// Messages are passed as a list and are separated by horizontal lines.
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

// Prints the main menu and its options. Doesn't use printHeader as it needs to print horizontal lines
// joined by some vertical elements for the left menu options.
func printMainMenu(message string) {
	printLogo(7)

	terminalWidth, _ := getTerminalDimensions()
	fmt.Println(strings.Repeat("─", terminalWidth))
	fmt.Printf("%s\n", centeredText(fmt.Sprintf("Current Server: %s", server.IP), 0))
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

// The interactive portion of the main menu. It has the following options:
// Create - Allows a user to create a new room and determine room settings.
// Join - Allows a user to join an already created room.
// Exit - Allows a user to go back to the server selection screen.
func mainMenu(player *Player) (bool, error) {
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

		doCreate := createOptions(&isPrivate, &numRounds, &handSize)

		// If a user has chosen to create the room, create it with the specified values and continue with the game.
		// Otherwise, return false (which returns to the main menu in the main function loop).
		if doCreate {
			err := createRoom(player, isPrivate, numRounds, handSize)
			if err != nil {
				return false, err
			}

			return true, nil
		} else {
			return false, nil
		}
	case "2", "join", "JOIN", "j", "J":
		doJoin, err := joinOptions(player)
		if err != nil {
			return false, err
		}

		// If a user has chosen a valid room to join, continue with the game.
		// Otherwise, return false (which returns to the main menu in the main function loop).
		if doJoin {
			return true, nil
		} else {
			return false, nil
		}
	case "3", "exit", "EXIT", "e", "E":
		// Exit specifically sets server.IP blank along with returning false,
		// that way the server selection screen appears at the top of the loop.
		server.IP = ""
		return false, nil
	}
	return false, nil
}

// Prints menu options for the room creation screen.
func printRoomMenu(isPrivate bool, numRounds int, handSize int, message string) {
	terminalWidth, _ := getTerminalDimensions()
	extraLines := 10
	if message != "" {
		extraLines += 2
	}

	var publicOption string
	if isPrivate {
		publicOption = "Private"
	} else {
		publicOption = "Public"
	}

	printLogo(extraLines)
	fmt.Println(strings.Repeat("─", terminalWidth))
	fmt.Printf("%s\n", centeredText("Room Options", 0))
	fmt.Println("─┬" + strings.Repeat("─", terminalWidth-2))

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

// The interactive portion of the room creation menu menu. It has the following options:
// Create - Returns true which tells the main menu to create the room and continue.
// Exit - Returns false which sends the player back to the main menu.
// Public/Private - Allows a user to specify if the created room appears in the join room menu.
// Rounds - Allows a user to specify how many rounds to play in the game.
// handSize - Allows a user to specify how many cards each player starts a round with.
func createOptions(isPrivate *bool, numRounds *int, handSize *int) bool {
	choice := ""
	errorMessage := ""

	for {
		printRoomMenu(*isPrivate, *numRounds, *handSize, errorMessage)

		choice = input("Choose a menu option: ")
		if !slices.Contains([]string{
			"1", "2", "3",
			"c", "C", "create", "CREATE",
			"e", "E", "exit", "EXIT",
			"public", "private",
			"round", "rounds",
			"hand", "handsize",
		}, choice) {
			errorMessage = "Invalid menu option selected."
			continue
		} else {
			errorMessage = ""
		}

		switch choice {
		case "c", "C", "create", "CREATE":
			return true
		case "e", "E", "exit", "EXIT":
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
		case "2", "round", "rounds":
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
		case "3", "hand", "handsize":
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

// The interactive portion of the join room menu. It has the following options:
// RoomCode - A user can enter any text to see if an available room code exists. If it does, returns true to continue the game.
// Exit - Returns false which sends the player back to the main menu.
func joinOptions(player *Player) (bool, error) {
	errorMessage := ""

	for {
		roomsList, err := getRooms()
		if err != nil {
			return false, err
		}

		messages := []string{fmt.Sprintf("Open Rooms: %s", roomsList)}
		if errorMessage != "" {
			messages = append(messages, errorMessage)
		}
		printHeader(messages, 1)

		roomCode := input("Enter a room code: ")
		if slices.Contains([]string{"exit", "EXIT", "e", "E"}, roomCode) {
			return false, nil
		}

		joinedSuccessfully, err := joinRoom(player, roomCode)
		if err != nil {
			return false, err
		}

		if joinedSuccessfully {
			return true, nil
		} else {
			errorMessage = "Error, please choose a valid room code."
		}
	}
}

// Acts as the beginning of the game screen and supplies the initial hand to the player.
// If the player created the room, they can start the game by entering Y or cancel it with N.
// If the player joined the room, they simply wait until they receive the go ahead from the server.
func beginGame(player *Player) (bool, error) {
	var choice string
	errorMessage := ""
	if player.CanStartGame {
		for {
			messages := []string{fmt.Sprintf("Room Code: %s", player.RoomCode)}
			if errorMessage != "" {
				messages = append(messages, errorMessage)
			}
			printHeader(messages, 1)

			choice = input("Press Y to start the game: ")
			switch choice {
			case "y", "Y", "yes", "YES", "start", "START", "c", "C", "create", "CREATE":
				newHandList, err := player.startGame()
				if err != nil {
					return false, err
				} else {
					player.Hand = newHandList
					return true, nil
				}
			case "n", "N", "no", "NO", "cancel", "CANCEL", "e", "E", "exit", "EXIT":
				_, err := player.cancelGame()
				if err != nil {
					return false, err
				}
				return false, nil
			default:
				errorMessage = "Error, please choose y/n to start or cancel the game."
			}
		}
	} else {
		printHeader([]string{"Waiting for game to start..."}, 0)

		canceledGame, newHandList, err := player.waitForGameStart()
		if err != nil {
			return false, err
		} else if canceledGame {
			return false, nil
		} else {
			player.Hand = newHandList
			return true, nil
		}
	}
}

// Handles when the player plays a card. If the played card breaks the rules, continues the loop until a successful play.
// On a successful play, returns if a winner has been determined along with other game elements and stats.
func playTurn(player *Player, turnPlayerNumber int, turnTopCard string, gameMessage string) (bool, bool, int, int, map[string]int, error) {
	var playedCard string
	for {
		printTurn(*player, turnPlayerNumber+1, turnTopCard, gameMessage)
		for {
			playedCard = input("Choose a card to play (or 'draw' to draw): ")
			if !slices.Contains(strings.Split(player.Hand, ", "), playedCard) && playedCard != "draw" && playedCard != "leave" {
				gameMessage = "Error: The card you entered is not in hand."
				printTurn(*player, turnPlayerNumber+1, turnTopCard, gameMessage)
			} else {
				break
			}
		}

		if playedCard == "leave" {
			err := player.leaveRoom()
			if err != nil {
				return true, false, -1, 0, make(map[string]int), err
			}
			return true, false, -1, 0, make(map[string]int), nil
		}

		playSucceeded, currentHand, wonRound, winner, round, stats, err := player.playCard(playedCard)
		if err != nil {
			return true, false, -1, 0, make(map[string]int), err
		}

		player.Hand = currentHand
		if playSucceeded {
			return false, wonRound, winner, round, stats, nil
		} else {
			gameMessage = "You broke a rule and incurred a penalty."
		}
	}
}

// Prints the end game screen whenever a player has won.
func endScreen(player Player, winner int, stats map[string]int) {
	if winner == player.PlayerNumber {
		printHeader([]string{"You won the game!"}, len(stats)+2)
	} else {
		printHeader([]string{fmt.Sprintf("Game Over. Player %d wins!", winner+1)}, len(stats)+2)
	}
	printStats(stats)
	_ = input("Enter anything to leave the game: ")
}

// Prints the round winning screen along with allowing the player to add a new rule to the current Mao game.
func winnerScreen(player *Player, round int) (bool, string, error) {
	var gameMessage string

	stats, err := player.getStats()
	if err != nil {
		return true, "", err
	}

	printHeader([]string{fmt.Sprintf("You won Round %d!", round)}, len(stats)+2)
	printStats(stats)

	newRule := input("As your reward, describe a new rule to add to the game: ")
	if newRule == "leave" {
		err := player.leaveRoom()
		if err != nil {
			return true, "", err
		}
		return true, "", nil
	}

	addedRule, err := player.addRule(newRule)
	if err != nil {
		return true, "", err
	}

	if addedRule {
		gameMessage = "Rule added successfully."
	} else {
		gameMessage = "Your rule could not be added."
	}

	newHandList, err := player.startGame()
	if err != nil {
		return true, "", err
	} else {
		player.Hand = newHandList
		return false, gameMessage, nil
	}
}

// Prints the waiting screen for when a player is waiting for their turn.
func waitingScreen(player Player, turnPlayerNumber int, turnTopCard string, gameMessage string) {
	printTurn(player, turnPlayerNumber+1, turnTopCard, gameMessage)
	terminalWidth, _ := getTerminalDimensions()
	fmt.Println(centeredText("Waiting for turn...", 0))
	fmt.Println(strings.Repeat("─", terminalWidth))
}

// Prints the losing screen if the player lost the round and
// waits for the winning player to add a new rule.
func loserScreen(player *Player, roundWinner int, round int) error {
	stats, err := player.getStats()
	if err != nil {
		return err
	}
	terminalWidth, _ := getTerminalDimensions()
	message := fmt.Sprintf("Player %d won Round %d.", roundWinner, round)
	printHeader([]string{message}, len(stats)+2)
	printStats(stats)

	fmt.Println(centeredText(fmt.Sprintf("Waiting for player %d to add a new rule...", roundWinner), 0))
	fmt.Println(strings.Repeat("─", terminalWidth))

	_, newHandList, err := player.waitForGameStart()
	if err != nil {
		return err
	} else {
		player.Hand = newHandList
		return nil
	}
}

// Sends the room creation data to the server to create a room.
// Also populates the player struct with the returned data, such as the playerID.
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

	err = mapstructure.Decode(playerData, &player)
	if err != nil {
		panic(err)
	}
	return nil
}

// Sends a request to join a room to the server.
// If the room exists, supplies the user struct with player details.
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

// Sends a request to the server to leave the room.
func (player Player) leaveRoom() error {
	_, err := sendData(fmt.Sprintf("{\"playerID\": \"%s\", \"roomCode\": \"%s\", \"action\": \"leaveRoom\"}\n", player.PlayerID, player.RoomCode))
	return err
}

// Sends a request to the server to get a list of currently open and public rooms.
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

	err = mapstructure.Decode(roomsData, &roomsDetails)
	if err != nil {
		panic(err)
	}
	return roomsDetails.Rooms, nil
}

// Sends a request to the server to start the game. Only succeeds if the player created the room.
func (player Player) startGame() (string, error) {
	netData, err := sendData(fmt.Sprintf("{\"playerID\": \"%s\", \"roomCode\": \"%s\", \"action\": \"startGame\"}\n", player.PlayerID, player.RoomCode))
	if err != nil {
		return "", err
	}

	type InitialHandDetails struct {
		InitialHand string
		Success     bool
	}

	var initialHandDetails InitialHandDetails
	var handData map[string]any
	err = json.Unmarshal([]byte(netData), &handData)
	if err != nil {
		fmt.Println(netData)
		panic(err)
	}

	err = mapstructure.Decode(handData, &initialHandDetails)
	if err != nil {
		panic(err)
	}
	return initialHandDetails.InitialHand, nil
}

// Sends a request to the server to cancel the game. Only succeeds if the player created the room.
func (player Player) cancelGame() (bool, error) {
	netData, err := sendData(fmt.Sprintf("{\"playerID\": \"%s\", \"roomCode\": \"%s\", \"action\": \"cancelGame\"}\n", player.PlayerID, player.RoomCode))
	if err != nil {
		return false, err
	}

	type CancelDetails struct {
		Success bool
	}

	var cancelDetails CancelDetails
	var cancelData map[string]any
	err = json.Unmarshal([]byte(netData), &cancelData)
	if err != nil {
		fmt.Println(netData)
		panic(err)
	}

	err = mapstructure.Decode(cancelData, &cancelDetails)
	if err != nil {
		panic(err)
	}
	return cancelDetails.Success, nil
}

// Sends a request to the server to wait for the game to start. Once the game has begun on the server side,
// returns the players starting hand.
func (player Player) waitForGameStart() (bool, string, error) {
	netData, err := sendData(fmt.Sprintf("{\"playerID\": \"%s\", \"roomCode\": \"%s\", \"action\": \"waitStart\"}\n", player.PlayerID, player.RoomCode))
	if err != nil {
		return false, "", err
	}

	type InitialHandDetails struct {
		InitialHand string
		Success     bool
	}

	var initialHandDetails InitialHandDetails
	var handData map[string]any
	err = json.Unmarshal([]byte(netData), &handData)
	if err != nil {
		fmt.Println(netData)
		panic(err)
	}

	err = mapstructure.Decode(handData, &initialHandDetails)
	if err != nil {
		panic(err)
	}
	return !initialHandDetails.Success, initialHandDetails.InitialHand, nil
}

// Sends a request to the server for a turn. The server responds with whose turn it currently is and,
// depending on the response, allows the player to play their turn in the main gameplay loop.
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

	err = mapstructure.Decode(turnData, &requestTurnDetails)
	if err != nil {
		panic(err)
	}

	fmt.Println(requestTurnDetails.PlayerNumber)
	return requestTurnDetails.PlayerNumber, requestTurnDetails.TopCard, nil
}

// Sends a wait turn request to the server. This is when it is another player's turn and the client
// wants a response when it is the next player's turn.
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

	err = mapstructure.Decode(gameData, &gameWonDetails)
	if err != nil {
		panic(err)
	}

	return gameWonDetails.WonRound, gameWonDetails.WinningRoundPlayer, gameWonDetails.WinningGamePlayer, gameWonDetails.Round, gameWonDetails.Stats, nil
}

// Sends a request to the server to play a card. The server responds with helpful details like:
// RulesPassed - A bool representing if the played card was valid.
// CurrentHand - The current hand of the player.
// WonRound - If the player has won the round.
// Winner - If a player has won the game.
// Round - What round it is.
// Stats - A map of players and their win counts.
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

	err = mapstructure.Decode(ruleData, &ruleCheckResults)
	if err != nil {
		panic(err)
	}

	return ruleCheckResults.RulesPassed, ruleCheckResults.CurrentHand, ruleCheckResults.WonRound, ruleCheckResults.Winner, ruleCheckResults.Round, ruleCheckResults.Stats, nil
}

// Sends a request to the server to add a new rule. Depending on if the server is able to
// accomplish the request, it sends a bool designating success.
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

	err = mapstructure.Decode(ruleData, &addRuleResults)
	if err != nil {
		panic(err)
	}

	return addRuleResults.Success, nil
}

// Sends a request to the server to see each player and how many wins they have.
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

	err = mapstructure.Decode(statsData, &stats)
	if err != nil {
		panic(err)
	}

	return stats.Stats, nil
}

// Prints game details during a player's turn.
func printTurn(player Player, adjustedPlayerNumber int, turnTopCard string, message string) {
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
	fmt.Println(centeredText(player.Hand, 0))
	fmt.Println(strings.Repeat("─", terminalWidth))

	if message != "" {
		fmt.Println(centeredText(message, 0))
		fmt.Println(strings.Repeat("─", terminalWidth))
	}
}

// Prints player win count stats.
func printStats(stats map[string]int) {
	terminalWidth, _ := getTerminalDimensions()
	var sortedStats []string
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

		sortedStats = append(sortedStats, centeredText(fmt.Sprintf("Player %d - %d %s", intPlayerNumber+1, numWins, winText), 0))
	}

	sort.Strings(sortedStats)
	for _, stat := range sortedStats {
		fmt.Println(stat)
	}
	fmt.Println(strings.Repeat("─", terminalWidth))
}

// The main driver of the program and gameplay loop.
func main() {
	var player Player

	// A goroutine for user interrupts. If one occurs, has the player leave the room before exiting.
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

	// Begin application loop.
	for {
		// If a server IP has not been selected, prompot the user for one.
		if server.IP == "" {
			chooseServer()
		}

		// If the user chose to exit, quit the program.
		if slices.Contains([]string{"exit", "EXIT", "e", "E"}, server.IP) {
			return
		}

		// Presents the user with the main menu.
		printMainMenu("")
		continueGame, err := mainMenu(&player)
		if err != nil {
			server.Message = "Lost connection to server."
			server.IP = ""
		}

		// If the user chose not to create or join a room, returns back to the server selection screen.
		if !continueGame {
			continue
		}

		// Presents the game start waiting screen.
		// If the user created the room, prompts user to start the game. Otherwise, the player waits.
		gameStarted, err := beginGame(&player)
		if err != nil {
			server.Message = "Lost connection to server."
			server.IP = ""
		}

		// If the who created the room canceled the game, returns back to the main menu.
		if !gameStarted {
			continue
		}

		// Begins the actual gameplay loop during a game of Mao.
		gameMessage := ""
		for {
			// The client requests whose turn it is.
			turnPlayerNumber, turnTopCard, err := player.requestTurn()
			if err != nil {
				server.LostConnection = true
				break
			}

			// If it is the player's turn, play a card.
			if player.PlayerNumber == turnPlayerNumber {
				leaveGame, wonRound, winner, round, stats, err := playTurn(&player, turnPlayerNumber, turnTopCard, gameMessage)
				gameMessage = ""
				if err != nil {
					server.LostConnection = true
					break
				} else if leaveGame {
					break
				}

				// If a game winner has been announced by the server, present the end screen.
				if winner != -1 {
					endScreen(player, winner, stats)
					break
				}

				// If a round winner has been announced by the server, show the round winner screen and rule prompt.
				// It is assumed this player has won the round given they played the last card.
				if wonRound {
					leaveGame, gameMessage, err = winnerScreen(&player, round)
					if err != nil {
						server.LostConnection = true
						break
					} else if leaveGame {
						break
					}
				}

			} else {
				// Print the turn waiting screen.
				waitingScreen(player, turnPlayerNumber, turnTopCard, gameMessage)

				// If it is not the player's turn, wait for the next turn.
				wonRound, roundWinner, winner, round, stats, err := player.waitTurn()
				if err != nil {
					server.LostConnection = true
					break
				}

				// If a game winner has been announced by the server, present the end screen.
				if winner != -1 {
					endScreen(player, winner, stats)
					break
				}

				// If a round winner has been announced by the server, show the round loser screen and waiting message.
				// It is assumed this player has lost the round given they did not play the last card.
				if wonRound {
					err := loserScreen(&player, roundWinner, round)
					if err != nil {
						server.LostConnection = true
						break
					}
				}
			}
		}

		// If at any point the server has lost connection, return to the server selection screen.
		if server.LostConnection {
			server.Message = "Lost connection to server."
			server.IP = ""
		}
	}
}
