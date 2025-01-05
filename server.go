package main

import (
	"bufio"
	"encoding/gob"
	"fmt"
	"net"

	// "time"
	"strconv"
	"bytes"
)

const (
    PacketPlayer int = iota
    PacketPing 
    ServerPort = "32004"
)

type PacketServerInfo struct {
    Id          string
    PlayerCount uint8
    MaxPlayers  uint8
}

type Server struct {
    Clients     []net.Conn
    Players     []Player
    gameStarted bool
    Id          string
    PlayerCount uint8
    MaxPlayers  uint8
    Latency     int64
    Addr        string
}

func NewServer(id string, playerCount uint8, maxPlayers uint8) Server {
    s := Server{
        Id: id,
        PlayerCount: playerCount,
        MaxPlayers: maxPlayers,
    }

//  listener, err := net.ListenPacket("udp", ":" + ServerListeningPort)
//  if err != nil {
//      panic(err)
//  }

//  udpAddr, err := net.ResolveUDPAddr("udp", GetIPBroadcastAddr(GetNetworkIP(true)) + ":" + ServerPort)
//  if err != nil {
//      panic(err)
//  }

    addr, err :=  net.ResolveUDPAddr("udp", GetIPBroadcastAddr(GetNetworkIP(true)) + ":" + ServerPort)
    if err != nil {
        panic(err)
    }

    conn, err := net.ListenUDP("udp", addr)
    if err != nil {
        panic(err)
    }

    buf := make([]byte, 1024)
    fmt.Printf("address: %s\n", addr.String())
    go func() {
        for !s.gameStarted {
            packet := &PacketServerInfo{
                Id: "server1",
                PlayerCount: s.PlayerCount,
                MaxPlayers: s.MaxPlayers, 
            }

            var packBuf bytes.Buffer
            enc := gob.NewEncoder(&packBuf)
            err = enc.Encode(packet)
            if err != nil {
                panic(err)
            }

            _, remoteAddr, err := conn.ReadFromUDP(buf)
            if err != nil {
                panic(err)
            }
            fmt.Printf("from addr %s: %s\n", remoteAddr.IP.String(), buf)

            // a is not necessary
            a, _ := net.ResolveUDPAddr("udp", remoteAddr.IP.String() + ":" + ServerPort)
            _, err = conn.WriteToUDP(packBuf.Bytes(), a)
            if err != nil {
                panic(err)
            }
            fmt.Printf("sended packet to %s\n", remoteAddr.IP.String() + ":" + ServerPort)
        }
        conn.Close()
    }()

//  go func() {
//      for {
//          conn, err := l.Accept()
//          if err != nil {
//              panic(err)
//          }

//          s.clients = append(s.clients, conn)
//          go s.Receive(conn)
//      }
//  }()

    return s
}

func (s *Server) Receive(conn net.Conn) {
    reader := bufio.NewReader(conn)
    fmt.Printf("read?\n")

    buf := make([]byte, 512)

    for {
        //      n, err := conn.Read(buf)
        //      if err != nil {
        //          fmt.Printf("error reading from connection %s: %s\n", conn.LocalAddr().String(), err.Error())
        //          return
        //      }

        //      msg := buf[:n]

        _, err := reader.Read(buf)
        if err != nil {
            fmt.Printf("error reading from connection %s: %s\n", conn.LocalAddr().String(), err.Error())
            return
        }

        _, err = conn.Write([]byte(""))
        if err != nil {
            panic(err)
        }



    }
}

func GetServerInfoStr(server Server) []string {
    var str []string 
    str = append(str, server.Id)
    str = append(str, strconv.Itoa(int(server.PlayerCount)) + "/" + strconv.Itoa(int(server.MaxPlayers)))
    str = append(str, strconv.Itoa(int(server.Latency)))

    return str
}

var separators = []string{"Server", "Players", "Latency"}
