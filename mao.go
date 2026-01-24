package main

import (
	"fmt"
	"math/rand/v2"
	"regexp"
)

type Card struct {
	Rank  string
	Suit  string
	Value string
}

type Deck struct {
	Pile        []*Card
	DiscardPile []*Card
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
	rand.Shuffle(len(pile), func(i, j int) {
		pile[i], pile[j] = pile[j], pile[i]
	})

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

type Hand struct {
	Cards []*Card
}

func (hand *Hand) drawCard(deck *Deck) {
	drawnCard := deck.Pile[0]
	hand.Cards = append(hand.Cards, drawnCard)
	deck.Pile = deck.Pile[1:]
}

func (hand *Hand) drawHand(deck *Deck) {
	hand.Cards = deck.Pile[:7]
	deck.Pile = deck.Pile[7:]
}

func (hand *Hand) playCard(handPosition int, card *Card, deck *Deck) {
	deck.DiscardPile = append(deck.DiscardPile, card)
	hand.Cards = append(hand.Cards[:handPosition], hand.Cards[handPosition+1:]...)
}

func (hand *Hand) playTurn(deck *Deck) {
	deck.printPile("discardPile")
	deck.printCurrentTop()
	hand.printHand()

	// Eventually, should modulate ALL of this.
	// - If a player wants to make a rule to play nothing, that should update the regex
	// - If a player wants to make a rule for how cards interact with the deck, should update card comparison rules

	// Player chooses a card from their hand or chooses to draw
	var choice string
	for {
		fmt.Print("Choose a card to play (or type 'draw' to draw a card): ")
		fmt.Scan(&choice)

		match, _ := regexp.MatchString("^(?:(?:[2-9JQKA]|10)[SCHD]|draw)$", choice)
		if !match {
			fmt.Println("Error, please play a card or choose to draw.")
		} else {
			// If player chose to draw, pulls a card from the top of the deck and ends turn
			if choice == "draw" {
				hand.drawCard(deck)
				return
			}

			// If player chose to play a card, validates that the card is actually in hand
			var playedCard *Card
			var handPosition int
			for i, card := range hand.Cards {
				if choice == card.Value {
					playedCard = card
					handPosition = i
					break
				}
			}

			if playedCard != nil {
				if baseRule(playedCard, deck) {
					hand.playCard(handPosition, playedCard, deck)
					break
				} else {
					fmt.Println("Broke a rule, incur a penalty.\n")
					hand.drawCard(deck)

					deck.printPile("discardPile")
					deck.printCurrentTop()
					hand.printHand()
				}
			} else {
				fmt.Println("Error, please play a card from your hand.")
			}
		}
	}
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

func baseRule(card *Card, deck *Deck) bool {
	currentCard := deck.getCurrentTop()

	if card.Rank == currentCard.Rank {
		return true
	} else if card.Suit == currentCard.Suit {
		return true
	} else {
		return false
	}
}

func main() {
	deck := initializeDeck()
	hand := Hand{}
	hand.drawHand(&deck)

	deck.printDeck()

	hand.playTurn(&deck)
	deck.printDeck()
	hand.printHand()
}
