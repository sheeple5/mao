package main

import (
	"bufio"
	"fmt"
	"math/rand/v2"
	"net"
	"regexp"
	"strconv"
	"strings"
	"sync"

	"github.com/google/uuid"
)

type Rules func(card Card, deck Deck) string

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

func rulesCheck(player Player, playedCard string, room Room) bool {
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

	card := convertCard(playedCard)
	for _, rule := range room.Rules {
		ruleResult := rule(card, room.Deck)

		if ruleResult == "true" {
			return true
		} else if ruleResult == "false" {
			return false
		} else if ruleResult == "continue" {
			continue
		} else {
			continue
		}
	}
	return true
}

type Player struct {
	Hand         Hand
	PlayerID     string
	PlayerNumber int
	CanStartGame bool
}

type Room struct {
	RoomCode   string
	Players    map[string]Player
	Deck       Deck
	IsStarted  bool
	DrawTurn   int
	PlayerTurn int
	Rules      []Rules
}

type Card struct {
	Rank  string
	Suit  string
	Value string
}

func convertCard(playedCard string) Card {
	re := regexp.MustCompile("((?:[AKQJ2-9]|10))([SCHD])")
	rank := re.FindStringSubmatch(playedCard)[1]
	suit := re.FindStringSubmatch(playedCard)[2]

	return Card{Rank: rank, Suit: suit, Value: playedCard}
}

type Deck struct {
	Pile        []*Card
	DiscardPile []*Card
}

type Hand struct {
	Cards []*Card
}

func (hand Hand) printHand() {
	fmt.Println("Hand: ")
	fmt.Println("----------------------")
	for i, card := range hand.Cards {
		if i != len(hand.Cards)-1 {
			fmt.Printf("%s, ", card.Value)
		} else {
			fmt.Printf("%s\n\n", card.Value)
		}
	}
}

func (hand *Hand) drawCard(deck *Deck) {
	if len(deck.Pile) == 0 {
		deck.Pile = deck.DiscardPile[:len(deck.DiscardPile)-1]
		shuffleCards(deck.Pile)
		deck.DiscardPile = deck.DiscardPile[len(deck.DiscardPile)-1:]
	}

	drawnCard := deck.Pile[0]
	hand.Cards = append(hand.Cards, drawnCard)
	deck.Pile = deck.Pile[1:]
}

func (player *Player) drawHand(room *Room) {
	player.Hand.Cards = room.Deck.Pile[:7]
	room.Deck.Pile = room.Deck.Pile[7:]
}

func shuffleCards(pile []*Card) {
	rand.Shuffle(len(pile), func(i, j int) {
		pile[i], pile[j] = pile[j], pile[i]
	})
}

func initializeDeck() Deck {
	// Load up card pile
	pile := []*Card{}
	for _, suit := range []string{"S", "C", "H", "D"} {
		for _, rank := range []string{"A", "K", "Q", "J", "10", "9", "8", "7", "6", "5", "4", "3", "2"} {
			newCard := Card{Rank: rank, Suit: suit, Value: rank + suit}
			pile = append(pile, &newCard)
		}
	}

	// Shuffle cards
	shuffleCards(pile)

	// Start discard pile
	discardPile := []*Card{pile[0]}
	pile = pile[1:]

	return Deck{pile, discardPile}
}

func (deck Deck) getCurrentTop() *Card {
	currentCard := deck.DiscardPile[len(deck.DiscardPile)-1]
	return currentCard
}

func (deck Deck) printCurrentTop() {
	currentCard := deck.getCurrentTop()

	fmt.Println("Current Card: ")
	fmt.Println("----------------------")
	fmt.Printf("%s\n\n", currentCard.Value)
}

func (deck Deck) printPile(pileType string) {
	if pileType == "pile" {
		fmt.Println("Pile: ")
		fmt.Println("----------------------")
		printPile(deck.Pile)
	} else if pileType == "discardPile" {
		fmt.Println("Discard Pile: ")
		fmt.Println("----------------------")
		printPile(deck.DiscardPile)
	}
}

func (deck Deck) printDeck() {
	deck.printPile("pile")
	deck.printPile("discardPile")
}

