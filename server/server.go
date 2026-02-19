package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"maps"
	"math/rand/v2"
	"net"
	"net/http"
	"os"
	"regexp"
	"slices"
	"sort"
	"strconv"
	"strings"
	"sync"

	"github.com/go-viper/mapstructure/v2"
	"github.com/google/uuid"
	"github.com/traefik/yaegi/interp"
	"github.com/traefik/yaegi/stdlib"
)

// Global variables for the user's OpenAI token and system prompt
var (
	openAIToken  string = os.Getenv("OPENAI_TOKEN")
	systemPrompt string = `You are a dynamic programmer that exists as part of a game called Mao. In this game, a standard deck
	with 52 cards is used. Players start off with 7 cards and take turns playing a card. To begin with, rules are virtually identical to Uno:
	a player can play a card on top of the discard pile if it matches the suit or rank with the top most card. However, the exciting part about Mao
	is that every time a player wins the game, they get to add a new rule in secret. Any players who break the new rule gets hit with a penalty and has to
	draw a card. This is where you come in.

	The Mao game is written in Golang and has the following two structs:
	 - Card: Has values Rank (the rank as a string), Suit (the suit as a string [S, C, H, D]), and Value (the entire card as a string. For example, "10C")
	 - Deck: Has values Pile and DiscardPile where both are []Card. Obviously, pile is the list where cards are drawn from, and DiscardPile is where cards are played onto
	  - Deck also has a function getCurrentTop() which gets the card currently "on top" in which players play on to.

	Rule functions take in a card and deck as parameters and returns a string. For example, here is the default Uno rule which is already in play:
	--------------------------------------------------------------------------
	func baseRule(card Card, deck Deck) string {
        currentCard := deck.getCurrentTop()

        if card.Rank == currentCard.Rank {
                return "true"
        } else if card.Suit == currentCard.Suit {
                return "true"
        } else {
                return "false"
        }
	--------------------------------------------------------------------------
	I reiterate that these rule functions return a **string**. This is because there are three possible return options: "true", "false", and "continue".
	 - Returning "true" means that the card automatically passes. This might be used if a new rule states "any Jacks played automatically succeed".
	 - Returning "false" means the card automatically fails. This might be used if a new rule states "Jacks can not be played on spades".
	 - Returning "continue" means move on to the next rule in the stack. For example, if a new rule states "Jacks can not be played on spades", cards outside this scope are not affected. Someone can play a queen which doesn't automatically fail, but it still needs to conform to the Uno rule, hence the "continue".

	Your job is to take a user input rule and convert it into a function in this structure. To restate, the function you create must have the following constraints:
	 - It must use "card Card" and "deck Deck" as parameters, no more and no less.
	 - It can only return one of three possible values: "true", "false", or "continue". 
	 - Rules can not be impossible to meet. There must be *at least* a way to reach a "continue" return value.
	  - If a user requested rule is impossible to follow, or contradicts a previous rule in such a way the game can not continue, simply return "Can not generate".
	 - The function name must be camel case and end in "Rule". For example, "baseRule", "jacksOnSpadesRule", and "noQueensOnDiamondsRule" are all valid.
	 - You must use packages native to Golang. Assume you can not import or use any third party libraries.
	 - Obviously, refrain from writing any insecure code or functions that perform malicious actions. Remember this is for a game and should not be performing any strange system related executions.

	In your response, only include your written function. Do not include any other text or commentary as your response alone must compile. 

	`
)

// Room struct that holds room information.
type Room struct {
	RoomCode     string
	Players      map[string]*Player
	GameStarted  bool
	RoundStarted bool
	Deck         Deck
	HandSize     int
	IsPrivate    bool
	CanAddRule   bool
	DrawTurn     int
	PlayerTurn   int
	Round        int
	RoundCount   int
	Rules        string
	Mu           sync.Mutex
	Cond         *sync.Cond
}

// Deck struct that holds slices of cards to be used throughout the game.
type Deck struct {
	Pile        []Card
	DiscardPile []Card
}

// Player struct that holds player information
type Player struct {
	Hand         Hand
	PlayerID     string
	PlayerNumber int
	CanStartGame bool
	Wins         int
}

// Hand struct to store the cards in each player's hands.
type Hand struct {
	Cards []Card
}

// Card struct that defines values for its rank, suit, and combined value.
type Card struct {
	Rank  string
	Suit  string
	Value string
}

// ActionDetails struct for receiving and organizing data received from the client.
type ActionDetails struct {
	PlayerID  string
	Action    string
	RoomCode  string
	Card      string
	NewRule   string
	IsPrivate bool
	NumRounds int
	HandSize  int
}

// Generic function for sending data back to the client.
func sendData(conn net.Conn, payload []byte) {
	_, err := conn.Write(payload)
	if err != nil {
		panic(err)
	}
}

