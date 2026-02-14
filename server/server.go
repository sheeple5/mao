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

var (
	OPENAI_TOKEN string = os.Getenv("OPENAI_TOKEN")
	instructions string = `You are a dynamic programmer that exists as part of a game called Mao. In this game, a standard deck
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

type Room struct {
	RoomCode   string
	Players    map[string]*Player
	Deck       Deck
	HandSize   int
	IsStarted  bool
	IsPrivate  bool
	DrawTurn   int
	PlayerTurn int
	Round      int
	RoundCount int
	Mu         sync.Mutex
	Cond       *sync.Cond
}

type Deck struct {
	Pile        []Card
	DiscardPile []Card
}

type Player struct {
	Hand         Hand
	PlayerID     string
	PlayerNumber int
	CanStartGame bool
	Wins         int
}

type Hand struct {
	Cards []Card
}

type Card struct {
	Rank  string
	Suit  string
	Value string
}

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

func sendData(conn net.Conn, payload []byte) {
	_, err := conn.Write(payload)
	if err != nil {
		panic(err)
	}
}

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

	// Loads the JSON data into the struct using mapstructure
	err = mapstructure.Decode(actionData, &actionDetails)
	if err != nil {
		panic(err)
	}
	return actionDetails
}

func rulesCheck(player Player, playedCard string, room *Room) bool {
	// Check if card is actually in hand
	foundCard := false
	for _, card := range player.Hand.Cards {
		if playedCard == card.Value {
			foundCard = true
			break
		}
	}
	if !foundCard {
		return false
	}

	// Can probably change this to only read file on startup and when a new rule is added, not every time
	content, err := os.ReadFile(fmt.Sprintf("rules/rules_%s.txt", room.RoomCode))
	if err != nil {
		// Log the error and exit if file reading fails
		log.Fatal(err)
	}

	interpreter := interp.New(interp.Options{})
	err = interpreter.Use(stdlib.Symbols)
	if err != nil {
		panic(err)
	}

	_, err = interpreter.Eval(string(content))
	if err != nil {
		panic(err)
	}

	converter, err := interpreter.Eval("rules.checkRules")
	if err != nil {
		panic(err)
	}

	rulesCheck := converter.Interface().(func(string, string, string) bool)
	return rulesCheck(playedCard, room.Deck.listCards("pile"), room.Deck.listCards("discard"))
}

func generateRoomCode() string {
	var roomCode bytes.Buffer
	for range 4 {
		randomLetter := rand.IntN(26) + 65
		roomCode.WriteString(string(rune(randomLetter)))
	}
	return roomCode.String()
}

func copyRules(roomCode string) {
	sourceFile, err := os.Open("rules/rules_template.txt")
	if err != nil {
		panic(err)
	}
	defer sourceFile.Close()

	destinationFile, err := os.Create(fmt.Sprintf("rules/rules_%s.txt", roomCode))
	if err != nil {
		panic(err)
	}
	defer destinationFile.Close()

	_, err = io.Copy(destinationFile, sourceFile)
	if err != nil {
		panic(err)
	}

	err = destinationFile.Sync()
	if err != nil {
		panic(err)
	}
}

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

func (player *Player) drawHand(room *Room) {
	hand := make([]Card, room.HandSize)

	if len(room.Deck.Pile) < room.HandSize {
		newDeck := initializeDeck()
		room.Deck.Pile = append(room.Deck.Pile, newDeck.Pile...)
		room.Deck.Pile = append(room.Deck.Pile, newDeck.DiscardPile...)
	}

	copy(hand, room.Deck.Pile[:room.HandSize])

	player.Hand.Cards = hand
	room.Deck.Pile = room.Deck.Pile[room.HandSize:]
}

func (player Player) listCards() string {
	cardValues := []string{}
	for _, card := range player.Hand.Cards {
		cardValues = append(cardValues, card.Value)
	}
	return strings.Join(cardValues, ", ")
}

func initializeDeck() Deck {
	pile := []Card{}
	for _, suit := range []string{"S", "C", "H", "D"} {
		for _, rank := range []string{"A", "K", "Q", "J", "10", "9", "8", "7", "6", "5", "4", "3", "2"} {
			newCard := Card{Rank: rank, Suit: suit, Value: rank + suit}
			pile = append(pile, newCard)
		}
	}

	// Shuffle cards
	shuffleCards(pile)

	// Start discard pile
	discardPile := []Card{pile[0]}
	pile = pile[1:]

	return Deck{pile, discardPile}
}

func (deck Deck) getCurrentTop() Card {
	return deck.DiscardPile[len(deck.DiscardPile)-1]
}

func (deck Deck) listCards(pileType string) string {
	cardValues := []string{}
	var pile []Card
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

func shuffleCards(pile []Card) {
	rand.Shuffle(len(pile), func(i, j int) {
		pile[i], pile[j] = pile[j], pile[i]
	})
}

func addRule(room *Room, newRule string) bool {
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

	requestData := map[string]string{
		"model":        "gpt-5.2",
		"input":        newRule,
		"instructions": instructions,
	}
	jsonBody, err := json.Marshal(requestData)
	if err != nil {
		log.Fatal(err)
	}

	baseURL := "https://api.openai.com/v1/responses"
	client := &http.Client{}
	req, err := http.NewRequest("POST", baseURL, bytes.NewBuffer(jsonBody))
	if err != nil {
		panic(err)
	}
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", OPENAI_TOKEN))

	resp, err := client.Do(req)
	if err != nil {
		panic(err)
	}

	defer func() {
		if err := resp.Body.Close(); err != nil {
			panic(err)
		}
	}()

	body, err := io.ReadAll((resp.Body))
	if err != nil {
		panic(err)
	}
	stringBody := string(body)

	// Converts the received JSON data and converts it into a map
	var gptData map[string]any
	err = json.Unmarshal([]byte(stringBody), &gptData)
	if err != nil {
		fmt.Println(stringBody)
		panic(err)
	}

	// Loads the JSON data into the weather structs using mapstructure
	var gptResponse GPTResponseObject
	err = mapstructure.Decode(gptData, &gptResponse)
	if err != nil {
		panic(err)
	}

	if gptResponse.Output[0].Content[0].Text == "Can not generate" {
		return false
	} else {
		// Append rule to corresponding file and add function name to rules list
		updateFile(room, gptResponse.Output[0].Content[0].Text)
		return true
	}
}

func updateFile(room *Room, gptRule string) {
	fileName := fmt.Sprintf("rules/rules_%s.txt", room.RoomCode)
	currentRulesBytes, err := os.ReadFile(fileName)
	if err != nil {
		panic(err)
	}

	currentRules := strings.Split(string(currentRulesBytes), "\n")
	rulesSlice := currentRules[4]

	re := regexp.MustCompile(`^func ([a-zA-Z0-9-]+)\(`)
	functionName := re.FindStringSubmatch(gptRule)[1]
	newRulesSlice := rulesSlice[0:28] + functionName + ", " + rulesSlice[28:]
	currentRules[4] = newRulesSlice

	finalFile := strings.Join(append(currentRules, strings.Split(gptRule, "\n")...), "\n") + "\n"
	err = os.WriteFile(fileName, []byte(finalFile), 0o644)
	if err != nil {
		log.Fatalf("Failed to write to file: %v", err)
	}
}

func leaveRoom(rooms map[string]*Room, room *Room, player Player) {
	room.Cond.L.Lock()
	if player.PlayerNumber == room.PlayerTurn {
		room.PlayerTurn = getNextTurn(room)
	}
	room.Cond.Signal()
	room.Cond.L.Unlock()

	if !room.IsStarted && player.CanStartGame {
		room.IsStarted = true
	}

	room.Deck.Pile = append(room.Deck.Pile, player.Hand.Cards...)
	delete(room.Players, player.PlayerID)

	if len(room.Players) == 0 {
		deleteRoom(rooms, room)
	}
}

func deleteRoom(rooms map[string]*Room, room *Room) {
	err := os.Remove(fmt.Sprintf("rules/rules_%s.txt", room.RoomCode))
	if err != nil {
		panic(err)
	}
	delete(rooms, room.RoomCode)
}

func getWinningPlayer(room *Room) (Player, error) {
	maxWins := 0
	maxCount := 0
	var maxPlayer Player
	for _, allPlayer := range room.Players {
		if allPlayer.Wins > maxWins {
			maxWins = allPlayer.Wins
			maxPlayer = *allPlayer
			maxCount = 1
		} else if allPlayer.Wins == maxWins {
			maxCount += 1
		}
	}

	if maxCount == 1 {
		return maxPlayer, nil
	} else {
		return maxPlayer, errors.New("tied winners")
	}
}

func getRoom(rooms map[string]*Room, roomCode string) (*Room, error) {
	if !slices.Contains(slices.Collect(maps.Keys(rooms)), roomCode) {
		return nil, errors.New("room not found")
	}
	return rooms[roomCode], nil
}

func getPlayer(room *Room, playerID string) (*Player, error) {
	if !slices.Contains(slices.Collect(maps.Keys(room.Players)), playerID) {
		return nil, errors.New("player not found in this room")
	}
	return room.Players[playerID], nil
}

func getNextTurn(room *Room) int {
	var playerNumbers []int
	for _, player := range room.Players {
		playerNumbers = append(playerNumbers, player.PlayerNumber)
	}
	sort.Ints(playerNumbers)

	playerPosition := (slices.Index(playerNumbers, room.PlayerTurn) + 1) % len(room.Players)
	return playerNumbers[playerPosition]
}

func main() {
	if OPENAI_TOKEN == "" {
		fmt.Println("Please set your OpenAI token.")
		os.Exit(0)
	}
	rooms := make(map[string]*Room)

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

	for {
		conn, err := listener.Accept()
		if err != nil {
			panic(err)
		}

		go handleConnection(conn, rooms)
	}
}

func handleConnection(conn net.Conn, rooms map[string]*Room) {
	defer func() {
		if closeErr := conn.Close(); closeErr != nil {
			panic(closeErr)
		}
	}()

	actionDetails := receiveData(conn)
	switch actionDetails.Action {
	case "healthCheck":
		sendData(conn, []byte("{\"service\": \"Mao Game\", \"success\": true}\n"))
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
	case "createRoom":
		if actionDetails.HandSize < 0 || actionDetails.HandSize > 15 {
			sendData(conn, []byte("{\"action\": \"createRoom\", \"success\": false, \"message\": \"Specified handsize not between 1 and 15.\"}\n"))
			return

		}

		if actionDetails.NumRounds < 0 || actionDetails.NumRounds > 10 {
			sendData(conn, []byte("{\"action\": \"createRoom\", \"success\": false, \"message\": \"Specified number of rounds not between 1 and 10.\"}\n"))
			return

		}

		roomCode := generateRoomCode()
		newRoom := Room{RoomCode: roomCode, Deck: initializeDeck(), HandSize: actionDetails.HandSize, IsStarted: false, Players: make(map[string]*Player), IsPrivate: actionDetails.IsPrivate, Round: 0, RoundCount: actionDetails.NumRounds}
		newRoom.Cond = sync.NewCond(&newRoom.Mu)

		newPlayer := Player{PlayerID: uuid.NewString(), PlayerNumber: 0, CanStartGame: true}

		newRoom.Players[newPlayer.PlayerID] = &newPlayer
		rooms[roomCode] = &newRoom

		copyRules(roomCode)

		sendData(conn, fmt.Appendf(nil, "{\"playerID\": \"%s\", \"playerNumber\": %d, \"canStartGame\": %s, \"roomCode\": \"%s\"}\n", newPlayer.PlayerID, newPlayer.PlayerNumber, strconv.FormatBool(newPlayer.CanStartGame), newRoom.RoomCode))
	case "joinRoom":
		if !slices.Contains(slices.Collect(maps.Keys(rooms)), actionDetails.RoomCode) {
			sendData(conn, []byte("{\"action\": \"joinRoom\", \"success\": false, \"message\": \"Room does not exist.\"}\n"))
			return
		}

		room := rooms[actionDetails.RoomCode]

		newPlayer := Player{PlayerID: uuid.NewString(), PlayerNumber: len(room.Players), CanStartGame: false}

		room.Players[newPlayer.PlayerID] = &newPlayer

		sendData(conn, fmt.Appendf(nil, "{\"action\": \"joinRoom\", \"success\": true, \"player\": {\"playerID\": \"%s\", \"playerNumber\": %d, \"canStartGame\": %s}}\n", newPlayer.PlayerID, newPlayer.PlayerNumber, strconv.FormatBool(newPlayer.CanStartGame)))
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
	case "getRooms":
		roomCodes := []string{}
		for roomCode, room := range rooms {
			if !room.IsStarted && !room.IsPrivate {
				roomCodes = append(roomCodes, roomCode)
			}
		}
		sendData(conn, fmt.Appendf(nil, "{\"rooms\": \"%s\"}\n", strings.Join(roomCodes, " ")))
	case "waitingStart":
		room, err := getRoom(rooms, actionDetails.RoomCode)
		if err != nil {
			sendData(conn, []byte("{\"action\": \"waitingStart\", \"success\": false, \"message\": \"Room does not exist.\"}\n"))
			return
		}

		player, err := getPlayer(room, actionDetails.PlayerID)
		if err != nil {
			sendData(conn, []byte("{\"action\": \"waitingStart\", \"success\": false, \"message\": \"Player does not exist in this room.\"}\n"))
			return
		}

		room.Cond.L.Lock()
		for !room.IsStarted {
			room.Cond.Wait()
		}

		player.drawHand(room)
		cardList := player.listCards()
		sendData(conn, fmt.Appendf(nil, "{\"initialHand\": \"%s\"}\n", cardList))

		room.Cond.Signal()
		room.Cond.L.Unlock()
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

		// Checks to see if the given player can start the game and, if so, does so
		room.Cond.L.Lock()
		if player.CanStartGame {
			room.IsStarted = true
			player.drawHand(room)
			cardList := player.listCards()
			sendData(conn, fmt.Appendf(nil, "{\"initialHand\": \"%s\"}\n", cardList))
		} else {
			sendData(conn, []byte("{\"gameStarted\": \"false\", \"message\": \"Only first player can start the game\"}\n"))
		}

		room.Cond.Signal()
		room.Cond.L.Unlock()
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

		playedCard := actionDetails.Card
		room.Cond.L.Lock()
		if player.PlayerNumber == room.PlayerTurn {
			if playedCard == "draw" {
				player.Hand.drawCard(&room.Deck)

				cardList := player.listCards()
				sendData(conn, fmt.Appendf(nil, "{\"round\": %d, \"rulesPassed\": true, \"currentHand\": \"%s\", \"wonRound\": false}\n", room.Round, cardList))
				room.PlayerTurn = getNextTurn(room)
			} else if rulesCheck(*player, playedCard, room) {
				// Pop card
				var popIndex int
				for i, card := range player.Hand.Cards {
					if card.Value == playedCard {
						popIndex = i
						break
					}
				}
				room.Deck.DiscardPile = append(room.Deck.DiscardPile, player.Hand.Cards[popIndex])

				poppedHand := make([]Card, 0, len(player.Hand.Cards)-1)
				poppedHand = append(poppedHand, player.Hand.Cards[:popIndex]...)
				poppedHand = append(poppedHand, player.Hand.Cards[popIndex+1:]...)
				player.Hand.Cards = poppedHand

				cardList := player.listCards()
				if len(player.Hand.Cards) > 0 {
					sendData(conn, fmt.Appendf(nil, "{\"round\": %d, \"rulesPassed\": true, \"currentHand\": \"%s\", \"wonRound\": false}\n", room.Round, cardList))
				} else {
					player.Wins += 1
					room.IsStarted = false
					room.Round += 1
					room.Deck = initializeDeck()

					for _, allPlayer := range room.Players {
						allPlayer.Hand.Cards = allPlayer.Hand.Cards[:0]
						if allPlayer.CanStartGame {
							allPlayer.CanStartGame = false
						}
					}
					player.CanStartGame = true

					if room.Round < room.RoundCount {
						sendData(conn, fmt.Appendf(nil, "{\"round\": %d, \"rulesPassed\": true, \"currentHand\": \"%s\", \"wonRound\": true}\n", room.Round, cardList))
					} else {
						winningPlayer, err := getWinningPlayer(room)

						// If err returns non-nil, it means there's a tie and another round must be played
						if err != nil {
							room.RoundCount += 1
							sendData(conn, fmt.Appendf(nil, "{\"round\": %d, \"rulesPassed\": true, \"currentHand\": \"%s\", \"wonRound\": true}\n", room.Round, cardList))
						} else {
							var playerStats []string
							for _, player := range room.Players {
								playerStats = append(playerStats, fmt.Sprintf("\"%d\": %d", player.PlayerNumber, player.Wins))
							}

							deleteRoom(rooms, room)
							sendData(conn, fmt.Appendf(nil, "{\"round\": %d, \"rulesPassed\": true, \"currentHand\": \"%s\", \"wonRound\": true, \"winner\": \"%d\", \"stats\": {%s}}\n", room.Round, cardList, winningPlayer.PlayerNumber, strings.Join(playerStats, ", ")))
						}
					}
				}
				room.PlayerTurn = getNextTurn(room)
			} else {
				player.Hand.drawCard(&room.Deck)

				cardList := player.listCards()
				sendData(conn, fmt.Appendf(nil, "{\"round\": %d, \"rulesPassed\": false, \"currentHand\": \"%s\", \"wonRound\": false}\n", room.Round, cardList))
			}
		} else {
			sendData(conn, []byte("{\"playCard\": \"false\", \"message\": \"It is not this player's turn\"}\n"))
		}

		room.Cond.Signal()
		room.Cond.L.Unlock()
	case "requestTurn":
		room, err := getRoom(rooms, actionDetails.RoomCode)
		if err != nil {
			sendData(conn, []byte("{\"action\": \"requestTurn\", \"success\": false, \"message\": \"Room does not exist.\"}\n"))
			return
		}

		sendData(conn, fmt.Appendf(nil, "{\"playerNumber\": %d, \"topCard\": \"%s\"}\n", room.PlayerTurn, room.Deck.getCurrentTop().Value))
	case "waitTurn":
		room, err := getRoom(rooms, actionDetails.RoomCode)
		if err != nil {
			sendData(conn, []byte("{\"action\": \"waitTurn\", \"success\": false, \"message\": \"Room does not exist.\"}\n"))
			return
		}

		currentPlayer := room.PlayerTurn
		room.Cond.L.Lock()
		for currentPlayer == room.PlayerTurn {
			room.Cond.Wait()
		}

		for _, player := range room.Players {
			if currentPlayer == player.PlayerNumber {
				if len(player.Hand.Cards) == 0 {
					if room.Round < room.RoundCount {
						sendData(conn, fmt.Appendf(nil, "{\"round\": %d, \"wonRound\": true, \"winningRoundPlayer\": \"%d\"}\n", room.Round, currentPlayer+1))
					} else {
						winningPlayer, err := getWinningPlayer(room)
						var playerStats []string
						for _, player := range room.Players {
							playerStats = append(playerStats, fmt.Sprintf("\"%d\": %d", player.PlayerNumber, player.Wins))
						}

						// If err returns non-nil, it means there's a tie and another round must be played. Don't increase round count here cause playCard block handles it
						if err != nil {
							sendData(conn, fmt.Appendf(nil, "{\"round\": %d, \"wonRound\": true, \"winningRoundPlayer\": \"%d\"}\n", room.Round, currentPlayer+1))
						} else {
							sendData(conn, fmt.Appendf(nil, "{\"round\": %d, \"wonRound\": true, \"winningRoundPlayer\": \"%d\", \"winningGamePlayer\": \"%d\", \"stats\": {%s}}\n", room.Round, currentPlayer+1, winningPlayer.PlayerNumber, strings.Join(playerStats, ", ")))
						}
					}
				}
				break
			}
		}

		room.Cond.Signal()
		room.Cond.L.Unlock()
		sendData(conn, fmt.Appendf(nil, "{\"round\": %d, \"wonRound\": false}\n", room.Round))
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
		// Validate the received player actually won
		if len(player.Hand.Cards) == 0 {
			success := addRule(room, newRule)
			sendData(conn, fmt.Appendf(nil, "{\"success\": %t}\n", success))
		} else {
			sendData(conn, []byte("\"success\": false}\n"))
		}
	default:
		fmt.Printf("Received message: %v\n", actionDetails)
	}
}