func printPile(pile []*Card) {
	for i, card := range pile {
		if i != len(pile)-1 {
			fmt.Printf("%s, ", card.Value)
		} else {
			fmt.Printf("%s\n\n", card.Value)
		}
	}
}

func (deck *Deck) drawHand() []string {
	var cardValues []string
	drawnCards := deck.Pile[:7]
	for _, card := range drawnCards {
		cardValues = append(cardValues, card.Value)
	}
	deck.Pile = deck.Pile[7:]
	return cardValues
}

var (
	mu   sync.Mutex
	cond = sync.NewCond(&mu)
)

func main() {
	// Create deck
	room := Room{Deck: initializeDeck(), IsStarted: false, Players: make(map[string]Player), Rules: []Rules{baseRule}}
	rooms := make(map[string]*Room)

	l, err := net.Listen("tcp", ":9090")
	if err != nil {
		panic(err)
	}
	defer l.Close()
	fmt.Println("Listening on all interfaces on port 9090")

	room.Deck.printDeck()
	for {
		conn, err := l.Accept()
		if err != nil {
			panic(err)
		}

		go handleConnection(conn, rooms)
	}
}

func handleConnection(conn net.Conn, rooms map[string]*Room) {
	defer conn.Close()

	netData, err := bufio.NewReader(conn).ReadString('\n')
	if err != nil {
		panic(err)
	}

	netData = strings.TrimSpace(netData)
	if netData == "{\"action\": \"createRoom\"}" {
		var roomCode string
		for range 4 {
			randomLetter := rand.IntN(26) + 65
			roomCode += string(randomLetter)
		}
		newRoom := Room{RoomCode: roomCode, Deck: initializeDeck(), IsStarted: false, Players: make(map[string]Player), Rules: []Rules{baseRule}}

		newPlayer := Player{PlayerID: uuid.NewString()}
		newPlayer.PlayerNumber = 0
		newPlayer.CanStartGame = true
		newPlayer.drawHand(&newRoom)

		newRoom.Players[newPlayer.PlayerID] = newPlayer
		rooms[roomCode] = &newRoom
		conn.Write([]byte(fmt.Sprintf("{\"playerID\": \"%s\", \"playerNumber\": %d, \"canStartGame\": %s, \"roomCode\": \"%s\"}\n", newPlayer.PlayerID, newPlayer.PlayerNumber, strconv.FormatBool(newPlayer.CanStartGame), newRoom.RoomCode)))
		return

	} else if ok, _ := regexp.MatchString("{\"joinRoom\": \"[A-Z]{4}\"}", netData); ok {
		re := regexp.MustCompile("{\"joinRoom\": \"([A-Z]{4})\"}")
		roomCode := re.FindStringSubmatch(netData)[1]
		room := rooms[roomCode]
		// Need to do a check if the game state has started or not. If yes, give player UUID and stuff. If no, return a can't join error
		// - OR Have a client send a create room message? And then that player controls the room?
		// Also need to check if any players have joined. If no, canStartGame is returned true. Else, false
		newPlayer := Player{PlayerID: uuid.NewString()}
		newPlayer.PlayerNumber = len(room.Players)
		newPlayer.CanStartGame = false
		newPlayer.drawHand(room)

		room.Players[newPlayer.PlayerID] = newPlayer
		conn.Write([]byte(fmt.Sprintf("{\"playerID\": \"%s\", \"playerNumber\": %d, \"canStartGame\": %s}\n", newPlayer.PlayerID, newPlayer.PlayerNumber, strconv.FormatBool(newPlayer.CanStartGame))))
		return
	} else if ok, _ := regexp.MatchString("\\{\"playerID\": \"[a-z0-9]{8}-[a-z0-9]{4}-[a-z0-9]{4}-[a-z0-9]{4}-[a-z0-9]{12}\", \"roomCode\": \"[A-Z]{4}\", \"action\": \"waitingStart\"\\}", netData); ok {
		re := regexp.MustCompile("\\{\"playerID\": \"([a-f0-9]{8}-[a-z0-9]{4}-[a-z0-9]{4}-[a-z0-9]{4}-[a-z0-9]{12})\", \"roomCode\": \"([A-Z]{4})\", \"action\": \"waitingStart\"\\}")
		receivedID := re.FindStringSubmatch(netData)[1]
		roomCode := re.FindStringSubmatch(netData)[2]
		room := rooms[roomCode]

		cond.L.Lock()
		for !room.IsStarted {
			fmt.Println("waiting")
			cond.Wait()
		}
		cardValues := []string{}
		for _, card := range room.Players[receivedID].Hand.Cards {
			cardValues = append(cardValues, card.Value)
		}
		conn.Write([]byte(fmt.Sprintf("{\"initialHand\": \"%s\"}\n", strings.Join(cardValues, ", "))))

		cond.Signal()
		cond.L.Unlock()
		return
	} else if ok, _ := regexp.MatchString("\\{\"playerID\": \"[a-z0-9]{8}-[a-z0-9]{4}-[a-z0-9]{4}-[a-z0-9]{4}-[a-z0-9]{12}\", \"roomCode\": \"[A-Z]{4}\", \"action\": \"startGame\"\\}", netData); ok {
		re := regexp.MustCompile("\\{\"playerID\": \"([a-f0-9]{8}-[a-z0-9]{4}-[a-z0-9]{4}-[a-z0-9]{4}-[a-z0-9]{12})\", \"roomCode\": \"([A-Z]{4})\", \"action\": \"startGame\"\\}")
		receivedID := re.FindStringSubmatch(netData)[1]
		roomCode := re.FindStringSubmatch(netData)[2]
		room := rooms[roomCode]

		// Checks to see if a real player's UUID was received
		cond.L.Lock()
		if _, ok := room.Players[receivedID]; !ok {
			conn.Write([]byte("{\"gameStarted\": \"false\", \"message\": \"Did not receive a valid UUID\"}\n"))
			return
		}

		if room.Players[receivedID].CanStartGame {
			room.IsStarted = true
			cardValues := []string{}
			for _, card := range room.Players[receivedID].Hand.Cards {
				cardValues = append(cardValues, card.Value)
			}
			conn.Write([]byte(fmt.Sprintf("{\"initialHand\": \"%s\"}\n", strings.Join(cardValues, ", "))))
		} else {
			conn.Write([]byte("{\"gameStarted\": \"false\", \"message\": \"Only first player can start the game\"}\n"))
		}
		cond.Signal()
		cond.L.Unlock()
		return
	} else if ok, _ := regexp.MatchString("\\{\"playerID\": \"[a-f0-9]{8}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{12}\", \"roomCode\": \"[A-Z]{4}\", \"action\": \"drawHand\"\\}", netData); ok {
		re := regexp.MustCompile("\\{\"playerID\": \"([a-f0-9]{8}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{12})\", \"roomCode\": \"([A-Z]{4})\", \"action\": \"drawHand\"\\}")
		receivedID := re.FindStringSubmatch(netData)[1]
		roomCode := re.FindStringSubmatch(netData)[2]
		room := rooms[roomCode]
		player := room.Players[receivedID]

		for {
			if player.PlayerNumber == room.DrawTurn {
				conn.Write([]byte(fmt.Sprintf("{\"cards\": \"%s\"}\n", strings.Join(room.Deck.drawHand(), ","))))
				room.DrawTurn += 1
				return
			}
		}
	} else if ok, _ := regexp.MatchString("\\{\"playerID\": \"[a-f0-9]{8}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{12}\", \"roomCode\": \"[A-Z]{4}\", \"playCard\": \"(?:draw|(?:[AKQJ2-9]|10)[SCHD])\"\\}", netData); ok {
		re := regexp.MustCompile("\\{\"playerID\": \"([a-f0-9]{8}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{12})\", \"roomCode\": \"([A-Z]{4})\", \"playCard\": \"(draw|(?:[AKQJ2-9]|10)[SCHD])\"\\}")
		receivedID := re.FindStringSubmatch(netData)[1]
		roomCode := re.FindStringSubmatch(netData)[2]
		room := rooms[roomCode]
		playedCard := re.FindStringSubmatch(netData)[3]
		player := room.Players[receivedID]

		cond.L.Lock()
		if player.PlayerNumber == room.PlayerTurn {
			if playedCard == "draw" {
				player.Hand.drawCard(&room.Deck)
				cardValues := []string{} // Need to turn this into a Hand method, this is reused a lot
				for _, card := range player.Hand.Cards {
					cardValues = append(cardValues, card.Value)
				}
				room.Players[player.PlayerID] = player

				conn.Write([]byte(fmt.Sprintf("{\"rulesPassed\": true, \"currentHand\": \"%s\", \"wonGame\": false}\n", strings.Join(cardValues, ", "))))
				room.PlayerTurn = (room.PlayerTurn + 1) % len(room.Players)
			} else if rulesCheck(player, playedCard, *room) { // check against the rules. Need to check that card is in hand and that it passes the rules. Presumably updates the hand before response is passed
				// Pop card
				var popIndex int
				for i, card := range player.Hand.Cards {
					if card.Value == playedCard {
						popIndex = i
					}
				}
				player.Hand.printHand()
				room.Deck.DiscardPile = append(room.Deck.DiscardPile, player.Hand.Cards[popIndex])
				appended := append(player.Hand.Cards[:popIndex], player.Hand.Cards[popIndex+1:]...)
				fmt.Println(appended)
				player.Hand.Cards = appended
				room.Players[player.PlayerID] = player
				player.Hand.printHand()

				// Convert cards to string list
				cardValues := []string{}
				for _, card := range player.Hand.Cards {
					cardValues = append(cardValues, card.Value)
				}
				player.Hand.printHand()

				if len(player.Hand.Cards) > 0 {
					conn.Write([]byte(fmt.Sprintf("{\"rulesPassed\": true, \"currentHand\": \"%s\", \"wonGame\": false}\n", strings.Join(cardValues, ", "))))
				} else {
					conn.Write([]byte(fmt.Sprintf("{\"rulesPassed\": true, \"currentHand\": \"%s\", \"wonGame\": true}\n", strings.Join(cardValues, ", "))))
				}
				room.PlayerTurn = (room.PlayerTurn + 1) % len(room.Players)
			} else {
				player.Hand.drawCard(&room.Deck)
				cardValues := []string{} // Need to turn this into a Hand method, this is reused a lot
				for _, card := range player.Hand.Cards {
					cardValues = append(cardValues, card.Value)
				}
				room.Players[player.PlayerID] = player

				conn.Write([]byte(fmt.Sprintf("{\"rulesPassed\": false, \"currentHand\": \"%s\", \"wonGame\": false}\n", strings.Join(cardValues, ", "))))
			}
		}
		cond.Signal()
		cond.L.Unlock()
		return
	} else if ok, _ := regexp.MatchString("{\"roomCode\": \"[A-Z]{4}\", \"action\": \"requestTurn\"}", netData); ok {
		re := regexp.MustCompile("{\"roomCode\": \"([A-Z]{4})\", \"action\": \"requestTurn\"}")
		roomCode := re.FindStringSubmatch(netData)[1]
		room := rooms[roomCode]
		conn.Write([]byte(fmt.Sprintf("{\"playerNumber\": %d, \"topCard\": \"%s\"}\n", room.PlayerTurn, room.Deck.getCurrentTop().Value)))
		return
	} else if ok, _ := regexp.MatchString("{\"roomCode\": \"[A-Z]{4}\", \"action\": \"waitTurn\"}", netData); ok {
		re := regexp.MustCompile("{\"roomCode\": \"([A-Z]{4})\", \"action\": \"waitTurn\"}")
		roomCode := re.FindStringSubmatch(netData)[1]
		room := rooms[roomCode]
		currentPlayer := room.PlayerTurn
		cond.L.Lock() // Should probably convert other waiting periods to this as well, like waiting on game start
		for currentPlayer == room.PlayerTurn {
			fmt.Println("waiting...")
			cond.Wait()
		}

		for _, player := range room.Players {
			if currentPlayer == player.PlayerNumber {
				if len(player.Hand.Cards) == 0 {
					conn.Write([]byte(fmt.Sprintf("{\"wonGame\": true, \"winningPlayer\": \"%d\"}\n", currentPlayer+1)))
				}
				break
			}
		}

		cond.Signal()
		cond.L.Unlock()
		conn.Write([]byte("{\"wonGame\": false}\n"))
		return
	}

	fmt.Printf("Received message: %s\n", netData)
	conn.Write([]byte(fmt.Sprintf("Message received: %s\n", netData)))
}
