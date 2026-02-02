package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"math/rand/v2"
	"net"
	"os"
	"strconv"
	"strings"
	"sync"

	"github.com/go-viper/mapstructure/v2"
	"github.com/google/uuid"
	"github.com/traefik/yaegi/interp"
	"github.com/traefik/yaegi/stdlib"
)

type Rules func(card Card, deck Deck) string

type Room struct {
	RoomCode   string
	Players    map[string]*Player
	Deck       Deck
	IsStarted  bool
	DrawTurn   int
	PlayerTurn int
	Rules      []Rules
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
	PlayerID string
	Action   string
	RoomCode string
	Card     string
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

// The base "UNO" rule
func baseRule(card Card, deck Deck) string {
	currentCard := deck.getCurrentTop()

	if card.Rank == currentCard.Rank {
		return "true"
	} else if card.Suit == currentCard.Suit {
		return "true"
	} else {
		return "false"
	}
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
	// NEED TO SEPARATE BY ROOM. Maybe on room creation, copy from this template rules.txt and then make rules_ABCD.txt or something for the AI to update
	// Then would obviously read from the room specific file. On game completion, delete rules file
	content, err := os.ReadFile("rules.txt")
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
	hand := make([]Card, 7)
	copy(hand, room.Deck.Pile[:7])

	player.Hand.Cards = hand
	room.Deck.Pile = room.Deck.Pile[7:]
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

func main() {
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
	case "createRoom":
		roomCode := generateRoomCode()
		newRoom := Room{RoomCode: roomCode, Deck: initializeDeck(), IsStarted: false, Players: make(map[string]*Player), Rules: []Rules{baseRule}}
		newRoom.Cond = sync.NewCond(&newRoom.Mu)

		newPlayer := Player{PlayerID: uuid.NewString(), PlayerNumber: 0, CanStartGame: true}
		newPlayer.drawHand(&newRoom)

		newRoom.Players[newPlayer.PlayerID] = &newPlayer
		rooms[roomCode] = &newRoom

		sendData(conn, fmt.Appendf(nil, "{\"playerID\": \"%s\", \"playerNumber\": %d, \"canStartGame\": %s, \"roomCode\": \"%s\"}\n", newPlayer.PlayerID, newPlayer.PlayerNumber, strconv.FormatBool(newPlayer.CanStartGame), newRoom.RoomCode))
	case "joinRoom":
		// Need to validate room code is valid
		room := rooms[actionDetails.RoomCode]

		newPlayer := Player{PlayerID: uuid.NewString(), PlayerNumber: len(room.Players), CanStartGame: false}
		newPlayer.drawHand(room)

		room.Players[newPlayer.PlayerID] = &newPlayer

		sendData(conn, fmt.Appendf(nil, "{\"playerID\": \"%s\", \"playerNumber\": %d, \"canStartGame\": %s}\n", newPlayer.PlayerID, newPlayer.PlayerNumber, strconv.FormatBool(newPlayer.CanStartGame)))
	case "getRooms":
		roomCodes := []string{}
		for roomCode, room := range rooms {
			if !room.IsStarted {
				roomCodes = append(roomCodes, roomCode)
			}
		}
		sendData(conn, fmt.Appendf(nil, "{\"rooms\": \"%s\"}\n", strings.Join(roomCodes, " ")))
	case "waitingStart":
		// Need to validate roomCode and playerID meets a regex check (and playerID is in the room)
		room := rooms[actionDetails.RoomCode]
		player := room.Players[actionDetails.PlayerID]

		room.Cond.L.Lock()
		for !room.IsStarted {
			room.Cond.Wait()
		}

		cardList := player.listCards()
		sendData(conn, fmt.Appendf(nil, "{\"initialHand\": \"%s\"}\n", cardList))

		room.Cond.Signal()
		room.Cond.L.Unlock()
	case "startGame":
		// Need to validate roomCode and playerID meets a regex check (and playerID is in the room)
		room := rooms[actionDetails.RoomCode]
		player := room.Players[actionDetails.PlayerID]

		// Checks to see if the given player can start the game and, if so, does so
		room.Cond.L.Lock()
		if player.CanStartGame {
			room.IsStarted = true
			cardList := player.listCards()
			sendData(conn, fmt.Appendf(nil, "{\"initialHand\": \"%s\"}\n", cardList))
		} else {
			sendData(conn, []byte("{\"gameStarted\": \"false\", \"message\": \"Only first player can start the game\"}\n"))
		}

		room.Cond.Signal()
		room.Cond.L.Unlock()
	case "playCard":
		// Need to validate roomCode and playerID meets a regex check (and playerID is in the room)
		room := rooms[actionDetails.RoomCode]
		player := room.Players[actionDetails.PlayerID]
		playedCard := actionDetails.Card

		room.Cond.L.Lock()
		if player.PlayerNumber == room.PlayerTurn {
			if playedCard == "draw" {
				player.Hand.drawCard(&room.Deck)

				cardList := player.listCards()
				sendData(conn, fmt.Appendf(nil, "{\"rulesPassed\": true, \"currentHand\": \"%s\", \"wonGame\": false}\n", cardList))
				room.PlayerTurn = (room.PlayerTurn + 1) % len(room.Players) // Will need to change this to cycling through a slice for if a player leaves the room
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
					sendData(conn, fmt.Appendf(nil, "{\"rulesPassed\": true, \"currentHand\": \"%s\", \"wonGame\": false}\n", cardList))
				} else {
					sendData(conn, fmt.Appendf(nil, "{\"rulesPassed\": true, \"currentHand\": \"%s\", \"wonGame\": true}\n", cardList))
				}
				room.PlayerTurn = (room.PlayerTurn + 1) % len(room.Players) // Will need to change this to cycling through a slice for if a player leaves the room
			} else {
				player.Hand.drawCard(&room.Deck)

				cardList := player.listCards()
				sendData(conn, fmt.Appendf(nil, "{\"rulesPassed\": false, \"currentHand\": \"%s\", \"wonGame\": false}\n", cardList))
			}
		} else {
			sendData(conn, []byte("{\"playCard\": \"false\", \"message\": \"It is not this player's turn\"}\n"))
		}

		room.Cond.Signal()
		room.Cond.L.Unlock()
	case "requestTurn":
		// Need to validate roomCode
		room := rooms[actionDetails.RoomCode]
		sendData(conn, fmt.Appendf(nil, "{\"playerNumber\": %d, \"topCard\": \"%s\"}\n", room.PlayerTurn, room.Deck.getCurrentTop().Value))
	case "waitTurn":
		// Need to validate roomCode
		room := rooms[actionDetails.RoomCode]
		currentPlayer := room.PlayerTurn

		room.Cond.L.Lock()
		for currentPlayer == room.PlayerTurn {
			room.Cond.Wait()
		}

		for _, player := range room.Players {
			if currentPlayer == player.PlayerNumber {
				if len(player.Hand.Cards) == 0 {
					sendData(conn, fmt.Appendf(nil, "{\"wonGame\": true, \"winningPlayer\": \"%d\"}\n", currentPlayer+1))
				}
				break
			}
		}

		room.Cond.Signal()
		room.Cond.L.Unlock()
		sendData(conn, []byte("{\"wonGame\": false}\n"))
	default:
		fmt.Printf("Received message: %v\n", actionDetails)
	}
}
