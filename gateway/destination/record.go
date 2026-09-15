package destination

import (
	"bytes"
	"encoding/json"
	"fmt"
	"time"

	"github.com/LdDl/go-egts/egts/packet"
	"github.com/LdDl/go-egts/gateway/logger"
	"github.com/rs/zerolog"
)

type Source struct {
	RemoteAddress string  `json:"remote_address"`
	TerminalID    *uint32 `json:"terminal_id"`
}

type Record struct {
	ReceivedAt time.Time     `json:"received_at"`
	Source     Source        `json:"source"`
	Packet     packet.Packet `json:"packet"`
	// JSON encodes the original wire bytes as base64.
	Raw []byte `json:"raw"`
}

func NewRecord(receivedAt time.Time, source Source, raw []byte) (*Record, error) {
	decoded, err := packet.ReadPacket(raw)
	if err != nil {
		return nil, fmt.Errorf("Can't decode packet for destination: %w", err)
	}
	if source.TerminalID != nil {
		terminalID := *source.TerminalID
		source.TerminalID = &terminalID
	}
	return &Record{
		ReceivedAt: receivedAt.UTC(),
		Source:     source,
		Packet:     decoded,
		Raw:        append([]byte(nil), raw...),
	}, nil
}

func (r *Record) Encode() ([]byte, error) {
	if r == nil {
		return nil, fmt.Errorf("Packet record is nil")
	}
	data, err := json.Marshal(r)
	if err != nil {
		return nil, fmt.Errorf("Can't encode packet record: %w", err)
	}
	// Format in memory so delivery errors are returned by the destination writer.
	var buffer bytes.Buffer
	packetLogger := zerolog.New(&buffer)
	packetLogger.Log().
		Str("application", "egts_gateway").
		Str("scope", logger.SCOPE_PACKETS).
		Str("event", logger.EVENT_PACKET_RECEIVED).
		RawJSON("record", data).
		Send()
	if buffer.Len() == 0 {
		return nil, fmt.Errorf("Packet record was not formatted")
	}
	return buffer.Bytes(), nil
}
