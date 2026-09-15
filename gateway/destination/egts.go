package destination

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net"
	"strconv"
	"sync"
	"time"

	"github.com/LdDl/go-egts/egts/packet"
	"github.com/LdDl/go-egts/egts/subrecord"
	"github.com/LdDl/go-egts/gateway/configuration"
)

// EGTS owns the outgoing connection for one incoming terminal session.
type EGTS struct {
	mu       sync.Mutex
	cfg      configuration.EGTSDestinationConf
	server   configuration.ServerConf
	conn     net.Conn
	identity []byte
	closed   bool
}

type recordConfirmation struct {
	number    uint16
	source    uint8
	recipient uint8
}

func PrepareEGTS(cfg configuration.EGTSDestinationConf, server configuration.ServerConf) (*EGTS, error) {
	if !cfg.Enabled {
		return nil, fmt.Errorf("EGTS destination is disabled")
	}
	err := cfg.Validate()
	if err != nil {
		return nil, err
	}
	return &EGTS{cfg: cfg, server: server}, nil
}

func (e *EGTS) WriteContext(ctx context.Context, record *Record, identity []byte) (err error) {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.closed {
		return net.ErrClosed
	}
	if record == nil {
		return fmt.Errorf("EGTS packet record is nil")
	}
	if ctx.Err() != nil {
		return ctx.Err()
	}
	pkg, err := packet.ReadPacket(record.Raw)
	if err != nil {
		return err
	}
	if pkg.PacketType != packet.EGTS_PT_APPDATA {
		return fmt.Errorf("EGTS relay requires an APPDATA packet")
	}
	expectAuth := false
	for _, service := range *pkg.ServicesFrameData.(*packet.ServicesFrameData) {
		for _, item := range service.RecordsData {
			if item.SubrecordType == packet.AuthInfo {
				return fmt.Errorf("Incoming credentials cannot be relayed; configure destination authentication")
			}
			if item.SubrecordType == packet.TermIdentity && service.SourceServiceType == packet.SERVICE_AUTH && service.RecipientServiceType == packet.SERVICE_AUTH {
				expectAuth = true
			}
		}
	}
	defer func() {
		if err != nil && e.conn != nil {
			closeErr := e.conn.Close()
			e.conn = nil
			if closeErr != nil {
				err = fmt.Errorf("%w; Can't close EGTS connection: %v", err, closeErr)
			}
		}
	}()
	if e.conn != nil && !bytes.Equal(e.identity, identity) {
		err = e.conn.Close()
		e.conn = nil
		if err != nil {
			return err
		}
	}
	connected := false
	if e.conn == nil {
		dialer := net.Dialer{Timeout: time.Duration(e.cfg.ConnectTimeoutSeconds) * time.Second}
		e.conn, err = dialer.DialContext(ctx, "tcp", net.JoinHostPort(e.cfg.Host, strconv.Itoa(e.cfg.Port)))
		if err != nil {
			return fmt.Errorf("Can't connect to EGTS destination: %w", err)
		}
		remote := e.conn.RemoteAddr().(*net.TCPAddr)
		local := e.conn.LocalAddr().(*net.TCPAddr)
		listeningIP := net.ParseIP(e.server.Host)
		if remote.Port == e.server.Port && remote.IP.Equal(local.IP) && (listeningIP.IsUnspecified() || listeningIP.Equal(remote.IP) || e.server.Host == "localhost") {
			return fmt.Errorf("EGTS destination points back to this gateway")
		}
		connected = true
	}
	conn := e.conn
	finished := make(chan struct{})
	interrupted := make(chan error, 1)
	go func() {
		select {
		case <-ctx.Done():
			interrupted <- conn.SetDeadline(time.Now())
		case <-finished:
			interrupted <- nil
		}
	}()
	defer func() {
		close(finished)
		interruptErr := <-interrupted
		if err == nil && interruptErr != nil {
			err = interruptErr
		}
	}()
	if connected {
		err = e.authenticate(ctx, pkg.PacketID, identity)
		if err != nil {
			return err
		}
		e.identity = append([]byte(nil), identity...)
	}
	required, err := e.exchange(ctx, &pkg, record.Raw, expectAuth)
	if err != nil {
		return err
	}
	if required {
		return fmt.Errorf("EGTS destination requested authentication while receiving data")
	}
	return nil
}

func (e *EGTS) authenticate(ctx context.Context, pid uint16, identity []byte) error {
	var records []packet.BytesData
	if len(identity) > 0 {
		term := &subrecord.SRTermIdentity{}
		err := term.Decode(identity)
		if err != nil {
			return fmt.Errorf("Invalid relay terminal identity: %w", err)
		}
		records = append(records, term)
	}
	if e.cfg.Auth.Enabled {
		records = append(records, &subrecord.SRAuthInfo{UserName: e.cfg.Auth.UserName, Password: e.cfg.Auth.Password})
	}
	pid -= uint16(len(records))
	for _, data := range records {
		srt := packet.TermIdentity
		_, credentials := data.(*subrecord.SRAuthInfo)
		if credentials {
			srt = packet.AuthInfo
		}
		service := &packet.ServiceDataRecord{
			RecordNumber: pid, SSOD: "0", RSOD: "0", GRP: "0", RPP: "00", TMFE: "0", EVFE: "0", OBFE: "0",
			SourceServiceType: packet.SERVICE_AUTH, RecipientServiceType: packet.SERVICE_AUTH,
			RecordsData: packet.RecordsData{&packet.RecordData{SubrecordType: srt, SubrecordLength: data.Len(), SubrecordData: data}},
		}
		service.RecordLength = service.RecordsData.Len()
		services := packet.ServicesFrameData{service}
		pkg := packet.Packet{
			ProtocolVersion: 1, PRF: "00", PR: "00", CMP: "0", ENA: "00", RTE: "0", HeaderLength: 11,
			FrameDataLength: services.Len(), PacketID: pid, PacketType: packet.EGTS_PT_APPDATA, ServicesFrameData: &services,
		}
		raw, err := pkg.Encode()
		if err != nil {
			return err
		}
		required, err := e.exchange(ctx, &pkg, raw, true)
		if err != nil {
			return fmt.Errorf("EGTS destination authentication failed: %w", err)
		}
		if required && (!e.cfg.Auth.Enabled || credentials) {
			return fmt.Errorf("EGTS destination requires authentication")
		}
		pid++
	}
	return nil
}