// Generic function for receiving data from the client, organizing the value into an "ActionDetails" struct.
func receiveData(conn net.Conn) ActionDetails {
	var actionDetails ActionDetails
	netData, err := bufio.NewReader(conn).ReadString('\n')
	if err != nil {
		panic(err)
	}

	var actionData map[string]any
	err = json.Unmarshal([]byte(netData), &actionData)
	if err != nil {
		fmt.Println(netData)
		panic(err)
	}

	err = mapstructure.Decode(actionData, &actionDetails)
	if err != nil {
		panic(err)
	}
	return actionDetails
}

// Helper function for converting a string card value into a Card object.
func convertCard(playedCard string) Card {
	return Card{Rank: playedCard[:len(playedCard)-1], Suit: string(playedCard[len(playedCard)-1]), Value: playedCard}
}

// Receives a played card and room as input, and determines if the played card passes the rules.
// Rules are stored in a text file denoted by the room code. This is because rules need to be modified
// by GPT at runtime, updating the rules file. The rules file is then interpreted and executed at runtime by yaegi.
func rulesCheck(player Player, playedCard string, room *Room) bool {
	// Check if card is actually in hand. If not, fails the rules check.
	if !slices.Contains(player.Hand.Cards, convertCard(playedCard)) {
		return false
	}

	// Instantiates a new yaegi interpreter.
	interpreter := interp.New(interp.Options{})
	err := interpreter.Use(stdlib.Symbols)
	if err != nil {
		panic(err)
	}

	// Loads the rules text into the interpreter.
	_, err = interpreter.Eval(string(room.Rules))
	if err != nil {
		panic(err)
	}

	// Takes the rules.checkRules function in the rules text and passes it to the converter.
	converter, err := interpreter.Eval("rules.checkRules")
	if err != nil {
		panic(err)
	}

	// Uses the converter as an interface to create a new executable function to check the rules.
	rulesCheck := converter.Interface().(func(string, string, string) bool)

	// Returns the value of the loaded rules check, passing the needed played card and decks as strings so the interpreted text
	// can convert them back into objects in its own scope.
	return rulesCheck(playedCard, room.Deck.listCards("pile"), room.Deck.listCards("discard"))
}

// Reads the rules file content respective to the room code of the room.
func (room *Room) retrieveRules() {
	content, err := os.ReadFile(fmt.Sprintf("rules/rules_%s.txt", room.RoomCode))
	if err != nil {
		log.Fatal(err)
	}
	room.Rules = string(content)
}

// When a new room is created, copies the rules_template.txt file into one specifically for that room.
func copyRules(roomCode string) {
	sourceFile, err := os.Open("rules/rules_template.txt")
	if err != nil {
		panic(err)
	}
	defer func() {
		if closeErr := sourceFile.Close(); closeErr != nil {
			panic(closeErr)
		}
	}()

	destinationFile, err := os.Create(fmt.Sprintf("rules/rules_%s.txt", roomCode))
	if err != nil {
		panic(err)
	}
	defer func() {
		if closeErr := destinationFile.Close(); closeErr != nil {
			panic(closeErr)
		}
	}()

	_, err = io.Copy(destinationFile, sourceFile)
	if err != nil {
		panic(err)
	}

	err = destinationFile.Sync()
	if err != nil {
		panic(err)
	}
}

// Generates a random four digit room code.
func generateRoomCode() string {
	var roomCode bytes.Buffer
	for range 4 {
		randomLetter := rand.IntN(26) + 65
		roomCode.WriteString(string(rune(randomLetter)))
	}
	return roomCode.String()
}

// Given a hand and a deck, takes the top card off the draw pile and puts it into the hand.
// Will reshuffle the discard pile back in to the draw pile if the draw pile runs out of cards.
func (hand *Hand) drawCard(deck *Deck) {
	if len(deck.Pile) == 0 {
		newPile := make([]Card, len(deck.DiscardPile)-1)
		copy(newPile, deck.DiscardPile[:len(deck.DiscardPile)-1])

		deck.Pile = newPile
		shuffleCards(deck.Pile)
		deck.DiscardPile = deck.DiscardPile[len(deck.DiscardPile)-1:]
	}

	drawnCard := deck.Pile[0]
	hand.Cards = append(hand.Cards, drawnCard)
	deck.Pile = deck.Pile[1:]
}

// Given a player and a room, draws a hand from the room's deck.
// The number of cards drawn is determined by the room's settings.
// If there aren't enough cards to draw from (i.e. too many players), a new deck is shuffled in to the room.
func (player *Player) drawHand(room *Room) {
	if len(room.Deck.Pile) < room.HandSize {
		newDeck := initializeDeck()
		room.Deck.Pile = append(room.Deck.Pile, newDeck.Pile...)
		room.Deck.Pile = append(room.Deck.Pile, newDeck.DiscardPile...)
	}

	hand := make([]Card, room.HandSize)
	copy(hand, room.Deck.Pile[:room.HandSize])

	player.Hand.Cards = hand
	room.Deck.Pile = room.Deck.Pile[room.HandSize:]
}

