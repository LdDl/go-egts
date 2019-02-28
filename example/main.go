package main

import (
	"fmt"
	"log"
	"net"
	"time"

	packet "github.com/LdDl/go-egts/egts/packet"
	"github.com/LdDl/go-egts/egts/subrecord"
)

var (
	port = "8081"
)

func main() {

	go func() {
		log.Println("Starting TCP server on port:", port)
		ln, err := net.Listen("tcp", ":"+port)
		if err != nil {
			log.Panicln(err)
		}
		conn, err := ln.Accept()
		if err != nil {
			log.Panicln(err)
		}
		defer conn.Close()

		buff := make([]byte, 65535)
		for {
			req, err := conn.Read(buff)
			if err != nil {
				log.Panicln(err)
			}

			data, responseCode := packet.ReadPacket(buff[:req])
			for i := range data.ServicesFrameData {
				switch data.ServicesFrameData[i].RecordData.SubrecordType {
				case 16:
					sub := data.ServicesFrameData[i].RecordData.SubrecordData.(subrecord.EgtsSrPosData)
					fmt.Printf(
						"SubRecordData:\n\tID: %v\n\tNavigationTime:%v\n\tLongitude:%v\n\tLatitude:%v\n\tSpeed:%v\n\tDirection:%v\n",
						data.ServicesFrameData[i].ObjectIdentifier, sub.NavigationTime,
						sub.Longitude, sub.Latitude,
						sub.Speed, sub.Direction,
					)
					break
				case 1:
					// log.Println("auth")
					break
				default:
					break
				}
			}
			fmt.Println("Response code:", responseCode)
			_, err = conn.Write(data.ResponseData)
			if err != nil {
				log.Printf("Can not write response:\n\tError: %v | IP: %v\n", err, conn.RemoteAddr())
				return
			}
			time.Sleep(100 * time.Millisecond)
		}
	}()

	log.Println("Sending bytes in 2 seconds...")
	time.Sleep(2 * time.Second)

	conn, err := net.Dial("tcp", "localhost:"+port)
	if err != nil {
		fmt.Println(err)
	}
	pac := []byte{1, 0, 0, 11, 0, 40, 0, 17, 81, 1, 18, 29, 0, 17, 81, 1, 150, 147, 56, 49, 2, 2, 16, 26, 0, 154, 136, 129, 16, 16, 209, 106, 154, 124, 34, 200, 68, 129, 0, 0, 42, 0, 0, 0, 0, 16, 133, 0, 0, 0, 0, 49, 198}
	data := make([]byte, 65535)
	conn.Write(pac)
	n, _ := conn.Read(data)

	if n != 0 {
		p, responseCode := packet.ReadPacket(data)
		log.Println("client;Response code:", responseCode)
		p.Print()
	}

}
