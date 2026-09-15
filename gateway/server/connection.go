package server

import (
	"context"
	"crypto/subtle"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"
	"time"

	"github.com/LdDl/go-egts/crc"
	"github.com/LdDl/go-egts/egts/packet"
	"github.com/LdDl/go-egts/egts/subrecord"
	"github.com/LdDl/go-egts/gateway/destination"
	"github.com/LdDl/go-egts/gateway/logger"
	"github.com/rs/zerolog/log"
)

func readPacket(conn io.Reader) ([]byte, error) {
	header := make([]byte, 10)
	_, err := io.ReadFull(conn, header)
	if err != nil {
		return nil, err
	}
	headerLength := 11
	if header[2]&0x20 != 0 {
		headerLength = 16
	}
	if header[0] != 1 || int(header[3]) != headerLength {
		return nil, fmt.Errorf("Invalid EGTS header")
	}
	header = append(header, make([]byte, headerLength-10)...)
	_, err = io.ReadFull(conn, header[10:])
	if err != nil {
		if errors.Is(err, io.EOF) {
			return nil, io.ErrUnexpectedEOF
		}
		return nil, err
	}
	if byte(crc.Crc(8, header[:headerLength-1])) != header[headerLength-1] {
		return nil, fmt.Errorf("Invalid EGTS header checksum")
	}
	bodyLength := int(binary.LittleEndian.Uint16(header[5:7]))
	if bodyLength > 0 {
		bodyLength += 2
	}
	if headerLength+bodyLength > 65535 {
		return nil, fmt.Errorf("EGTS packet exceeds 65535 bytes")
	}
	raw := append(header, make([]byte, bodyLength)...)
	_, err = io.ReadFull(conn, raw[headerLength:])
	if err != nil {
		if errors.Is(err, io.EOF) {
			return nil, io.ErrUnexpectedEOF
		}
		return nil, err
	}
	return raw, nil
}

func (s *server) handleConnection(ctx context.Context, conn net.Conn) error {
	authorized := !s.cfg.AuthCfg.Enabled
	source := destination.Source{RemoteAddress: conn.RemoteAddr().String()}
	var pid uint16
	var rn uint16
	for {
		raw, err := readPacket(conn)
		if errors.Is(err, io.EOF) {
			return nil
		}
		if err != nil {
			return fmt.Errorf("Can't read EGTS packet: %w", err)
		}
		receivedAt := time.Now().UTC()
		pkg, decodeErr := packet.ReadPacket(raw)
		if decodeErr != nil {
			log.Log().Str("scope", logger.SCOPE_SERVER).Str("event", logger.EVENT_PACKET_REJECTED).
				Str("remote_address", source.RemoteAddress).Err(decodeErr).Msg("Invalid EGTS packet")
			if pkg.PacketType != packet.EGTS_PT_APPDATA {
				continue
			}
		} else if pkg.PacketType != packet.EGTS_PT_APPDATA {
			continue
		}
		authRequest := false
		identityRequest := false
		hasCredentials := false
		onlyIdentity := true
		requestCredentials := false
		nextAuthorized := authorized
		nextSource := source
		if decodeErr == nil {
			services := pkg.ServicesFrameData.(*packet.ServicesFrameData)
			for _, service := range *services {
				if service.SourceServiceType != packet.SERVICE_AUTH || service.RecipientServiceType != packet.SERVICE_AUTH {
					onlyIdentity = false
					continue
				}
				for _, record := range service.RecordsData {
					switch data := record.SubrecordData.(type) {
					case *subrecord.SRTermIdentity:
						id := data.TerminalIdentifier
						nextSource.TerminalID = &id
						authRequest = true
						identityRequest = true
					case *subrecord.SRAuthInfo:
						hasCredentials = true
						onlyIdentity = false
						authRequest = true
						if s.cfg.AuthCfg.Enabled {
							valid := subtle.ConstantTimeCompare([]byte(data.Password), []byte(s.cfg.AuthCfg.Password)) == 1
							if !valid || data.ServerSequence != nil {
								pkg.ErrorCode = packet.EGTS_PC_AUTH_DENIED
							} else {
								nextAuthorized = true
							}
						}
					default:
						onlyIdentity = false
					}
				}
			}
			requestCredentials = !authorized && identityRequest && onlyIdentity && !hasCredentials
			if !nextAuthorized && !requestCredentials {
				pkg.ErrorCode = packet.EGTS_PC_AUTH_DENIED
			}
			if pkg.ErrorCode == packet.EGTS_PC_OK && !requestCredentials {
				record := &destination.Record{ReceivedAt: receivedAt, Source: nextSource, Packet: pkg, Raw: raw}
				item, err := s.enqueue(record)
				if err != nil {
					pkg.ErrorCode = packet.EGTS_PC_NO_RES_AVAIL
				} else if s.cfg.DeliveryCfg.AckMode == "delivered" {
					select {
					case <-item.done:
					case <-ctx.Done():
						return nil
					}
				}
			}
		}
		pid++
		answer := pkg.PrepareAnswer(rn, pid)
		response := answer.ServicesFrameData.(*packet.PTResponse)
		if response.SDR != nil {
			for _, service := range *response.SDR.(*packet.ServicesFrameData) {
				rn++
				service.RecordNumber = rn
			}
		}
		encoded, err := answer.Encode()
		if err != nil {
			return err
		}
		err = conn.SetWriteDeadline(time.Now().Add(30 * time.Second))
		if err != nil {
			return err
		}
		n, err := conn.Write(encoded)
		if err != nil {
			return fmt.Errorf("Can't write EGTS response: %w", err)
		}
		if n != len(encoded) {
			return io.ErrShortWrite
		}
		if authRequest && decodeErr == nil {
			pid++
			rn++
			result := pkg.PrepareSRResultCode(pkg.ErrorCode, rn, pid)
			if requestCredentials {
				services := result.ServicesFrameData.(*packet.ServicesFrameData)
				(*services)[0].RecordsData[0] = &packet.RecordData{
					SubrecordType: packet.AuthParams, SubrecordLength: 1, SubrecordData: &subrecord.SRAuthParams{},
				}
			}
			encoded, err = result.Encode()
			if err != nil {
				return err
			}
			n, err = conn.Write(encoded)
			if err != nil {
				return fmt.Errorf("Can't write EGTS authentication result: %w", err)
			}
			if n != len(encoded) {
				return io.ErrShortWrite
			}
			log.Log().Str("scope", logger.SCOPE_SERVER).Str("event", logger.EVENT_RECIEVED_AUTH).
				Str("remote_address", source.RemoteAddress).Uint8("result", pkg.ErrorCode).Msg("Processed EGTS authentication")
		}
		if pkg.ErrorCode == packet.EGTS_PC_OK {
			authorized = nextAuthorized
			source = nextSource
		}
		if pkg.ErrorCode == packet.EGTS_PC_AUTH_DENIED {
			return nil
		}
	}
}