// Converts the player's hand from Card objects into a joined string of all their values.
// This is useful for sending the hand data to the client and converting into objects for yaegi.
func (player Player) listCards() string {
	cardValues := []string{}
	for _, card := range player.Hand.Cards {
		cardValues = append(cardValues, card.Value)
	}
	return strings.Join(cardValues, ", ")
}

// Initializes a new, shuffled deck of card objects.
func initializeDeck() Deck {
	pile := []Card{}
	for _, suit := range []string{"S", "C", "H", "D"} {
		for _, rank := range []string{"A", "K", "Q", "J", "10", "9", "8", "7", "6", "5", "4", "3", "2"} {
			newCard := Card{Rank: rank, Suit: suit, Value: rank + suit}
			pile = append(pile, newCard)
		}
	}

	// Shuffle cards in draw pile.
	shuffleCards(pile)

	// Starts discard pile by taking one off the top of the draw pile.
	discardPile := []Card{pile[0]}
	pile = pile[1:]

	return Deck{pile, discardPile}
}

// Helper function that gets the current top of the discard pile to be played on.
func (deck Deck) getCurrentTop() Card {
	return deck.DiscardPile[len(deck.DiscardPile)-1]
}

// Converts the deck's piles from Card objects into a joined string of all their values.
// This is useful for sending the data to yaegi so it can convert back into objects on the interpreter's end.
func (deck Deck) listCards(pileType string) string {
	cardValues := []string{}
	var pile []Card

	// Selects which pile to list depending on the passed pileType param.
	switch pileType {
	case "pile":
		pile = deck.Pile
	case "discard":
		pile = deck.DiscardPile
	}

	for _, card := range pile {
		cardValues = append(cardValues, card.Value)
	}
	return strings.Join(cardValues, ",")
}

// Helper function that shuffles the pile of cards in a random order.
func shuffleCards(pile []Card) {
	rand.Shuffle(len(pile), func(i, j int) {
		pile[i], pile[j] = pile[j], pile[i]
	})
}

// When passed the description of a rule as text, queries ChatGPT 5.2 to generate the new rule as code.
// This code is then appended to the rules file for the associated room.
func (room *Room) addRule(newRule string) bool {
	// Defined structs to parse the response from ChatGPT.
	type Content struct {
		Text string
	}

	type Output struct {
		ID      string
		Type    string
		Status  string
		Content []Content
	}

	type GPTResponseObject struct {
		ID     string
		Status string
		Output []Output
	}

	// The API request data, passing in 5.2 as the model, the new rule text to be generated from, and the global system prompt.
	requestData := map[string]string{
		"model":        "gpt-5.2",
		"input":        newRule,
		"instructions": systemPrompt,
	}

	// Converts the request data into json.
	jsonBody, err := json.Marshal(requestData)
	if err != nil {
		log.Fatal(err)
	}

	// Instantiates an HTTP request object using OpenAI's responses API, passing our data as the body in the POST request.
	baseURL := "https://api.openai.com/v1/responses"
	client := &http.Client{}
	req, err := http.NewRequest("POST", baseURL, bytes.NewBuffer(jsonBody))
	if err != nil {
		panic(err)
	}

	// Sets the Authorization header to our OpenAI token as a bearer token for authorization/authentication.
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", openAIToken))

	// Sends the request to OpenAI's API.
	resp, err := client.Do(req)
	if err != nil {
		panic(err)
	}

	defer func() {
		if err := resp.Body.Close(); err != nil {
			panic(err)
		}
	}()

	// Retrieves ChatGPT's response and converts it to a string.
	body, err := io.ReadAll((resp.Body))
	if err != nil {
		panic(err)
	}
	stringBody := string(body)

	// Unmarshal's GPT's response into structs that can pull the new rule code out easily.
	var gptData map[string]any
	err = json.Unmarshal([]byte(stringBody), &gptData)
	if err != nil {
		fmt.Println(stringBody)
		panic(err)
	}

	var gptResponse GPTResponseObject
	err = mapstructure.Decode(gptData, &gptResponse)
	if err != nil {
		panic(err)
	}

	// If ChatGPT determined that the requested rule could not be generated (as defined in the system prompt), returns false.
	// Else, append rule to corresponding file and add function name to rules list.
	if gptResponse.Output[0].Content[0].Text == "Can not generate" {
		return false
	} else {
		room.updateFile(gptResponse.Output[0].Content[0].Text)
		return true
	}
}

