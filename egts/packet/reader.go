package packet

import (
	"encoding/binary"
	"errors"
	"fmt"
	"github.com/LdDl/go-egts/crc"
	"io"
)

// ReadFrame reads one complete EGTS transport frame from a stream.
func ReadFrame(conn io.Reader) ([]byte, error) {
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
