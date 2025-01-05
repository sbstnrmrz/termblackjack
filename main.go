package main

import (
    "bytes"
    "encoding/gob"
	"fmt"
	"math/rand"
	"net"
	"os"
	"strconv"
	"time"
    "bufio"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

const (
    ScreenMainMenu         int = iota
    ScreenSoloGameMenu
    ScreenSoloGame
    ScreenMultiplayerMenu 
    ScreenCreateServerMenu
    ScreenJoinServerMenu
)

const (


    ButtonHit int = iota
    ButtonDouble
    ButtonStand
    ButtonNextRound


    ButtonSoloGame = 0
    ButtonMultiplayerGame = 1
    ButtonExitGame = 2
)

func RandomString() string {
    var str string
    for i := 0; i < 4; i++ {
        str += string(rand.Intn(25)+97)
    }

    return str
}

type MyButton struct {
    *tview.TextView
    pressFunc func()
    selected bool
    visible bool
}

func NewMyButton(text string, pressFunc func()) *MyButton {
    return &MyButton{
        TextView: tview.NewTextView().SetText(text),
        pressFunc: pressFunc,
        selected: false,
        visible: true,
    }
}

func ClearSelected(buttons []*MyButton) {
    for _, button := range buttons {
        button.selected = false
    }
}

type App struct {
    *tview.Box
    app          *tview.Application
    game         Game
}

type Game struct {
    *tview.Box
    app          *tview.Application
    servers      []Server
    player       Player
    dealer       Player
    drawPile     Card
    gameOver     bool
    gameStarted  bool
    roundOver    bool
    roundStarted bool
    roundResult int
    broadcastAddr *net.UDPAddr
    ui struct {
        screen int
        betOpts       []*MyButton
        currBetOpt    int
        txt struct {
            playerChips *tview.TextView
            playerScore *tview.TextView
            playerBet   *tview.TextView
            dealerScore *tview.TextView
            roundResult *tview.TextView
        }
        soloGameMenu struct {
            startGame *MyButton
        }
        mainMenu struct {
            buttons    []*MyButton
            currButton int
        }
        soloGame struct {
            playerOpts    []*MyButton
            currPlayerOpt int
            startRound    *MyButton
            nextRound     *MyButton
        }
        mpMenu struct {
            buttons    []*MyButton
            currButton int
        }
        createServerMenu struct {
            buttons    []*MyButton
            currButton int
        }
        joinServerMenu struct {
            serverListBox  *tview.Box
            serverType     []*MyButton
            currServerType int 
            buttons        []*MyButton
            currButton     int
            serverList     *tview.Table
        }
    }
    mouse struct {
        x int
        y int 
    }
}

func NewGame() *Game {
    g := &Game{}
    g.Box = tview.NewBox().SetTitle("Game").SetBorder(false)

    g.player = NewPlayer()
    g.dealer = NewPlayer()

    g.ui.mainMenu.buttons = append(g.ui.mainMenu.buttons, NewMyButton("Solo game", g.SoloGameMenu))
    g.ui.mainMenu.buttons = append(g.ui.mainMenu.buttons, NewMyButton("Multiplayer game", g.MultiplayerMenu))
    g.ui.mainMenu.buttons = append(g.ui.mainMenu.buttons, NewMyButton("Exit game", g.app.Stop))
    g.ui.mainMenu.currButton = -1

    g.ui.txt.playerChips = tview.NewTextView()
    g.ui.txt.playerBet = tview.NewTextView()
    g.ui.txt.playerScore = tview.NewTextView()
    g.ui.txt.dealerScore = tview.NewTextView()
    g.ui.txt.roundResult = tview.NewTextView()

    g.ui.soloGameMenu.startGame = NewMyButton("Start game", g.StartSoloGame) 
    // here
    g.ui.soloGame.startRound = NewMyButton("Start round", g.StartRound)
    g.ui.betOpts = append(g.ui.betOpts, NewMyButton("5", nil)) 
    g.ui.betOpts = append(g.ui.betOpts, NewMyButton("25", nil))
    g.ui.betOpts = append(g.ui.betOpts, NewMyButton("50", nil))
    g.ui.betOpts = append(g.ui.betOpts, NewMyButton("100", nil))
    g.ui.betOpts = append(g.ui.betOpts, NewMyButton("500", nil))
    g.ui.betOpts = append(g.ui.betOpts, NewMyButton("1000", nil))
    g.ui.currBetOpt = -1

    g.ui.soloGame.playerOpts = append(g.ui.soloGame.playerOpts, NewMyButton("Hit", func () {g.player.HitCard(true)})) 
    g.ui.soloGame.playerOpts = append(g.ui.soloGame.playerOpts, NewMyButton("Double", g.player.DoubleDown))
    g.ui.soloGame.playerOpts = append(g.ui.soloGame.playerOpts, NewMyButton("Stand", g.player.Stand))
    g.ui.soloGame.nextRound = NewMyButton("Next round", g.NextRound)

//  for _, button := range g.ui.betOpts {
//      button.pressFunc = func() {
//          c, err := strconv.Atoi(button.GetText(true))
//          if err != nil {
//              panic(err)
//          }
//          g.player.AddChips(c)
//      }
//  } 

    g.ui.mpMenu.buttons = append(g.ui.mpMenu.buttons, NewMyButton("Host and play", nil))
    g.ui.mpMenu.buttons = append(g.ui.mpMenu.buttons, NewMyButton("Join server", g.JoinServerMenu))
    g.ui.mpMenu.buttons = append(g.ui.mpMenu.buttons, NewMyButton("Back to menu", g.MainMenu))

    // join server menu
    g.ui.joinServerMenu.serverType = append(g.ui.joinServerMenu.serverType, NewMyButton("Internet", func(){}))
    g.ui.joinServerMenu.serverType = append(g.ui.joinServerMenu.serverType, NewMyButton("LAN", func(){}))

    g.ui.joinServerMenu.buttons = append(g.ui.joinServerMenu.buttons, NewMyButton("Join", g.MultiplayerMenu))
    g.ui.joinServerMenu.buttons = append(g.ui.joinServerMenu.buttons, NewMyButton("Refresh list", g.RefreshServerList))
    g.ui.joinServerMenu.buttons = append(g.ui.joinServerMenu.buttons, NewMyButton("Back", g.MultiplayerMenu))
    g.ui.joinServerMenu.serverList = tview.NewTable().
        SetFixed(1, 1).
        SetSelectable(true, false).
        SetBorders(false).
        SetSeparator(tview.Borders.Vertical)

    g.ui.joinServerMenu.serverListBox = tview.NewBox()

    for i := 0; i < len(separators); i++ {
        cell := tview.NewTableCell(separators[i]).SetSelectable(false).SetAlign(tview.AlignLeft)
        g.ui.joinServerMenu.serverList.SetCell(0, i, cell)
    }


    g.app = tview.NewApplication()

    go func() {
        dur := time.Millisecond * 16
        for range time.Tick(dur) {
            g.app.QueueUpdateDraw(func() {

            })
        }
    }()

    return g
}

func (g *Game) MainMenu() {
    g.ui.mpMenu.currButton = 0
    g.ui.screen = ScreenMainMenu
}

func (g *Game) SoloGameMenu() {
    g.ui.currBetOpt = 0
    g.ui.screen = ScreenSoloGameMenu
    for _, button := range g.ui.betOpts {
        button.pressFunc = func() {
            c, err := strconv.Atoi(button.GetText(true))
            if err != nil {
                panic(err)
            }
            g.player.AddChips(c)
        }
    } 
}

func (g *Game) NextRound() {
    g.player.cards = []Card{}
    g.player.bet = 0
    g.player.score = 0
    g.player.state = 0

    g.dealer.cards = []Card{}
    g.dealer.score = 0
    
    for i := 0; i < 2; i++ {
        g.player.HitCard(true)
    }

    for i := 0; i < 1; i++ {
        g.dealer.HitCard(true)
    }
    g.dealer.HitCard(false)

    g.ui.currBetOpt = -1
    g.ui.soloGame.currPlayerOpt = -1
    g.roundStarted = false
    g.roundOver = false;
}

func (g *Game) RefreshServerList() {
    g.servers = []Server{}
    buf := make([]byte, 1024)

    go func() {
        broadcast, err := net.DialUDP("udp", nil, g.broadcastAddr)
        if err != nil {
            panic(err)
        }
        defer broadcast.Close()

        addr, err := net.ResolveUDPAddr("udp", GetNetworkIP(false) + ":" + ServerPort)
        if err != nil {
            panic(err)
        }

        conn, err := net.ListenUDP("udp", addr)
        if err != nil {
            panic(err)
        }
        defer conn.Close()

//      for {
            conn.SetReadDeadline(time.Now().Add(5000 * time.Millisecond))
            t1 := time.Now()
            _, err = broadcast.Write([]byte("hello"))
            if err != nil {
                panic(err)
            }

            _, remoteAddr, err := conn.ReadFromUDP(buf)
            if err != nil {
                opErr, ok := err.(*net.OpError)
                if ok && opErr.Timeout() {
                    fmt.Printf("se acabo el tiempo de espera\n")
//                  break
                }
                panic(err)
            }

            t2 := time.Now().Sub(t1).Nanoseconds()

            packet := bytes.NewBuffer(buf)  
            dec := gob.NewDecoder(packet)

            serverInfo := &PacketServerInfo{}

            err = dec.Decode(serverInfo)
            if err != nil {
                panic(err)
            }

            g.servers = append(g.servers, Server{
                Addr: remoteAddr.String(),
                Id: serverInfo.Id,   
                PlayerCount: serverInfo.PlayerCount,
                MaxPlayers: serverInfo.MaxPlayers,
                Latency: t2,
            })

//          fmt.Printf("server found:\n")
//          fmt.Printf("  address: %s\n", remoteAddr.String())
//          fmt.Printf("  ping: %dms\n", t2)
//          fmt.Printf("  max players: %d\n", serverInfo.MaxPlayers)
//          fmt.Printf("  players: %d\n\n", serverInfo.PlayerCount)
//      }
    }()


}

func (g *Game) StartRound() {
    g.ui.currBetOpt = -1
    g.ui.soloGame.currPlayerOpt = -1
    g.roundStarted = true

    for _, button := range g.ui.soloGame.playerOpts {
        button.visible = true
    }
}

func (g *Game) StartSoloGame() {
    for _, button := range g.ui.betOpts {
        button.pressFunc = func() {
            c, err := strconv.Atoi(button.GetText(true))
            if err != nil {
                panic(err)
            }
            if (g.player.chips - c >= 0) {
                g.player.IncBet(c)
                g.player.RemoveChips(c)
            }
        }
    } 
    g.ui.screen = ScreenSoloGame
    g.ui.currBetOpt = -1
    g.ui.soloGame.currPlayerOpt = -1
    g.NextRound();
    g.gameStarted = true
}

func (g *Game) EndRound() {
    for _, button := range g.ui.soloGame.playerOpts {
        button.visible = false
    }

    g.dealer.cards[len(g.dealer.cards)-1].revealed = true
    g.dealer.CalcScore()
    for g.dealer.score < 17 && g.player.score < 22 {
        g.dealer.HitCard(true) 
        g.dealer.CalcScore()
    }
    g.roundOver = true;

    g.CheckWinner()
}

func (g *Game) CheckWinner() {
    if g.player.score > 21 || (g.player.score < g.dealer.score && g.dealer.score <= 21) {
        g.roundResult = 1
    } else if g.player.score == g.dealer.score {
        g.roundResult = 2
        g.player.chips += g.player.bet
    } else {
        g.roundResult = 0
        g.player.chips += g.player.bet * 2
    }
}


func (g *Game) MultiplayerMenu() {
    g.ui.mainMenu.currButton = 0
    g.ui.screen = ScreenMultiplayerMenu
}

func (g *Game) JoinServerMenu() {
    g.ui.mpMenu.currButton = 0
    g.ui.screen = ScreenJoinServerMenu

    addr, err := net.ResolveUDPAddr("udp", GetIPBroadcastAddr(GetNetworkIP(true)) + ":" + ServerPort)
    if err != nil {
        panic(err)
    }

    g.broadcastAddr = addr
}

func (g *Game) Draw(screen tcell.Screen) {
    g.Box.DrawForSubclass(screen, g)
    _, _, winW, winH := g.Box.GetRect()
    playerOptW := 6+2

    playerCardBoundsX := (winW - CardW * len(g.player.cards))/2
    playerCardBoundsY :=  winH/2
    dealerCardBoundsX := (winW - CardW * len(g.dealer.cards))/2
    dealerCardBoundsY := winH/4

    switch g.ui.screen {
    case ScreenMainMenu:
        for i, button := range g.ui.mainMenu.buttons {
            buttonW := 16+2
            buttonH := 1
            buttonX := (winW - buttonW)/2  
            buttonY := (winH - buttonH)/2 + (buttonH+1)*i

            button.SetTextAlign(1)
            if i == g.ui.mainMenu.currButton {
                button.SetBackgroundColor(tcell.ColorBlue)
            }
            button.SetRect(buttonX, buttonY, buttonW, buttonH)
            button.Draw(screen)
            button.SetBackgroundColor(tcell.ColorDarkBlue)
        }
    case ScreenSoloGameMenu:
        for i, button := range g.ui.betOpts {
            buttonW := 4+2
            buttonH := 1
            buttonX := (winW - len(g.ui.betOpts) * buttonW)/2 + (buttonW+1) * i  
            buttonY := int(float32(winH)/float32(1.2))

            button.SetTextAlign(1)
            if i == g.ui.currBetOpt {
                button.SetBackgroundColor(tcell.ColorBlue)
            }
            button.SetRect(buttonX, buttonY, buttonW, buttonH)
            button.Draw(screen)
            button.SetBackgroundColor(tcell.ColorDarkBlue)
        }

        button := g.ui.soloGameMenu.startGame
        buttonW := len(g.ui.soloGameMenu.startGame.GetText(true))+2
        buttonH := 1
        buttonX := (winW - buttonW)/2
        _, buttonY, _, _ := g.ui.betOpts[0].GetRect()
        button.SetRect(buttonX, buttonY+2, buttonW, buttonH)
        if g.ui.currBetOpt == 10 {
            button.SetBackgroundColor(tcell.ColorBlue)
        }
        button.SetTextAlign(1)
        button.Draw(screen)
        button.SetBackgroundColor(tcell.ColorDarkBlue)

        g.ui.txt.playerChips.SetText(fmt.Sprintf("Total chips: %d", g.player.chips))
        g.ui.txt.playerChips.SetRect((winW - len(g.ui.txt.playerChips.GetText(true)))/2, 
                                     playerCardBoundsY+6, 
                                     len(g.ui.txt.playerChips.GetText(true)), 
                                     1)
        g.ui.txt.playerChips.Draw(screen)        
    case ScreenSoloGame:
        if !g.roundStarted {
            for i, button := range g.ui.betOpts {
                buttonW := 4+2
                buttonH := 1
                buttonX := (winW - len(g.ui.betOpts) * buttonW)/2 + (buttonW+1) * i  
                buttonY := int(float32(winH)/float32(1.2))

                button.SetTextAlign(1)
                if i == g.ui.currBetOpt {
                    button.SetBackgroundColor(tcell.ColorBlue)
                }
                button.SetRect(buttonX, buttonY, buttonW, buttonH)
                button.Draw(screen)
                button.SetBackgroundColor(tcell.ColorDarkBlue)
            }
            button := g.ui.soloGame.startRound
            buttonW := len(g.ui.soloGame.startRound.GetText(true))+2
            buttonH := 1
            buttonX := (winW - buttonW)/2
            _, buttonY, _, _ := g.ui.betOpts[0].GetRect()
            button.SetRect(buttonX, buttonY+2, buttonW, buttonH)
            if g.ui.currBetOpt == 10 {
                button.SetBackgroundColor(tcell.ColorBlue)
            }
            button.SetTextAlign(1)
            button.Draw(screen)
            button.SetBackgroundColor(tcell.ColorDarkBlue)
        } else {
            // draws player options
            for i, button := range g.ui.soloGame.playerOpts {
                playerOptX := (winW - playerOptW * len(g.ui.soloGame.playerOpts))/2 + i * (playerOptW+1)
                playerOptY := int(float32(winH)/float32(1.2))
                button.SetDisabled(false)
                button.SetRect(playerOptX, playerOptY, playerOptW, 1)

                button.SetBackgroundColor(tcell.ColorDarkBlue)
                if i == g.ui.soloGame.currPlayerOpt {
                    g.ui.soloGame.playerOpts[i].SetBackgroundColor(tcell.ColorBlue)
                } 
                button.SetTextAlign(1)
                if button.visible {
                    button.Draw(screen)
                }
            }
            
            // draws player cards and calculates score
            for i, card := range g.player.cards {
                cardX := playerCardBoundsX + ((CardW) * i) 
                cardY := playerCardBoundsY 
                card.SetRect(cardX, cardY, 5, 4)
                card.Draw(screen)
            }

            // draws dealer cards
            g.dealer.score = 0
            for i, card := range g.dealer.cards {
                cardX := dealerCardBoundsX + ((CardW) * i)
                cardY := dealerCardBoundsY
                card.SetRect(cardX, cardY, 5, 4)
                card.Draw(screen)
            }
            g.player.CalcScore()
            g.dealer.CalcScore()

            g.ui.txt.playerScore.SetText(fmt.Sprintf("Player score: %d", g.player.score))
            playerScoreW := len(g.ui.txt.playerScore.GetText(true))
            g.ui.txt.playerScore.SetRect((winW - playerScoreW)/2, 
                                         playerCardBoundsY+4, 
                                         playerScoreW, 
                                         1)
            g.ui.txt.playerScore.Draw(screen)

            g.ui.txt.dealerScore.SetText(fmt.Sprintf("Dealer score: %d", g.dealer.score))
            dealerScoreW := len(g.ui.txt.dealerScore.GetText(true))
            g.ui.txt.dealerScore.SetRect((winW - dealerScoreW)/2, 
                                         dealerCardBoundsY+4, 
                                         dealerScoreW,
                                         1)
            g.ui.txt.dealerScore.Draw(screen)
        }

        g.ui.txt.playerChips.SetText(fmt.Sprintf("Total chips: %d", g.player.chips))
        g.ui.txt.playerChips.SetRect((winW - len(g.ui.txt.playerChips.GetText(true)))/2, 
                                     playerCardBoundsY+6, 
                                     len(g.ui.txt.playerChips.GetText(true)), 
                                     1)
        g.ui.txt.playerChips.Draw(screen)        

        g.ui.txt.playerBet.SetText(fmt.Sprintf("Current bet: %d", g.player.bet))
        g.ui.txt.playerBet.SetRect((winW - len(g.ui.txt.playerBet.GetText(true)))/2, 
                                   playerCardBoundsY+5, 
                                   len(g.ui.txt.playerBet.GetText(true)), 
                                   1)
        g.ui.txt.playerBet.Draw(screen)

        if g.roundOver {
            for _, button := range g.ui.soloGame.playerOpts {
                button.visible = false
            }

            switch g.roundResult {
            case 0:
                g.ui.txt.roundResult.SetText("Player wins!")
            case 1:
                g.ui.txt.roundResult.SetText("Dealer wins!")
            case 2:
                g.ui.txt.roundResult.SetText("Draw!")
            }

            roundResultW := len(g.ui.txt.roundResult.GetText(true))
            _, roundResultY, _, _  := g.ui.txt.playerChips.GetRect()
            g.ui.txt.roundResult.SetRect((winW - roundResultW)/2, 
                                         roundResultY + 2, 
                                         roundResultW, 
                                         1)
            g.ui.txt.roundResult.Draw(screen)

            g.ui.soloGame.nextRound.visible = true
            g.ui.soloGame.nextRound.SetTextAlign(1)
            g.ui.soloGame.nextRound.SetText("Next round")
            startButtonW := len(g.ui.soloGame.nextRound.GetText(true))+2
            g.ui.soloGame.nextRound.SetRect((winW - startButtonW)/2, 
                                            int(float32(winH)/float32(1.2)), 
                                            startButtonW, 
                                            1)
            g.ui.soloGame.nextRound.SetBackgroundColor(tcell.ColorDarkBlue)
            if g.ui.soloGame.currPlayerOpt == 10{
                g.ui.soloGame.nextRound.SetBackgroundColor(tcell.ColorBlue)
            } 
            g.ui.soloGame.nextRound.Draw(screen)
        } else if g.player.score > 21 || g.player.state == PlayerStand {
            g.EndRound()
        }
    case ScreenMultiplayerMenu:
        for i, button := range g.ui.mpMenu.buttons {
            buttonW := 13+2
            buttonH := 1
            buttonX := (winW - buttonW)/2  
            buttonY := (winH - buttonH)/2 + (buttonH+1)*i

            button.SetTextAlign(1)
            if i == g.ui.mpMenu.currButton {
                button.SetBackgroundColor(tcell.ColorBlue)
            }
            button.SetRect(buttonX, buttonY, buttonW, buttonH)
            button.Draw(screen)
            button.SetBackgroundColor(tcell.ColorDarkBlue)
        }
    case ScreenJoinServerMenu:
        _, _, _, listH := g.ui.joinServerMenu.serverList.GetRect()
        listW := 24 
        listX := (winW - listW)/2
        listY := (winH - listH)/2
        g.ui.joinServerMenu.serverList.SetRect(listX, listY, listW, listH)
        g.ui.joinServerMenu.serverList.Box.SetRect(listX-1, listY-1, listW+1, listH)
        g.ui.joinServerMenu.serverList.Box.SetBorder(true)
        g.ui.joinServerMenu.serverList.Draw(screen)

        for i, button := range g.ui.joinServerMenu.buttons {
            buttonW := 4+2
            buttonH := 1
            buttonX := (winW - buttonW)/2  
            buttonY := (int(float32(winH)/float32(1.3)) - buttonH/2) + (buttonH+1)*i

            button.SetTextAlign(1)
            if i == g.ui.joinServerMenu.currButton {
                button.SetBackgroundColor(tcell.ColorBlue)
            }
            button.SetRect(buttonX, buttonY, buttonW, buttonH)
            button.Draw(screen)
            button.SetBackgroundColor(tcell.ColorDarkBlue)
        }

        for i, button := range g.ui.joinServerMenu.serverType {
            buttonW := (listW+1)/2  
            buttonH := 1
            buttonX := listX + (buttonW*i) - 1
            buttonY := listY-1

            button.SetTextAlign(1)
            button.SetBackgroundColor(tcell.ColorDarkBlue)
            if i == g.ui.joinServerMenu.currServerType {
                button.SetBackgroundColor(tcell.ColorRoyalBlue)
            }
            if button.selected {
                button.SetBackgroundColor(tcell.ColorBlue)
            }
            button.SetRect(buttonX, buttonY, buttonW, buttonH)
            button.Draw(screen)
        }

        for i := 0; i < len(g.servers); i++ {
            for j, str := range GetServerInfoStr(g.servers[i]) {
                cell := tview.NewTableCell(str).SetSelectable(true)
                g.ui.joinServerMenu.serverList.SetCell(i+1, j, cell)
            }
        }

//      g.ui.joinServerMenu.currServerType = -1
    }
}

func (g *Game) InputHandler() func(event *tcell.EventKey, setFocus func(p tview.Primitive)) {
	return g.WrapInputHandler(func(event *tcell.EventKey, setFocus func(p tview.Primitive)) {
        switch g.ui.screen {
        case ScreenMainMenu:
            switch event.Key() {
            case tcell.KeyUp:
                g.ui.mainMenu.currButton--
                if g.ui.mainMenu.currButton < 0 {
                    g.ui.mainMenu.currButton =  len(g.ui.mainMenu.buttons)-1
                } 
            case tcell.KeyDown:
                g.ui.mainMenu.currButton++
                if g.ui.mainMenu.currButton > len(g.ui.mainMenu.buttons)-1 {
                    g.ui.mainMenu.currButton = 0 
                } 
            case tcell.KeyEnter:
                if g.ui.mainMenu.currButton > -1 {
                    if g.ui.mainMenu.buttons[g.ui.mainMenu.currButton].pressFunc != nil {
                        g.ui.mainMenu.buttons[g.ui.mainMenu.currButton].pressFunc()
                    }
                }
            }
        case ScreenSoloGameMenu:
            switch event.Key() {
            case tcell.KeyLeft:
                g.ui.currBetOpt--
                if g.ui.currBetOpt < 0 {
                    g.ui.currBetOpt =  len(g.ui.betOpts)-1
                } 
            case tcell.KeyRight:
                g.ui.currBetOpt++
                if g.ui.currBetOpt > len(g.ui.betOpts)-1 {
                    g.ui.currBetOpt = 0 
                } 
            case tcell.KeyUp:
                if g.ui.currBetOpt < 10 {
                    g.ui.currBetOpt = 10
                } else {
                    g.ui.currBetOpt = 0
                }
            case tcell.KeyDown:
                if g.ui.currBetOpt < 10 {
                    g.ui.currBetOpt = 10
                } else {
                    g.ui.currBetOpt = 0
                }
            case tcell.KeyEnter:
                if g.ui.currBetOpt == 10 {
                    g.ui.soloGameMenu.startGame.pressFunc()
                    return
                }
                if g.ui.betOpts[g.ui.currBetOpt].pressFunc != nil {
                    g.ui.betOpts[g.ui.currBetOpt].pressFunc()
                }
            }
        case ScreenSoloGame:
            if !g.gameStarted {
            
            }
            switch event.Key() {
            case tcell.KeyLeft:
                g.ui.currBetOpt--
                if g.ui.currBetOpt < 0 {
                    g.ui.currBetOpt =  len(g.ui.betOpts)-1
                } 
            case tcell.KeyRight:
                g.ui.currBetOpt++
                if g.ui.currBetOpt > len(g.ui.betOpts)-1 {
                    g.ui.currBetOpt = 0 
                } 
            case tcell.KeyUp:
                if g.ui.currBetOpt < 10 {
                    g.ui.currBetOpt = 10
                } else {
                    g.ui.currBetOpt = 0
                }
            case tcell.KeyDown:
                if g.ui.currBetOpt < 10 {
                    g.ui.currBetOpt = 10
                } else {
                    g.ui.currBetOpt = 0
                }
            case tcell.KeyEnter:
                if g.ui.currBetOpt == 10 {
                    g.ui.soloGame.startRound.pressFunc()
                    return
                }
                if g.ui.betOpts[g.ui.currBetOpt].pressFunc != nil {
                    g.ui.betOpts[g.ui.currBetOpt].pressFunc()
                }
            }

        case ScreenMultiplayerMenu:
            switch event.Key() {
            case tcell.KeyUp:
                g.ui.mpMenu.currButton--
                if g.ui.mpMenu.currButton < 0 {
                    g.ui.mpMenu.currButton =  len(g.ui.mpMenu.buttons)-1
                } 
            case tcell.KeyDown:
                g.ui.mpMenu.currButton++
                if g.ui.mpMenu.currButton > len(g.ui.mpMenu.buttons)-1 {
                    g.ui.mpMenu.currButton = 0 
                } 
            case tcell.KeyEnter:
                g.ui.mpMenu.buttons[g.ui.mpMenu.currButton].pressFunc()
            }
        case ScreenJoinServerMenu:
            if g.ui.joinServerMenu.serverList.HasFocus() {
                g.ui.joinServerMenu.serverList.InputHandler()(event, setFocus)
            }

            switch event.Key() {
            case tcell.KeyUp:
                g.ui.joinServerMenu.currButton--
                if g.ui.joinServerMenu.currButton < 0 {
                    g.ui.joinServerMenu.currButton =  len(g.ui.joinServerMenu.buttons)-1
                } 
            case tcell.KeyDown:
                g.ui.joinServerMenu.currButton++
                if g.ui.joinServerMenu.currButton > len(g.ui.joinServerMenu.buttons)-1 {
                    g.ui.joinServerMenu.currButton = 0 
                } 
            case tcell.KeyEnter:
                g.ui.joinServerMenu.buttons[g.ui.joinServerMenu.currButton].pressFunc()
            }

        }
	})
}

func (g *Game) MouseHandler() func(action tview.MouseAction, event *tcell.EventMouse, setFocus func(p tview.Primitive)) (consumed bool, capture tview.Primitive) {
	return g.WrapMouseHandler(func(action tview.MouseAction, event *tcell.EventMouse, setFocus func(p tview.Primitive)) (consumed bool, capture tview.Primitive) {
        g.mouse.x, g.mouse.y = event.Position()

        switch g.ui.screen {
        case ScreenMainMenu:
            for i, button := range g.ui.mainMenu.buttons {
                if button.InRect(g.mouse.x, g.mouse.y) {
                    g.ui.mainMenu.currButton = i
                    if action == tview.MouseLeftClick && button.pressFunc != nil {
                        button.pressFunc()
                    }
                    return
                }
            }
            g.ui.mainMenu.currButton = -1
        case ScreenSoloGameMenu:
            for i, button := range g.ui.betOpts {
                if button.InRect(g.mouse.x, g.mouse.y) {
                    g.ui.currBetOpt = i
                    if action == tview.MouseLeftClick && button.pressFunc != nil {
                        button.pressFunc()
                    }
                    return 
                }
            }
            if g.ui.soloGameMenu.startGame.InRect(g.mouse.x, g.mouse.y) {
                g.ui.currBetOpt = 10
                if action == tview.MouseLeftClick && g.ui.soloGameMenu.startGame.pressFunc != nil {
                    g.ui.soloGameMenu.startGame.pressFunc()
                }
                return 
            }
            g.ui.currBetOpt = -1
        case ScreenSoloGame:
            if !g.roundStarted {
                for i, button := range g.ui.betOpts {
                    if button.InRect(g.mouse.x, g.mouse.y) {
                        g.ui.currBetOpt = i
                        if action == tview.MouseLeftClick && button.pressFunc != nil {
                            button.pressFunc()
                        }
                        return 
                    }
                }
                if g.ui.soloGame.startRound.InRect(g.mouse.x, g.mouse.y) {
                    g.ui.currBetOpt = 10
                    if action == tview.MouseLeftClick && g.ui.soloGame.startRound.pressFunc != nil {
                        g.ui.soloGame.startRound.pressFunc()
                    }
                    return 
                }
                g.ui.currBetOpt = -1
                return
            } else {
                for i, button := range g.ui.soloGame.playerOpts {
                    if button.InRect(g.mouse.x, g.mouse.y) && button.visible {
                        g.ui.soloGame.currPlayerOpt = i
                        if action == tview.MouseLeftClick && button.pressFunc != nil {
                            button.pressFunc()
                        }
                        return 
                    }
                }

                button := g.ui.soloGame.nextRound
                if button.InRect(g.mouse.x, g.mouse.y) && button.visible {
                    g.ui.soloGame.currPlayerOpt = 10
                    if action == tview.MouseLeftClick && button.pressFunc != nil {
                        button.pressFunc()
                    }
                    return 
                }
                g.ui.soloGame.currPlayerOpt = -1
            }
        case ScreenMultiplayerMenu:
            for i, button := range g.ui.mpMenu.buttons {
                if button.InRect(g.mouse.x, g.mouse.y) {
                    g.ui.mpMenu.currButton = i
                    if action == tview.MouseLeftClick && button.pressFunc != nil {
                        button.pressFunc()
                    }
                }
            }
        case ScreenJoinServerMenu:
            for i, button := range g.ui.joinServerMenu.buttons {
                if button.InRect(g.mouse.x, g.mouse.y) {
                    g.ui.joinServerMenu.currButton = i
                    if action == tview.MouseLeftClick && button.pressFunc != nil {
                        button.pressFunc()
                    }
                }
            }
            for i, button := range g.ui.joinServerMenu.serverType {
                if button.InRect(g.mouse.x, g.mouse.y) {
                    g.ui.joinServerMenu.currServerType = i
                    if action == tview.MouseLeftClick {
                        ClearSelected(g.ui.joinServerMenu.serverType)
                        button.selected = true 
                        button.pressFunc()
                    }
                }
            }
            g.Box.MouseHandler()(action, event, setFocus)
            g.ui.joinServerMenu.serverList.MouseHandler()(action, event, setFocus)
        }

		return
	})
}

func CheckArgs(args []string) bool {
    cmp := []string{"c", "s", "tc", "ts"}
    for _, str := range cmp {
        if args[1] == ("-" + str) {
            return true 
        } 
    }

    return false
}

func main() {
    fmt.Printf("arg count: %d\n", len(os.Args))
    for i, arg := range os.Args {
        fmt.Printf("arg[%d]: %s\n", i, arg)
    } 

    if len(os.Args) < 2 {
        fmt.Printf("not enough args\n")
        os.Exit(1)
    } else if !CheckArgs(os.Args) {
        fmt.Printf("specify mode: client(-c) or server(-s)\n")
        os.Exit(1)
    } 

    switch os.Args[1] {
    case "-c":
        fmt.Printf("client mode selected\n")
        t := NewGame() 
        t.SetBorder(true)
        t.SetTitle("termblackjack")
        if err := t.app.SetRoot(t, true).EnableMouse(true).Run(); err != nil {
            panic(err)
        }
    case "-s":
        fmt.Printf("server mode selected\n")
        s := NewServer("server1", 4, 4)
        fmt.Printf("%v\n", s)
        for {

        }
    case "-tc":
        fmt.Printf("test client mode selected\n") 

        buf := make([]byte, 1024)
        broadcastAddr, err := net.ResolveUDPAddr("udp", GetIPBroadcastAddr(GetNetworkIP(true)) + ":" + ServerPort)
        if err != nil {
            panic(err)
        }

        broadcast, err := net.DialUDP("udp", nil, broadcastAddr)
        if err != nil {
            panic(err)
        }
        fmt.Printf("address: %s\n", broadcastAddr)

        addr, err := net.ResolveUDPAddr("udp", GetNetworkIP(false) + ":" + ServerPort)
        if err != nil {
            panic(err)
        }

        conn, err := net.ListenUDP("udp", addr)
        if err != nil {
            panic(err)
        }

        for {
            scanner := bufio.NewScanner(os.Stdin)
            fmt.Printf("> ")
            if !scanner.Scan() {

            }
            conn.SetReadDeadline(time.Now().Add(5000 * time.Millisecond))
            t1 := time.Now()
            _, err := broadcast.Write([]byte("hello"))
            if err != nil {
                panic(err)
            }

            _, remoteAddr, err := conn.ReadFromUDP(buf)
            if err != nil {
                opErr, ok := err.(*net.OpError)
                if ok && opErr.Timeout() {
                    fmt.Printf("se acabo el tiempo de espera\n")
                    continue
                }
                panic(err)
            }

            t2 := time.Now().Sub(t1).Nanoseconds()

            packet := bytes.NewBuffer(buf)  
            dec := gob.NewDecoder(packet)

            des := &PacketServerInfo{}

            err = dec.Decode(des)
            if err != nil {
                panic(err)
            }

            fmt.Printf("server found:\n")
            fmt.Printf("  address: %s\n", remoteAddr.String())
            fmt.Printf("  ping: %dms\n", t2)
            fmt.Printf("  max players: %d\n", des.MaxPlayers)
            fmt.Printf("  players: %d\n\n", des.PlayerCount)

            time.Sleep(1000 * time.Millisecond)
        }
    case "-ts":
        conn, err := net.Dial("udp", GetIPBroadcastAddr(GetNetworkIP(true)) + ":" + ServerPort)
        if err != nil {
            panic(err)
        }
        defer conn.Close()

        buf := make([]byte, 1024)
        for {
            _, err := conn.Read(buf)
            if err != nil {
                panic(err)
            }
            fmt.Printf("hola\n")

            _, err = conn.Write([]byte("skibi toilet"))
            if err != nil {
                panic(err)
            }
        }
        
//      addr, err := net.ResolveUDPAddr("udp", GetIPBroadcastAddr(GetNetworkIP()))
//      if err != nil {
//          panic(err)
//      }
//      fmt.Println(addr.String())

//      conn, err := net.DialUDP("udp", nil, addr)
//      if err != nil {
//          panic(err)
//      }
//      defer conn.Close()

//      buf := make([]byte, 1024)
//      for {
//          _, remoteAddr, err := conn.ReadFromUDP(buf)
//          if err != nil {
//              panic(err)
//          }

//          fmt.Println("connection from:", remoteAddr, "message:", buf)
//          conn.Write([]byte("server exists"))
//      }
    }

}
