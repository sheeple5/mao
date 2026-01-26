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
	Hand         Hand
	PlayerID     string
	PlayerNumber int
	CanStartGame bool
}

type Room struct {
	Players    map[string]Player
	Deck       Deck
	IsStarted  bool
	DrawTurn   int
	PlayerTurn int
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

type Hand struct {
	Cards []*Card
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

func main() {
	// Create deck
	room := Room{Deck: initializeDeck(), IsStarted: false, Players: make(map[string]Player)}

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

		go handleConnection(conn, &room)
	}
}

func handleConnection(conn net.Conn, room *Room) {
	defer conn.Close()

	netData, err := bufio.NewReader(conn).ReadString('\n')
	if err != nil {
		panic(err)
	}

	netData = strings.TrimSpace(netData)
	if netData == "{\"joinGame\": \"ABCD\"}" {
		// Need to do a check if the game state has started or not. If yes, give player UUID and stuff. If no, return a can't join error
		// - OR Have a client send a create room message? And then that player controls the room?
		// Also need to check if any players have joined. If no, canStartGame is returned true. Else, false
		newPlayer := Player{PlayerID: uuid.NewString()}

		if len(room.Players) == 0 {
			newPlayer.PlayerNumber = 0
			newPlayer.CanStartGame = true
		} else {
			newPlayer.PlayerNumber = len(room.Players)
			newPlayer.CanStartGame = false
		}
		newPlayer.drawHand(room)

		room.Players[newPlayer.PlayerID] = newPlayer
		conn.Write([]byte(fmt.Sprintf("{\"playerID\": \"%s\", \"playerNumber\": %d, \"canStartGame\": %s}\n", newPlayer.PlayerID, newPlayer.PlayerNumber, strconv.FormatBool(newPlayer.CanStartGame))))
		return
	} else if ok, _ := regexp.MatchString("\\{\"playerID\": \"[a-z0-9]{8}-[a-z0-9]{4}-[a-z0-9]{4}-[a-z0-9]{4}-[a-z0-9]{12}\", \"action\": \"waitingStart\"\\}", netData); ok {
		re := regexp.MustCompile("\\{\"playerID\": \"([a-f0-9]{8}-[a-z0-9]{4}-[a-z0-9]{4}-[a-z0-9]{4}-[a-z0-9]{12})\", \"action\": \"waitingStart\"\\}")
		receivedID := re.FindStringSubmatch(netData)[1]
		for {
			if room.IsStarted {
				cardValues := []string{}
				for _, card := range room.Players[receivedID].Hand.Cards {
					cardValues = append(cardValues, card.Value)
				}
				conn.Write([]byte(fmt.Sprintf("{\"initialHand\": \"%s\"}\n", strings.Join(cardValues, ", "))))
				return
			}
		}
	} else if ok, _ := regexp.MatchString("\\{\"playerID\": \"[a-z0-9]{8}-[a-z0-9]{4}-[a-z0-9]{4}-[a-z0-9]{4}-[a-z0-9]{12}\", \"action\": \"startGame\"\\}", netData); ok {
		re := regexp.MustCompile("\\{\"playerID\": \"([a-f0-9]{8}-[a-z0-9]{4}-[a-z0-9]{4}-[a-z0-9]{4}-[a-z0-9]{12})\", \"action\": \"startGame\"\\}")
		receivedID := re.FindStringSubmatch(netData)[1]

		// Checks to see if a real player's UUID was received
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
			return
		} else {
			conn.Write([]byte("{\"gameStarted\": \"false\", \"message\": \"Only first player can start the game\"}\n"))
			return
		}
	} else if ok, _ := regexp.MatchString("\\{\"playerID\": \"[a-f0-9]{8}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{12}\", \"action\": \"drawHand\"\\}", netData); ok {
		re := regexp.MustCompile("\\{\"playerID\": \"([a-f0-9]{8}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{12})\", \"action\": \"drawHand\"\\}")
		receivedID := re.FindStringSubmatch(netData)[1]
		player := room.Players[receivedID]

		for {
			if player.PlayerNumber == room.DrawTurn {
				conn.Write([]byte(fmt.Sprintf("{\"cards\": \"%s\"}\n", strings.Join(room.Deck.drawHand(), ","))))
				room.DrawTurn += 1
				return
			}
		}
	} else if netData == "{\"action\": \"requestTurn\"}" {
		conn.Write([]byte(fmt.Sprintf("{\"playerNumber\": %d, \"topCard\": \"%s\"}\n", room.PlayerTurn, room.Deck.getCurrentTop().Value)))
		return
	}

	fmt.Printf("Received message: %s\n", netData)
	conn.Write([]byte(fmt.Sprintf("Message received: %s\n", netData)))
}
