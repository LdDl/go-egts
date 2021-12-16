package main

import (
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"net"
	"sync/atomic"
	"time"

	"github.com/LdDl/go-egts/egts/packet"
	"github.com/LdDl/go-egts/egts/subrecord"
	"github.com/pkg/errors"
)

var (
	port                = 8081
	headerLen           = 10
	pidCounter          uint32
	rnCounter           uint32
	ErrNoEGTSPacket     = fmt.Errorf("no EGTS packet")
	ErrAcceptEGTSPacket = fmt.Errorf("can't accept EGTS packet")
)

func main() {
	tcpAddr, err := net.ResolveTCPAddr("tcp4", fmt.Sprintf(":%d", port))
	if err != nil {
		fmt.Println(err)
		return
	}
	ln, err := net.ListenTCP("tcp", tcpAddr)
	if err != nil {
		fmt.Println(err)
		return
	}
	log.Printf("Accept connection on port %d\n", port)
	for {
		conn, err := ln.AcceptTCP()
		if err != nil {
			fmt.Println(err)
			return
		}
		log.Printf("Calling handleConnection for remote address: %s\n", conn.RemoteAddr().String())
		go handleConnection(conn)
	}
}

func getNextPid() uint16 {
	if pidCounter < 65535 {
		atomic.AddUint32(&pidCounter, 1)
	} else {
		pidCounter = 0
	}
	return uint16(atomic.LoadUint32(&pidCounter))
}

func getNextRN() uint16 {
	if rnCounter < 65535 {
		atomic.AddUint32(&rnCounter, 1)
	} else {
		rnCounter = 0
	}
	return uint16(atomic.LoadUint32(&rnCounter))
}

func handleConnection(conn *net.TCPConn) error {
	defer conn.Close()
	err := conn.SetKeepAlive(true)
	if err != nil {
		return errors.Wrap(err, "Can't set keep-alive to 'true'")
	}

	recvPacket := []byte{}
	for {
	Received:
		recvPacket = nil

		// Read packet header
		headerBuf := make([]byte, headerLen)
		_, err := conn.Read(headerBuf)

		switch err {
		case nil:
			// Check if packet is EGTS
			if headerBuf[0] != 0x01 {
				log.Printf("Packet header from '%s' is not for EGTS", conn.RemoteAddr().String())
				conn.Close()
				return ErrNoEGTSPacket
			}
			// Evaluate length of packet as: HL (header length) + FDL (body length) + CRC (2 bytes if FDS exists)
			bodyLen := binary.LittleEndian.Uint16(headerBuf[5:7])
			pkgLen := uint16(headerBuf[3])
			if bodyLen > 0 {
				pkgLen += bodyLen + 2
			}
			// Recieve end of EGTS packet
			buf := make([]byte, pkgLen-uint16(headerLen))
			if _, err := io.ReadFull(conn, buf); err != nil {
				log.Printf("Can't read packet body from '%s' due the error: %s", conn.RemoteAddr().String(), err.Error())
				conn.Close()
				return ErrAcceptEGTSPacket
			}
			// Prepare full packet
			recvPacket = append(headerBuf, buf...)
		case io.EOF:
			log.Printf("Closing connection to '%s' due timeout", conn.RemoteAddr().String())
			conn.Close()
			return nil
		default:
			conn.Close()
			return nil
		}

		pkg := packet.Packet{}
		pkg, err = packet.ReadPacket(recvPacket)
		if err != nil {
			log.Printf("Can't parse EGTS packet from '%s' due the error: %s", conn.RemoteAddr().String(), err.Error())
			goto Received
		}
		currentTime := time.Now()
		srResultCode := packet.Packet{}
		switch pkg.PacketType {
		case packet.EGTS_PT_APPDATA:
			sfrd := pkg.ServicesFrameData.(*packet.ServicesFrameData)
			for i := range *sfrd {
				oid := (*sfrd)[i].ObjectIdentifier
				rd := (*sfrd)[i].RecordsData
				for r := range rd {
					switch rd[r].SubrecordType {
					case packet.PosData:
						switch rd[r].SubrecordData.(type) {
						case *subrecord.SRPosData:
							pos := rd[r].SubrecordData.(*subrecord.SRPosData)
							log.Printf("PosData is:\n\tOID: %d | Longitude: %f | Latitude: %f | Time: %v\n", oid, pos.Longitude, pos.Latitude, &currentTime)
						default:
							// Nothing
						}
					case packet.TermIdentity:
						switch rd[r].SubrecordData.(type) {
						case *subrecord.SRTermIdentity:
							term := rd[r].SubrecordData.(*subrecord.SRTermIdentity)
							log.Printf("SRTermIdentity is:\n\tOID: %d | MSISDN: %s | IMSI: %s\n", oid, term.MobileStationIntegratedServicesDigitalNetworkNumber, term.InternationalMobileSubscriberIdentity)
							srResultCode = pkg.PrepareSRResultCode(packet.EGTS_PC_OK, getNextRN(), getNextPid())
						default:
							// Nothing
						}
					default:
					}
				}
			}
		default:
			// Nothing
		}
		pkgResp := pkg.PrepareAnswer(getNextRN(), getNextPid())
		resp := pkgResp.Encode()
		_, err = conn.Write(resp)
		if err != nil {
			log.Printf("Can't write response to '%s' due the error: %s", conn.RemoteAddr().String(), err.Error())
			continue
		}
		srResultCodeBytes := srResultCode.Encode()
		if len(srResultCodeBytes) > 0 {
			_, err = conn.Write(srResultCodeBytes)
			if err != nil {
				log.Printf("Can't send result code to '%s' due the error: %s", conn.RemoteAddr().String(), err.Error())
				continue
			} else {
				log.Printf("Result code has been sent to '%s'", conn.RemoteAddr().String())
			}
		}
	}
}
