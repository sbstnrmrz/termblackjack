package main

import (
	//  "github.com/gdamore/tcell/v2"
	"math/bits"
	"net"
	"strconv"
	"strings"
	// "github.com/rivo/tview"
)

type Client struct {
    player Player
    conn net.Conn
}

func GetNetworkIP(netMask bool) string {
    addrs, err := net.InterfaceAddrs()
    if err != nil {
        panic(err)
    }
    var ipStr string
    for _, addr := range addrs {
        ipnet := addr.(*net.IPNet);
        if !ipnet.IP.IsLoopback() {
            if ipnet.IP.To4() != nil {
                mask, _ := strconv.ParseUint(ipnet.Mask.String(), 16, 32)
                ipStr = ipnet.IP.String()
                if netMask {
                    ipStr += "/" + strconv.Itoa(bits.OnesCount64(mask))
                }
            }
        }
    }

    return ipStr
}

func GetIPBroadcastAddr(addr string) string {
    netMaskBits, _ := strconv.Atoi(addr[len(addr)-2:])
    ip := addr[:len(addr)-3]
    subStr := strings.Split(ip, ".")
    var result string
    var netMaskStr string 
    for i := 0; i < 32; i++ {
        if i < netMaskBits {
            netMaskStr += "1"
        } else {
            netMaskStr += "0"
        }
    }

    for i, str := range subStr {
        netMask, _ := strconv.ParseInt(netMaskStr[i*8:(i+1)*8], 2, 64)
        conv, _ := strconv.Atoi(str)
//      fmt.Printf("ip %d | net %d\n", conv1, int(netMask) ^ 255)
        result += strconv.Itoa(conv | (int(netMask) ^ 255)) + "."
    }
    result = result[:len(result)-1]

    return result
}