func (e *EGTS) exchange(ctx context.Context, pkg *packet.Packet, raw []byte, expectAuth bool) (bool, error) {
	if ctx.Err() != nil {
		return false, ctx.Err()
	}
	deadline := time.Now().Add(time.Duration(e.cfg.AckTimeoutSeconds) * time.Second)
	contextDeadline, exists := ctx.Deadline()
	if exists && contextDeadline.Before(deadline) {
		deadline = contextDeadline
	}
	err := e.conn.SetDeadline(deadline)
	if err != nil {
		return false, err
	}
	pending := make(map[recordConfirmation]bool)
	for _, service := range *pkg.ServicesFrameData.(*packet.ServicesFrameData) {
		for _, record := range service.RecordsData {
			if record.SubrecordType != packet.RecordResponse {
				pending[recordConfirmation{number: service.RecordNumber, source: service.RecipientServiceType, recipient: service.SourceServiceType}] = true
			}
		}
	}
	n, err := e.conn.Write(raw)
	if err != nil {
		return false, fmt.Errorf("Can't send EGTS packet: %w", err)
	}
	if n != len(raw) {
		return false, io.ErrShortWrite
	}
	transportOK := false
	authOK := !expectAuth
	required := false
	for !transportOK || len(pending) > 0 || !authOK {
		frame, err := packet.ReadFrame(e.conn)
		if err != nil {
			return false, fmt.Errorf("Can't receive EGTS confirmation: %w", err)
		}
		response, err := packet.ReadPacket(frame)
		if err != nil {
			return false, fmt.Errorf("Invalid EGTS confirmation: %w", err)
		}
		var services *packet.ServicesFrameData
		if response.PacketType == packet.EGTS_PT_RESPONSE {
			confirmation := response.ServicesFrameData.(*packet.PTResponse)
			if confirmation.ResponsePacketID != pkg.PacketID {
				continue
			}
			if confirmation.ProcessingResult != packet.EGTS_PC_OK && confirmation.ProcessingResult != packet.EGTS_PC_IN_PROGRESS {
				return false, fmt.Errorf("EGTS packet %d rejected with code %d", pkg.PacketID, confirmation.ProcessingResult)
			}
			transportOK = confirmation.ProcessingResult == packet.EGTS_PC_OK
			if confirmation.SDR != nil {
				services = confirmation.SDR.(*packet.ServicesFrameData)
			}
		} else {
			services = response.ServicesFrameData.(*packet.ServicesFrameData)
		}
		if services != nil {
			for _, service := range *services {
				for _, record := range service.RecordsData {
					switch data := record.SubrecordData.(type) {
					case *subrecord.SRRecordResponse:
						key := recordConfirmation{number: data.ConfirmedRecordNumber, source: service.SourceServiceType, recipient: service.RecipientServiceType}
						if !pending[key] {
							continue
						}
						if data.RecordStatus == packet.EGTS_PC_OK {
							delete(pending, key)
						} else if data.RecordStatus != packet.EGTS_PC_IN_PROGRESS {
							return false, fmt.Errorf("EGTS record %d rejected with code %d", data.ConfirmedRecordNumber, data.RecordStatus)
						}
					case *subrecord.SRResultCode:
						if !expectAuth || service.SourceServiceType != packet.SERVICE_AUTH || service.RecipientServiceType != packet.SERVICE_AUTH {
							continue
						}
						if data.RCD != packet.EGTS_PC_OK && data.RCD != packet.EGTS_PC_IN_PROGRESS {
							return false, fmt.Errorf("EGTS authentication rejected with code %d", data.RCD)
						}
						authOK = data.RCD == packet.EGTS_PC_OK
					case *subrecord.SRAuthParams:
						if !expectAuth || service.SourceServiceType != packet.SERVICE_AUTH || service.RecipientServiceType != packet.SERVICE_AUTH {
							return false, fmt.Errorf("Unexpected EGTS authentication request")
						}
						required = true
						authOK = true
					default:
						return false, fmt.Errorf("Unexpected EGTS destination subrecord %d", record.SubrecordType)
					}
				}
			}
		}
		if response.PacketType == packet.EGTS_PT_APPDATA {
			answer := response.PrepareAnswer(0, response.PacketID)
			encoded, err := answer.Encode()
			if err != nil {
				return false, err
			}
			n, err = e.conn.Write(encoded)
			if err != nil {
				return false, err
			}
			if n != len(encoded) {
				return false, io.ErrShortWrite
			}
		}
	}
	return required, nil
}

func (e *EGTS) Close() error {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.closed = true
	if e.conn == nil {
		return nil
	}
	err := e.conn.Close()
	e.conn = nil
	return err
}
