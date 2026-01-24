package main

import (
	"fmt"
	"math/rand/v2"
)

type Card struct {
	Value string
}

type Deck struct {
	Pile        []*Card
	DiscardPile []*Card
}

func (deck Deck) printDeck() {
	fmt.Println("Pile: ")
	fmt.Println("----------------------")
	for i, card := range deck.Pile {
		if i != len(deck.Pile)-1 {
			fmt.Printf("%s, ", card.Value)
		} else {
			fmt.Printf("%s\n\n", card.Value)
		}
	}

	fmt.Println("Discard Pile: ")
	fmt.Println("----------------------")
	for i, card := range deck.DiscardPile {
		if i != len(deck.DiscardPile)-1 {
			fmt.Printf("%s, ", card.Value)
		} else {
			fmt.Printf("%s\n\n", card.Value)
		}
	}
}

type Hand struct {
	Cards []*Card
}

func (hand *Hand) drawHand(deck *Deck) {
	hand.Cards = deck.Pile[:7]
	deck.Pile = deck.Pile[7:]
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

func initializeDeck() Deck {
	// Load up card pile
	pile := []*Card{}
	for _, suit := range []string{"S", "C", "H", "D"} {
		for _, rank := range []string{"A", "K", "Q", "J", "10", "9", "8", "7", "6", "5", "4", "3", "2"} {
			newCard := Card{rank + suit}
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

func main() {
	deck := initializeDeck()
	deck.printDeck()

	hand := Hand{}
	hand.drawHand(&deck)
}