// Updates the given room's rules file with the new rule generated by GPT.
func (room *Room) updateFile(newRule string) {
	fileName := fmt.Sprintf("rules/rules_%s.txt", room.RoomCode)
	currentRulesBytes, err := os.ReadFile(fileName)
	if err != nil {
		panic(err)
	}

	// There is a slice in the rules file that holds each function name in a slice. This retrieves that slice definition.
	currentRules := strings.Split(string(currentRulesBytes), "\n")
	rulesSlice := currentRules[4]

	// Uses regex to search GPT's new rule and grabs the function name.
	re := regexp.MustCompile(`^func ([a-zA-Z0-9-]+)\(`)
	functionName := re.FindStringSubmatch(newRule)[1]

	// Adds the new function name to the rules slice in the file.
	newRulesSlice := rulesSlice[0:28] + functionName + ", " + rulesSlice[28:]
	currentRules[4] = newRulesSlice

	// Writes the final rules code back onto the file.
	finalFile := strings.Join(append(currentRules, strings.Split(newRule, "\n")...), "\n") + "\n"
	err = os.WriteFile(fileName, []byte(finalFile), 0o644)
	if err != nil {
		log.Fatalf("Failed to write to file: %v", err)
	}
}

// Gets the player who won the game. If there is a tie, throws an error.
func (room *Room) getWinningPlayer() (Player, error) {
	maxWins := 0
	numWinners := 0

	var winningPlayer Player
	for _, player := range room.Players {
		if player.Wins > maxWins {
			maxWins = player.Wins
			winningPlayer = *player
			numWinners = 1
		} else if player.Wins == maxWins {
			numWinners += 1
		}
	}

	if numWinners == 1 {
		return winningPlayer, nil
	} else {
		return winningPlayer, errors.New("tied winners")
	}
}

// Removes the passed player from the requested room.
func leaveRoom(rooms map[string]*Room, room *Room, player Player) {
	// If it is currently the turn of the leaving player, pass the turn to the next player.
	room.Cond.L.Lock()
	if player.PlayerNumber == room.PlayerTurn {
		room.PlayerTurn = getNextTurn(room)
	}
	room.Cond.Signal()
	room.Cond.L.Unlock()

	// If the game has not started and the player is the creator, cancels the game.
	// If the game has started but the round hasn't, and the player is the one who starts the game between rounds, starts the game.
	if !room.GameStarted && player.CanStartGame {
		room.Cond.L.Lock()

		room.GameStarted = true
		room.RoundStarted = true
		deleteRoom(rooms, room)

		room.Cond.Signal()
		room.Cond.L.Unlock()
		return
	} else if !room.RoundStarted && player.CanStartGame {
		room.RoundStarted = true
	}

	// Adds the cards from the player's hand to the bottom of the draw pile.
	room.Deck.Pile = append(room.Deck.Pile, player.Hand.Cards...)
	delete(room.Players, player.PlayerID)

	// If the leaving player is the last in the room, deletes the entire room.
	if len(room.Players) == 0 {
		deleteRoom(rooms, room)
	}
}

// Deletes the room from the map of rooms by deleting its rules file along with its entry in the map.
func deleteRoom(rooms map[string]*Room, room *Room) {
	err := os.Remove(fmt.Sprintf("rules/rules_%s.txt", room.RoomCode))
	if err != nil {
		panic(err)
	}
	delete(rooms, room.RoomCode)
}

// Gets the room given a room code. Throws an error if it does not exist.
func getRoom(rooms map[string]*Room, roomCode string) (*Room, error) {
	if !slices.Contains(slices.Collect(maps.Keys(rooms)), roomCode) {
		return nil, errors.New("room not found")
	}
	return rooms[roomCode], nil
}

// Gets the player given a player ID and room code. Throws an error if the player does not exist in that room.
func getPlayer(room *Room, playerID string) (*Player, error) {
	if !slices.Contains(slices.Collect(maps.Keys(room.Players)), playerID) {
		return nil, errors.New("player not found in this room")
	}
	return room.Players[playerID], nil
}

// Gets the next player's turn. Does this by getting every player's player number and sorting them.
// It then uses the current player's turn and finds the index of this slice so it can get the next one (which loops using modulus).
// This is done this way in case players leave the room, as simply incrementing with modulus would still use players who have left.
func getNextTurn(room *Room) int {
	var playerNumbers []int
	for _, player := range room.Players {
		playerNumbers = append(playerNumbers, player.PlayerNumber)
	}
	sort.Ints(playerNumbers)

	playerPosition := (slices.Index(playerNumbers, room.PlayerTurn) + 1) % len(room.Players)
	return playerNumbers[playerPosition]
}

