package main

import (
	"fmt"
	"math/rand/v2"
	"regexp"
	"slices"
)

type Rules func(card *Card, deck *Deck) string

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

type Hand struct {
	Cards []*Card
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

func (hand *Hand) drawHand(deck *Deck) {
	hand.Cards = deck.Pile[:7]
	deck.Pile = deck.Pile[7:]
}

func (hand *Hand) playCard(handPosition int, deck *Deck) {
	deck.DiscardPile = append(deck.DiscardPile, hand.Cards[handPosition])
	appended := append(hand.Cards[:handPosition], hand.Cards[handPosition+1:]...)
	hand.Cards = appended
}

func (hand *Hand) playTurn(deck *Deck, rules []Rules) {
	// deck.printPile("discardPile")
	deck.printDeck()
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
				break
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
				failedRule := false
				for _, rule := range rules {
					ruleResult := rule(playedCard, deck)

					if ruleResult == "true" {
						break
					} else if ruleResult == "false" {
						failedRule = true
						break
					} else if ruleResult == "continue" {
						continue
					} else {
						continue
					}
				}

				if !failedRule {
					hand.playCard(handPosition, deck)
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

// The base "UNO" rule
func baseRule(card *Card, deck *Deck) string {
	currentCard := deck.getCurrentTop()

	if card.Rank == currentCard.Rank {
		return "true"
	} else if card.Suit == currentCard.Suit {
		return "true"
	} else {
		return "false"
	}
}

// Test rule: You can play any face card with no penalty
func testRule(card *Card, deck *Deck) string {
	if slices.Contains([]string{"K", "Q", "J"}, card.Rank) {
		return "true"
	} else {
		return "continue"
	}
}

// Test rule 2: takes precedent over all face cards work rule in that the Jack of Spades is an auto penalty
func testRule2(card *Card, deck *Deck) string {
	if card.Value == "JS" {
		return "false"
	} else {
		return "continue"
	}
}

func main() {
	// Initialize deck and hand
	deck := initializeDeck()
	hand := Hand{}
	hand.drawHand(&deck)
	hand2 := Hand{}
	hand2.drawHand(&deck)

	// Base Rules array:
	// Have to do with rules regarding how played cards interact with the deck
	// Rules should be appended at index 0 so they run first in the loop.
	// New rules take priority over later rules.
	// Rules should be able to return three states: true, continue, and false
	//  - if true, auto succeed and skip remaining rules in the stack
	//  - if false, auto fail and skip remaining rules in the stack
	//  - if conitnue, move to next rule in stack
	rules := []Rules{testRule2, testRule, baseRule}

	// Need to figure out how to implement rules of other categories:
	// 1) Rules around user input (adjust regex. i.e. can say "autoplay" once and play any card. would need a base rule generated too to compensate)
	//  - would also have to somehow update hand state to track if autoplay has been used
	// 2) Rules that allow multiple cards to be played (i.e. can play multiple cards if they have the same rank.)
	// 3 ) Rules that change the win condition (i.e. anyone who plays the ace of spades wins)

	for {
		hand.playTurn(&deck, rules)
		if len(hand.Cards) == 0 {
			fmt.Println("Player 1 has won!")
			break
		}

		hand2.playTurn(&deck, rules)
		if len(hand2.Cards) == 0 {
			fmt.Println("Player 2 has won!")
			break
		}
	}
}
