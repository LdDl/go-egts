package main

import (
	"fmt"
	"log"
	"net"

	"github.com/LdDl/go-egts/egts/packet"
)

var (
	port = 8081
)

func main() {
	conn, err := net.Dial("tcp", fmt.Sprintf("localhost:%d", port))
	if err != nil {
		fmt.Println(err)
		return
	}
	packetBytes := []byte{1, 0, 0, 11, 0, 40, 0, 17, 81, 1, 18, 29, 0, 17, 81, 1, 150, 147, 56, 49, 2, 2, 16, 26, 0, 154, 136, 129, 16, 16, 209, 106, 154, 124, 34, 200, 68, 129, 0, 0, 42, 0, 0, 0, 0, 16, 133, 0, 0, 0, 0, 49, 198}
	data := make([]byte, 65535)
	_, err = conn.Write(packetBytes)
	if err != nil {
		fmt.Println(err)
		return
	}
	n, err := conn.Read(data)
	if err != nil {
		fmt.Println(err)
		return
	}
	if n != 0 {
		p, err := packet.ReadPacket(data[:n])
		if err != nil {
			log.Println("Error", err)
		}
		log.Println("Response code:", p.PrepareAnswer(0, p.PacketID))
		log.Println("Packet:", p)
	}
}
