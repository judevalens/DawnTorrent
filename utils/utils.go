package utils

import (
	"fmt"
	upnp "github.com/huin/goupnp/dcps/internetgateway1"
	"log"
	"net"
	"strconv"
	"strings"
	"torrent/parser"
)

const DEBUG = true
const PORT = 6881
const PORT2 = 6882
const UpFlag = "up"

var LocalAddr, _ =  net.ResolveTCPAddr("tcp",LocalAddress().String()+":"+strconv.Itoa(PORT))
var MyID = parser.GetRandomId()


func Debugln(st string){
	if DEBUG {
		fmt.Println(st)
	}
}

func LocalAddress() net.IP {
	list, err := net.Interfaces()
	if err != nil {
		panic(err)
	}

	for _, iface := range list {
		
		fmt.Printf("%v flag %v index %v\n", iface.Name, iface.Flags.String(), iface.Index)

		interfaceAddrs, _ := iface.Addrs()
		for _, addr := range interfaceAddrs {
			ip, _, _ := net.ParseCIDR(addr.String())

			flags := strings.Split(iface.Flags.String(), "|")

			if !ip.IsLoopback() {
				addrArr, _ := iface.Addrs()

				for _, flag := range flags {
					if flag == "up" {
						fmt.Printf("ip %v addr %v\n", addrArr[len(addrArr)-1].String(), ip)
						if ip.To4() != nil {
							return ip
						}

					}
				}

			}

		}

	}

	return net.IP{}
}

/// That's a work in progress
/// for now I will manually forward the port

func forwardPort(port string) {

	connectionClient, _, err := upnp.NewWANIPConnection1Clients()

	if err != nil {

		log.Fatal(err)

	}

	ip := connectionClient[0].AddPortMapping("", PORT, "tcp", PORT, LocalAddress().String(), true, "for torrent 2", 100000)

	fmt.Printf("len %v name %v\n", len(connectionClient), ip)

}