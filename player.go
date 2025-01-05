package main

import (
	"math/rand"
)

const (
    StateNone       int = 0 
    StatePlaying    int = 1 
    StateBetting    int = 2
    PlayerStand      int = 4
    PlayerDoubleDown int = 8
)

type Player struct {
    cards   []Card
    score   int
    bet     int
    chips   int
    state   int
}

func NewPlayer() Player {
    return Player{
        cards: []Card{},
        chips: 0,
        bet:   0,
        score: 0,
        state: 0,
    }
}

func (p *Player) CalcScore() {
    p.score = 0
    for i := 0; i < len(p.cards); i++ {
        if p.cards[i].revealed {
            pipNum := p.cards[i].pipNumber
            if pipNum >= 10 {
                p.score += 10 
            } else if pipNum == 1 && p.score < 11 {
                p.score += 11 
            } else {
                p.score += pipNum
            }
        }
    }
}

func (p *Player) AddChips(chips int) {
    p.chips += chips
}

func (p *Player) IncBet(bet int) {
    p.bet += bet 
}

func (p *Player) RemoveChips(chips int) {
    p.chips -= chips
}

func (p *Player) HitCard(revealed bool) {
    card := NewCard(rand.Intn(4), rand.Intn(13)+1, revealed)
    p.cards = append(p.cards, card)

//  if card.pipNumber >= 10 {
//      p.score += 10 
//  } else if card.pipNumber == 1 && p.score < 11 {
//      p.score += 11 
//  } else {
//      p.score += card.pipNumber
//  }
}

func (p *Player) Stand() {
    p.state |= PlayerStand
}

func (p *Player) DoubleDown() {
    p.state |= PlayerDoubleDown 
    p.chips -= p.bet 
    p.bet *= 2
    p.HitCard(true)
}