// The main driver of the program and manages client connections.
func main() {
	// If the user has not set their OpenAI API token, exit.
	if openAIToken == "" {
		fmt.Println("Please set your OpenAI token.")
		os.Exit(0)
	}

	// Instantiates the map of rooms for the server.
	rooms := make(map[string]*Room)

	// Sets up a listener on port 9090 to receive game data requests from the clients.
	listener, err := net.Listen("tcp", ":9090")
	if err != nil {
		panic(err)
	}
	defer func() {
		if closeErr := listener.Close(); closeErr != nil {
			panic(closeErr)
		}
	}()
	fmt.Println("Listening on all interfaces on port 9090")

	// Infinite loop to receive incoming client connections.
	for {
		conn, err := listener.Accept()
		if err != nil {
			panic(err)
		}

		go handleConnection(conn, rooms)
	}
}

// Handles client connections and performs actions based on the requested action from the client.
func handleConnection(conn net.Conn, rooms map[string]*Room) {
	defer func() {
		if closeErr := conn.Close(); closeErr != nil {
			panic(closeErr)
		}
	}()

	// Receives the client's data and converts it into a struct.
	// The requested action determines what the server does with the sent data.
	actionDetails := receiveData(conn)
	switch actionDetails.Action {

	// Simple health check action to see if the service is up.
	case "healthCheck":
		sendData(conn, []byte("{\"service\": \"Mao Game\", \"success\": true}\n"))

	// Gets the current game standings and sends the client each player's number of wins given a room code.
	case "getStats":
		room, err := getRoom(rooms, actionDetails.RoomCode)
		if err != nil {
			sendData(conn, []byte("{\"action\": \"getStats\", \"success\": false, \"message\": \"Room does not exist.\"}\n"))
			return
		}

		var playerStats []string
		for _, player := range room.Players {
			playerStats = append(playerStats, fmt.Sprintf("\"%d\": %d", player.PlayerNumber, player.Wins))
		}
		sendData(conn, fmt.Appendf(nil, "{\"stats\": {%s}}\n", strings.Join(playerStats, ", ")))

	// Creates a room based on the settings the client has sent the server. It also creates a new player and sends the player data back to the client.
	case "createRoom":
		// If the client tries to create a room with too many or too few cards as the initial hand size, rejects the creation of the room.
		if actionDetails.HandSize < 1 || actionDetails.HandSize > 15 {
			sendData(conn, []byte("{\"action\": \"createRoom\", \"success\": false, \"message\": \"Specified handsize not between 1 and 15.\"}\n"))
			return
		}

		// If the client tries to create a room with too many or too few rounds, rejects the creation of the room.
		if actionDetails.NumRounds < 1 || actionDetails.NumRounds > 10 {
			sendData(conn, []byte("{\"action\": \"createRoom\", \"success\": false, \"message\": \"Specified number of rounds not between 1 and 10.\"}\n"))
			return
		}

		// Generates a random room code and instantiates a room with the given settings from the client.
		roomCode := generateRoomCode()
		newRoom := Room{RoomCode: roomCode, Deck: initializeDeck(), HandSize: actionDetails.HandSize, GameStarted: false, RoundStarted: false, Players: make(map[string]*Player), IsPrivate: actionDetails.IsPrivate, CanAddRule: false, Round: 0, RoundCount: actionDetails.NumRounds}
		newRoom.Cond = sync.NewCond(&newRoom.Mu)

		// Creates the first player in the room with the ability to start the game.
		newPlayer := Player{PlayerID: uuid.NewString(), PlayerNumber: 0, CanStartGame: true}

		// Adds the new player to the room and adds the new room to the rooms map.
		newRoom.Players[newPlayer.PlayerID] = &newPlayer
		rooms[roomCode] = &newRoom

		// Copies the rules template into a new file for the newly created room and loads the string into the room object.
		copyRules(roomCode)
		newRoom.retrieveRules()

		// Sends the player data back to the client.
		sendData(conn, fmt.Appendf(nil, "{\"playerID\": \"%s\", \"playerNumber\": %d, \"canStartGame\": %s, \"roomCode\": \"%s\"}\n", newPlayer.PlayerID, newPlayer.PlayerNumber, strconv.FormatBool(newPlayer.CanStartGame), newRoom.RoomCode))

	// Creates a new player in the room if a client requsts to join.
	case "joinRoom":
		room, err := getRoom(rooms, actionDetails.RoomCode)
		if err != nil {
			sendData(conn, []byte("{\"action\": \"joinRoom\", \"success\": false, \"message\": \"Room does not exist.\"}\n"))
			return
		}

		// If the game has not started, creates a new player that can not start the game and adds them to the room.
		if !room.GameStarted {
			newPlayer := Player{PlayerID: uuid.NewString(), PlayerNumber: len(room.Players), CanStartGame: false}
			room.Players[newPlayer.PlayerID] = &newPlayer

			// Sends the player data back to the client.
			sendData(conn, fmt.Appendf(nil, "{\"action\": \"joinRoom\", \"success\": true, \"player\": {\"playerID\": \"%s\", \"playerNumber\": %d, \"canStartGame\": %s}}\n", newPlayer.PlayerID, newPlayer.PlayerNumber, strconv.FormatBool(newPlayer.CanStartGame)))
		} else {
			sendData(conn, []byte("{\"action\": \"joinRoom\", \"success\": false}\n"))
		}

	// Removes a player from the room if they request to leave.
	case "leaveRoom":
		room, err := getRoom(rooms, actionDetails.RoomCode)
		if err != nil {
			sendData(conn, []byte("{\"action\": \"leftRoom\", \"success\": false, \"message\": \"Room does not exist.\"}\n"))
			return
		}

		player, err := getPlayer(room, actionDetails.PlayerID)
		if err != nil {
			sendData(conn, []byte("{\"action\": \"leftRoom\", \"success\": false, \"message\": \"Player does not exist in this room.\"}\n"))
			return
		}

		leaveRoom(rooms, room, *player)

		sendData(conn, []byte("{\"action\": \"leftRoom\", \"success\": true}\n"))

	// Gets the list of public rooms and sends it back to the client.
	case "getRooms":
		roomCodes := []string{}
		for roomCode, room := range rooms {
			if !room.GameStarted && !room.IsPrivate {
				roomCodes = append(roomCodes, roomCode)
			}
		}
		sendData(conn, fmt.Appendf(nil, "{\"rooms\": \"%s\"}\n", strings.Join(roomCodes, " ")))

	// Holds a connection open for a client waiting for the game to start and returns their initial hand once it does.
	case "waitStart":
		room, err := getRoom(rooms, actionDetails.RoomCode)
		if err != nil {
			sendData(conn, []byte("{\"action\": \"waitStart\", \"success\": false, \"message\": \"Room does not exist.\"}\n"))
			return
		}

		player, err := getPlayer(room, actionDetails.PlayerID)
		if err != nil {
			sendData(conn, []byte("{\"action\": \"waitStart\", \"success\": false, \"message\": \"Player does not exist in this room.\"}\n"))
			return
		}

		// Waits for the room creator (or round winner) to start the game.
		room.Cond.L.Lock()
		for !room.RoundStarted {
			room.Cond.Wait()

			room, err := getRoom(rooms, actionDetails.RoomCode)
			if err != nil {
				sendData(conn, []byte("{\"action\": \"waitStart\", \"success\": false, \"message\": \"Room does not exist.\"}\n"))
				return
			}

			// Draws the player's hand for the new game/round and sends it back to the client.
			player.drawHand(room)
			cardList := player.listCards()
			sendData(conn, fmt.Appendf(nil, "{\"action\": \"waitStart\", \"success\": true, \"initialHand\": \"%s\"}\n", cardList))
		}

		room.Cond.Signal()
		room.Cond.L.Unlock()

	// Starts the game if the given player created the room or won the last round.
	case "startGame":
		room, err := getRoom(rooms, actionDetails.RoomCode)
		if err != nil {
			sendData(conn, []byte("{\"action\": \"startGame\", \"success\": false, \"message\": \"Room does not exist.\"}\n"))
			return
		}

		player, err := getPlayer(room, actionDetails.PlayerID)
		if err != nil {
			sendData(conn, []byte("{\"action\": \"startGame\", \"success\": false, \"message\": \"Player does not exist in this room.\"}\n"))
			return
		}

		// Checks to see if the given player can start the game and, if so, starts it and sends the player their drawn hand.
		room.Cond.L.Lock()
		if player.CanStartGame && !room.RoundStarted {
			room.GameStarted = true
			room.RoundStarted = true

			player.drawHand(room)
			cardList := player.listCards()
			sendData(conn, fmt.Appendf(nil, "{\"action\": \"startGame\", \"success\": true, \"initialHand\": \"%s\"}\n", cardList))
		} else {
			sendData(conn, []byte("{\"gameStarted\": \"false\", \"message\": \"Can not start game\"}\n"))
		}

		room.Cond.Signal()
		room.Cond.L.Unlock()

	// Cancels the game of a room if the player can start the game.
	case "cancelGame":
		room, err := getRoom(rooms, actionDetails.RoomCode)
		if err != nil {
			sendData(conn, []byte("{\"action\": \"cancelGame\", \"success\": false, \"message\": \"Room does not exist.\"}\n"))
			return
		}

		player, err := getPlayer(room, actionDetails.PlayerID)
		if err != nil {
			sendData(conn, []byte("{\"action\": \"cancelGame\", \"success\": false, \"message\": \"Player does not exist in this room.\"}\n"))
			return
		}

		room.Cond.L.Lock()
		if player.CanStartGame && !room.GameStarted {
			room.GameStarted = true
			room.RoundStarted = true
			deleteRoom(rooms, room)
			sendData(conn, []byte("{\"action\": \"cancelGame\", \"success\": true}\n"))
		} else {
			sendData(conn, []byte("{\"action\": \"cancelGame\", \"success\": false, \"message\": \"Player did not create room.\"}\n"))
		}
		room.Cond.Signal()
		room.Cond.L.Unlock()

	// Determines if a player can play a card (or draw) and updates the player's hand accordingly.
	case "playCard":
		room, err := getRoom(rooms, actionDetails.RoomCode)
		if err != nil {
			sendData(conn, []byte("{\"action\": \"playCard\", \"success\": false, \"message\": \"Room does not exist.\"}\n"))
			return
		}

		player, err := getPlayer(room, actionDetails.PlayerID)
		if err != nil {
			sendData(conn, []byte("{\"action\": \"playCard\", \"success\": false, \"message\": \"Player does not exist in this room.\"}\n"))
			return
		}

		room.Cond.L.Lock()

		// If it is the current player's turn, continue.
		playedCard := actionDetails.Card
		if player.PlayerNumber == room.PlayerTurn {
			// If the player chose to draw, draw a card and respond to the client with the updated hand.
			if playedCard == "draw" {
				player.Hand.drawCard(&room.Deck)

				cardList := player.listCards()
				sendData(conn, fmt.Appendf(nil, "{\"round\": %d, \"rulesPassed\": true, \"currentHand\": \"%s\", \"wonRound\": false, \"winner\": -1}\n", room.Round, cardList))
				room.PlayerTurn = getNextTurn(room)
			} else if rulesCheck(*player, playedCard, room) {
				// If the player played a card that passed the rules of the room, pop the card from the player's hand, along with updating the deck's discard pile.
				popIndex := slices.Index(player.Hand.Cards, convertCard(playedCard))

				room.Deck.DiscardPile = append(room.Deck.DiscardPile, player.Hand.Cards[popIndex])
				player.Hand.Cards = slices.Delete(player.Hand.Cards, popIndex, popIndex+1)

				// If the player still has cards in their hand, send the updated hand back to the client.
				cardList := player.listCards()
				if len(player.Hand.Cards) > 0 {
					sendData(conn, fmt.Appendf(nil, "{\"round\": %d, \"rulesPassed\": true, \"currentHand\": \"%s\", \"wonRound\": false, \"winner\": -1}\n", room.Round, cardList))
				} else {
					// If the player has no cards in hand, then they won the round. Initialize a new deck for the next round and set other room states.
					player.Wins += 1
					room.RoundStarted = false
					room.CanAddRule = true
					room.Round += 1
					room.Deck = initializeDeck()

					// Reduces each player's hand to zero cards and sets the winning player to be able to start the game.
					for _, allPlayer := range room.Players {
						allPlayer.Hand.Cards = allPlayer.Hand.Cards[:0]
						if allPlayer.CanStartGame {
							allPlayer.CanStartGame = false
						}
					}
					player.CanStartGame = true

					// If there are more rounds to be played, respond to the client that they won the round, but without a game winner.
					if room.Round < room.RoundCount {
						sendData(conn, fmt.Appendf(nil, "{\"round\": %d, \"rulesPassed\": true, \"currentHand\": \"%s\", \"wonRound\": true, \"winner\": -1}\n", room.Round, cardList))
					} else {
						// If there are no more rounds to be played, get the winning player.
						winningPlayer, err := room.getWinningPlayer()
						if err != nil {
							// If there is an error, it means that there was a tie. The game goes to an overtime round and continues like normal.
							room.RoundCount += 1
							sendData(conn, fmt.Appendf(nil, "{\"round\": %d, \"rulesPassed\": true, \"currentHand\": \"%s\", \"wonRound\": true, \"winner\": -1}\n", room.Round, cardList))
						} else {
							// If here is no tie, then the server responds to the client with the final stats along with the winner.
							var playerStats []string
							for _, player := range room.Players {
								playerStats = append(playerStats, fmt.Sprintf("\"%d\": %d", player.PlayerNumber, player.Wins))
							}

							deleteRoom(rooms, room)
							sendData(conn, fmt.Appendf(nil, "{\"round\": %d, \"rulesPassed\": true, \"currentHand\": \"%s\", \"wonRound\": true, \"winner\": %d, \"stats\": {%s}}\n", room.Round, cardList, winningPlayer.PlayerNumber, strings.Join(playerStats, ", ")))
						}
					}
				}
				// Regardless of what happens, the turn is passed to the next player.
				room.PlayerTurn = getNextTurn(room)
			} else {
				// If the played card broke a rule, the player draws a card and plays again.
				player.Hand.drawCard(&room.Deck)

				cardList := player.listCards()
				sendData(conn, fmt.Appendf(nil, "{\"round\": %d, \"rulesPassed\": false, \"currentHand\": \"%s\", \"wonRound\": false, \"winner\": -1}\n", room.Round, cardList))
			}
		} else {
			// Responds with a failure if a player tried to play when it wasn't there turn.
			sendData(conn, []byte("{\"playCard\": \"false\", \"message\": \"It is not this player's turn\"}\n"))
		}

		room.Cond.Signal()
		room.Cond.L.Unlock()

	// Responds to the client with whose turn it is.
	case "requestTurn":
		room, err := getRoom(rooms, actionDetails.RoomCode)
		if err != nil {
			sendData(conn, []byte("{\"action\": \"requestTurn\", \"success\": false, \"message\": \"Room does not exist.\"}\n"))
			return
		}

		sendData(conn, fmt.Appendf(nil, "{\"playerNumber\": %d, \"topCard\": \"%s\"}\n", room.PlayerTurn, room.Deck.getCurrentTop().Value))

	// Holds a connection open for a client waiting, for the current player to play their turn. Sends game state data when they finish.
	case "waitTurn":
		room, err := getRoom(rooms, actionDetails.RoomCode)
		if err != nil {
			sendData(conn, []byte("{\"action\": \"waitTurn\", \"success\": false, \"message\": \"Room does not exist.\"}\n"))
			return
		}

		// Waits for the next turn.
		currentPlayer := room.PlayerTurn
		room.Cond.L.Lock()
		for currentPlayer == room.PlayerTurn {
			room.Cond.Wait()
		}

		// Checks to see if the current player has zero cards left in hand.
		for _, player := range room.Players {
			if currentPlayer == player.PlayerNumber {
				if len(player.Hand.Cards) == 0 {
					// If there are more rounds to be played, respond to the client that the last player won the round, but without a game winner.
					if room.Round < room.RoundCount {
						sendData(conn, fmt.Appendf(nil, "{\"round\": %d, \"wonRound\": true, \"winningRoundPlayer\": %d, \"winningGamePlayer\": -1}\n", room.Round, currentPlayer+1))
					} else {
						// If there are no more rounds to be played, get the winning player.
						winningPlayer, err := room.getWinningPlayer()
						var playerStats []string
						for _, player := range room.Players {
							playerStats = append(playerStats, fmt.Sprintf("\"%d\": %d", player.PlayerNumber, player.Wins))
						}

						// If there is an error, it means that there was a tie. The game goes to an overtime round and continues like normal.
						// Doesn't increase round count here because playCard block handles it.
						if err != nil {
							sendData(conn, fmt.Appendf(nil, "{\"round\": %d, \"wonRound\": true, \"winningRoundPlayer\": %d, \"winningGamePlayer\": -1}\n", room.Round, currentPlayer+1))
						} else {
							// If here is no tie, then the server responds to the client with the final stats along with the winner.
							sendData(conn, fmt.Appendf(nil, "{\"round\": %d, \"wonRound\": true, \"winningRoundPlayer\": %d, \"winningGamePlayer\": %d, \"stats\": {%s}}\n", room.Round, currentPlayer+1, winningPlayer.PlayerNumber, strings.Join(playerStats, ", ")))
						}
					}
				}
				break
			}
		}

		room.Cond.Signal()
		room.Cond.L.Unlock()

		sendData(conn, fmt.Appendf(nil, "{\"round\": %d, \"wonRound\": false, \"winningGamePlayer\": -1}\n", room.Round))

	// Allows the client to add a rule to the room if they won the round.
	case "addRule":
		room, err := getRoom(rooms, actionDetails.RoomCode)
		if err != nil {
			sendData(conn, []byte("{\"action\": \"addRule\", \"success\": false, \"message\": \"Room does not exist.\"}\n"))
			return
		}

		player, err := getPlayer(room, actionDetails.PlayerID)
		if err != nil {
			sendData(conn, []byte("{\"action\": \"addRule\", \"success\": false, \"message\": \"Player does not exist in this room.\"}\n"))
			return
		}

		newRule := actionDetails.NewRule

		// Validate the received player actually won by checking their card count.
		// Also uses a room attribute to determine if a rule has been added yet this round. That way, multiple rules can't be added per won round.
		if len(player.Hand.Cards) == 0 && room.CanAddRule {
			room.CanAddRule = false
			success := room.addRule(newRule)

			// If the rule was successfully added, reload the rule code text into the room's attribute.
			if success {
				room.retrieveRules()
			}

			sendData(conn, fmt.Appendf(nil, "{\"action\": \"addRule\", \"success\": %t}\n", success))
		} else {
			sendData(conn, []byte("{\"action\": \"addRule\", \"success\": false}\n"))
		}
	default:
		fmt.Printf("Invalid request: %v\n", actionDetails)
	}
}
