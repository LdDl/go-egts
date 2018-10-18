package main

import (
	"fmt"
	"net"
)

func main() {

	fmt.Println("Launching server...")
	ln, _ := net.Listen("tcp", ":8081")
	conn, _ := ln.Accept()
	defer conn.Close()
	data := make([]byte, 65535)
	for {
		n, _ := conn.Read(data)

		if n != 0 {
			p, _ := packet.ReadPacket(data)
			//if err != nil {
			// fmt.Println("code error: ", err)
			//}
			// fmt.Println(data)
			p.Print()

			// fmt.Println(string(p.ServicesFrameData))

			// respPack, err := p.Response(0)
			// if err != nil {
			// 	fmt.Println(err)
			// }
			// resp := respPack.WritePacket()
			// fmt.Println(p.ResponseData)
			conn.Write(p.ResponseData)

			// pp, err := packet.ReadPacket(data)

		}

	}
}
