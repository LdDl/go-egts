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
	return uint16(atomic.AddUint32(&pidCounter, 1))
}

func getNextRN() uint16 {
	return uint16(atomic.AddUint32(&rnCounter, 1))
}

func handleConnection(conn *net.TCPConn) error {
	defer conn.Close()
	err := conn.SetKeepAlive(true)
	if err != nil {
		return errors.Wrap(err, "Can't set keep-alive to 'true'")
	}

	for {
		// Read packet header
		headerBuf := make([]byte, headerLen)
		_, err = io.ReadFull(conn, headerBuf)
		if err == io.EOF {
			log.Printf("Connection from '%s' has been closed", conn.RemoteAddr().String())
			return nil
		}
		if err != nil {
			return fmt.Errorf("%w: reading packet header: %v", ErrAcceptEGTSPacket, err)
		}
		// Check if packet is EGTS
		if headerBuf[0] != 0x01 {
			log.Printf("Packet header from '%s' is not for EGTS", conn.RemoteAddr().String())
			return ErrNoEGTSPacket
		}
		expectedHeaderLen := byte(11)
		if headerBuf[2]&0x20 != 0 {
			expectedHeaderLen = 16
		}
		if headerBuf[3] != expectedHeaderLen {
			return fmt.Errorf("%w: invalid header length %d", ErrAcceptEGTSPacket, headerBuf[3])
		}
		// HL + FDL + CRC (2 bytes when FDL is not zero).
		bodyLen := int(binary.LittleEndian.Uint16(headerBuf[5:7]))
		pkgLen := int(headerBuf[3]) + bodyLen
		if bodyLen > 0 {
			pkgLen += 2
		}
		if pkgLen > 65535 {
			return fmt.Errorf("%w: packet length exceeds 65535 bytes", ErrAcceptEGTSPacket)
		}
		// Receive the rest of the packet.
		buf := make([]byte, pkgLen-headerLen)
		_, err = io.ReadFull(conn, buf)
		if err != nil {
			log.Printf("Can't read packet body from '%s' due the error: %s", conn.RemoteAddr().String(), err.Error())
			return fmt.Errorf("%w: reading packet body: %v", ErrAcceptEGTSPacket, err)
		}
		recvPacket := append(headerBuf, buf...)

		pkg := packet.Packet{}
		pkg, err = packet.ReadPacket(recvPacket)
		if err != nil {
			log.Printf("Can't parse EGTS packet from '%s' due the error: %s", conn.RemoteAddr().String(), err.Error())
			if pkg.PacketType == packet.EGTS_PT_APPDATA && (pkg.ErrorCode == packet.EGTS_PC_DATACRC_ERROR || pkg.ErrorCode == packet.EGTS_PC_INC_DATAFORM) {
				answer := pkg.PrepareAnswer(0, getNextPid())
				encoded, err := answer.Encode()
				if err != nil {
					return errors.Wrap(err, "Can't encode error response")
				}
				_, err = conn.Write(encoded)
				if err != nil {
					return errors.Wrap(err, "Can't write error response")
				}
			}
			continue
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
			continue
		}
		pkgResp := pkg.PrepareAnswer(0, getNextPid())
		confirmation := pkgResp.ServicesFrameData.(*packet.PTResponse)
		if confirmation.SDR != nil {
			for _, record := range *confirmation.SDR.(*packet.ServicesFrameData) {
				record.RecordNumber = getNextRN()
			}
		}
		resp, err := pkgResp.Encode()
		if err != nil {
			return errors.Wrap(err, "Can't encode response")
		}
		_, err = conn.Write(resp)
		if err != nil {
			log.Printf("Can't write response to '%s' due the error: %s", conn.RemoteAddr().String(), err.Error())
			return errors.Wrap(err, "Can't write response")
		}
		if srResultCode.ServicesFrameData != nil {
			srResultCodeBytes, err := srResultCode.Encode()
			if err != nil {
				return errors.Wrap(err, "Can't encode result code")
			}
			_, err = conn.Write(srResultCodeBytes)
			if err != nil {
				log.Printf("Can't send result code to '%s' due the error: %s", conn.RemoteAddr().String(), err.Error())
				return errors.Wrap(err, "Can't send result code")
			}
			log.Printf("Result code has been sent to '%s'", conn.RemoteAddr().String())
		}
	}
}
