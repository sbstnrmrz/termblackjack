package main

import (
    "fmt"
    "strconv"
	"github.com/rivo/tview"
	"github.com/gdamore/tcell/v2"
)

const (
    Spades  int = iota 
    Clubs 
    Diamonds
    Hearts

    // unicode card suits
    CharCardBack rune = '\u2591';
    CharSpade    rune = '\u2660';
    CharClub     rune = '\u2661';
    CharDiamond  rune = '\u2666';
    CharHeart    rune = '\u2663';

    CardW = 5
    CardH = 4
)

type Card struct {
    *tview.Box
    suit       int
    pipNumber  int 
    suitChar   rune
    revealed   bool
}

func (c *Card) Draw(screen tcell.Screen) {
    c.Box.DrawForSubclass(screen, c)
    x, y, w, h := c.GetInnerRect()
    
    if c.revealed {
        var pip string 
        switch c.pipNumber {
        case 1:
            pip = "A"
        case 11:
            pip = "J"
        case 12:
            pip = "K"
        case 13:
            pip = "Q"
        default:
            pip = strconv.Itoa(c.pipNumber)
        }
        tview.Print(screen, pip, x, y, w, tview.AlignLeft, tcell.ColorWhite)
        tview.Print(screen, string(c.suitChar), x, y+1, w, tview.AlignLeft, tcell.ColorWhite)
    } else {
        for i := 0; i < w; i++ {
            for j := 0; j < h; j++ {
                tview.Print(screen, string(CharCardBack), x+i, y+j, w, tview.AlignLeft, tcell.ColorWhite)
            }
        }
    }
}

func NewCard(suit int, pips int, revealed bool) Card {
    if suit < 0 || suit > 3 {
        fmt.Printf("suit number should be between 0 and 3\n")
    }

    var suitChar rune
    switch suit {
    case Spades:
        suitChar = CharSpade 
    case Clubs:
        suitChar = CharClub 
    case Diamonds:
        suitChar = CharDiamond
    case Hearts:
        suitChar = CharHeart
    }

    var c Card  
    c.Box = tview.NewBox().SetBorder(true)
    c.Box.SetRect(0, 0, 5, 4)
    c.suit = suit
    c.suitChar = suitChar
    c.pipNumber = pips
    c.revealed = revealed
    
    return c
}
