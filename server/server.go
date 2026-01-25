package main

import (
	"bufio"
	"fmt"
	"math/rand/v2"
	"net"
	"regexp"
	"strconv"
	"strings"

	"github.com/google/uuid"
)

type Player struct {
	id           string
	canStartGame bool
}

type Room struct {
	players   map[string]Player
	isStarted bool
}

type Card struct {
	Rank  string
	Suit  string
	Value string
}

type Deck struct {
	Pile        []*Card
	DiscardPile []*Card
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

func main() {
	// Create deck
	deck := initializeDeck()
	room := Room{isStarted: false, players: make(map[string]Player)}

	l, err := net.Listen("tcp", ":9090")
	if err != nil {
		panic(err)
	}
	defer l.Close()
	fmt.Println("Listening on all interfaces on port 9090")

	deck.printDeck()
	for {
		conn, err := l.Accept()
		if err != nil {
			panic(err)
		}

		go handleConnection(conn, &deck, &room)
	}
}

func handleConnection(conn net.Conn, deck *Deck, room *Room) {
	defer conn.Close()

	netData, err := bufio.NewReader(conn).ReadString('\n')
	if err != nil {
		panic(err)
	}

	netData = strings.TrimSpace(netData)
	if netData == "{'action': 'drawHand'}" {
		conn.Write([]byte(strings.Join(deck.drawHand(), ",")))
		deck.printDeck()
		return
	} else if netData == "{'joinGame': 'ABCD'}" {
		// Need to do a check if the game state has started or not. If yes, give player UUID and stuff. If no, return a can't join error
		// - OR Have a client send a create room message? And then that player controls the room?
		// Also need to check if any players have joined. If no, canStartGame is returned true. Else, false
		newPlayer := Player{id: uuid.NewString()}

		if len(room.players) == 0 {
			newPlayer.canStartGame = true
		} else {
			newPlayer.canStartGame = false
		}

		room.players[newPlayer.id] = newPlayer
		conn.Write([]byte(fmt.Sprintf("{'joined': 'true', 'playerID': '%s', 'canStartGame', '%s'}\n", newPlayer.id, strconv.FormatBool(newPlayer.canStartGame))))
		return
	} else if netData == "{'waiting': 'true'}" {
		for {
			if room.isStarted {
				conn.Write([]byte("{'gameStarted': 'true'}\n"))
				return
			}
		}
	} else if ok, _ := regexp.MatchString("\\{'playerID': '[a-z0-9]{8}-[a-z0-9]{4}-[a-z0-9]{4}-[a-z0-9]{4}-[a-z0-9]{12}', 'startGame': 'true'\\}", netData); ok {
		re := regexp.MustCompile("\\{'playerID': '([a-f0-9]{8}-[a-z0-9]{4}-[a-z0-9]{4}-[a-z0-9]{4}-[a-z0-9]{12})', 'startGame': 'true'\\}")
		receivedID := re.FindStringSubmatch(netData)[1]

		// Checks to see if a real player's UUID was received
		if _, ok := room.players[receivedID]; !ok {
			conn.Write([]byte("{'gameStarted': 'false', 'message': 'Did not receive a valid UUID'}\n"))
			return
		}

		if room.players[receivedID].canStartGame {
			room.isStarted = true
			conn.Write([]byte("{'gameStarted': 'true'}\n"))
			return
		} else {
			conn.Write([]byte("{'gameStarted': 'false', 'message': 'Only first player can start the game'}\n"))
			return
		}
	}

	fmt.Printf("Received message: %s\n", netData)
	conn.Write([]byte(fmt.Sprintf("Message received: %s\n", netData)))
}
