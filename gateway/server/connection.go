package server

import (
	"context"
	"crypto/subtle"
	"errors"
	"fmt"
	"io"
	"net"
	"time"

	"github.com/LdDl/go-egts/egts/packet"
	"github.com/LdDl/go-egts/egts/subrecord"
	"github.com/LdDl/go-egts/gateway/destination"
	"github.com/LdDl/go-egts/gateway/logger"
	"github.com/rs/zerolog/log"
)

func (s *server) handleConnection(ctx context.Context, conn net.Conn) error {
	authorized := !s.cfg.AuthCfg.Enabled
	source := destination.Source{RemoteAddress: conn.RemoteAddr().String()}
	var pid uint16
	var rn uint16
	var session *relaySession
	var relayIdentity []byte
	if s.cfg.DestinationsCfg.EGTS.Enabled {
		var err error
		session, err = s.newRelaySession()
		if err != nil {
			return err
		}
		defer func() {
			err := s.endRelaySession(session)
			if err != nil {
				log.Log().Str("scope", logger.SCOPE_DELIVERY).Str("event", logger.EVENT_DELIVERY_ERROR).
					Err(err).Msg("Can't close terminal relay session")
			}
		}()
	}
	for {
		raw, err := packet.ReadFrame(conn)
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
		nextRelayIdentity := relayIdentity
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
						if session != nil {
							nextRelayIdentity, err = data.Encode()
							if err != nil {
								return err
							}
						}
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
				item, err := s.enqueue(record, &relayPacket{session: session, identity: nextRelayIdentity})
				if err != nil {
					pkg.ErrorCode = packet.EGTS_PC_NO_RES_AVAIL
					if errors.Is(err, ErrRelayCredentials) {
						pkg.ErrorCode = packet.EGTS_PC_INC_DATAFORM
					}
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
			relayIdentity = nextRelayIdentity
		}
		if pkg.ErrorCode == packet.EGTS_PC_AUTH_DENIED {
			return nil
		}
	}
}
