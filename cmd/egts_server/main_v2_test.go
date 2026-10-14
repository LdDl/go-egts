package main

import (
	"encoding/binary"
	"encoding/hex"
	"io"
	"net"
	"testing"
	"time"

	"github.com/LdDl/go-egts/egts/packet"
	"github.com/LdDl/go-egts/egts/subrecord"
	"github.com/stretchr/testify/assert"
)

func TestHandleConnectionV2(t *testing.T) {
	previous := serviceVersion
	serviceVersion = 2
	defer func() { serviceVersion = previous }()
	telemetry, err := hex.DecodeString("0100000b003e000b00018f27006500071122334455667788785634122a00000002021024002a000000ffffffffffffffff837bc02c563412a5042143057856341285ff57e80300feffc582")
	assert.NoError(t, err)
	if err != nil {
		return
	}
	identity, err := hex.DecodeString("0100000b006a000a00016b53006400071122334455667788785634122a0000000101010b008877665544332211003032031e003132333435363738394142434445464748010000000300000045585452410c2100004131323342430000000000000000000000000000000000000000000000000000282c")
	assert.NoError(t, err)
	if err != nil {
		return
	}
	identityResponse := serverResponseTestCase{
		length:          29,
		packetType:      packet.EGTS_PT_RESPONSE,
		confirmedPacket: 10,
		services:        []serverServiceResponseTestCase{{confirmedRecords: []uint16{100}, serviceType: packet.SERVICE_AUTH}},
	}
	resultResponse := serverResponseTestCase{
		length:     24,
		packetType: packet.EGTS_PT_APPDATA,
		services:   []serverServiceResponseTestCase{{serviceType: packet.SERVICE_AUTH}},
	}
	telemetryResponse := serverResponseTestCase{
		length:          29,
		packetType:      packet.EGTS_PT_RESPONSE,
		confirmedPacket: 11,
		services:        []serverServiceResponseTestCase{{confirmedRecords: []uint16{101}, serviceType: packet.SERVICE_DATA}},
	}
	cases := []serverConnectionTestCase{
		{
			name:      "authentication",
			chunks:    [][]byte{identity},
			responses: []serverResponseTestCase{identityResponse, resultResponse},
		},
		{
			name:      "telemetry",
			chunks:    [][]byte{telemetry},
			responses: []serverResponseTestCase{telemetryResponse},
		},
		{
			name:      "authentication then telemetry",
			chunks:    [][]byte{append(append([]byte(nil), identity...), telemetry...)},
			responses: []serverResponseTestCase{identityResponse, resultResponse, telemetryResponse},
		},
		{
			name:      "fragmented telemetry",
			chunks:    [][]byte{telemetry[:1], telemetry[1:10], telemetry[10:11], telemetry[11:]},
			responses: []serverResponseTestCase{telemetryResponse},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			listener, err := net.ListenTCP("tcp4", &net.TCPAddr{IP: net.IPv4(127, 0, 0, 1)})
			assert.NoError(t, err)
			if err != nil {
				return
			}
			defer listener.Close()
			done := make(chan error, 1)
			go func() {
				server, err := listener.AcceptTCP()
				if err != nil {
					done <- err
					return
				}
				err = server.SetDeadline(time.Now().Add(5 * time.Second))
				if err != nil {
					server.Close()
					done <- err
					return
				}
				done <- handleConnection(server)
			}()
			client, err := net.DialTCP("tcp4", nil, listener.Addr().(*net.TCPAddr))
			assert.NoError(t, err)
			if err != nil {
				return
			}
			defer client.Close()
			err = client.SetDeadline(time.Now().Add(5 * time.Second))
			assert.NoError(t, err)
			if err != nil {
				return
			}
			for i, chunk := range tc.chunks {
				if i > 0 {
					time.Sleep(5 * time.Millisecond)
				}
				n, err := client.Write(chunk)
				assert.NoError(t, err)
				assert.Equal(t, len(chunk), n)
				if err != nil {
					return
				}
			}
			responseLength := 0
			for _, expected := range tc.responses {
				responseLength += expected.length
			}
			received := make([]byte, responseLength)
			_, err = io.ReadFull(client, received)
			assert.NoError(t, err)
			if err != nil {
				return
			}
			err = client.CloseWrite()
			assert.NoError(t, err)
			extra, err := io.ReadAll(client)
			assert.NoError(t, err)
			assert.Empty(t, extra)
			select {
			case err = <-done:
				if tc.expectedError != nil {
					assert.ErrorIs(t, err, tc.expectedError)
				} else {
					assert.NoError(t, err)
				}
			case <-time.After(5 * time.Second):
				assert.Fail(t, "Connection handler did not finish")
				return
			}
			seenNumbers := make(map[uint16]bool)
			for _, expected := range tc.responses {
				if !assert.GreaterOrEqual(t, len(received), 11) {
					return
				}
				length := int(received[3]) + int(binary.LittleEndian.Uint16(received[5:7]))
				if binary.LittleEndian.Uint16(received[5:7]) > 0 {
					length += 2
				}
				assert.Equal(t, expected.length, length)
				if !assert.GreaterOrEqual(t, len(received), length) {
					return
				}
				response, err := packet.ReadPacketVersion(received[:length], packet.ServiceVersion2)
				assert.NoError(t, err)
				if err != nil {
					return
				}
				received = received[length:]
				assert.Equal(t, expected.packetType, response.PacketType)
				var serviceData packet.BytesData
				if expected.packetType == packet.EGTS_PT_RESPONSE {
					confirmation, ok := response.ServicesFrameData.(*packet.PTResponse)
					assert.True(t, ok)
					if !ok {
						return
					}
					assert.Equal(t, expected.confirmedPacket, confirmation.ResponsePacketID)
					assert.Equal(t, expected.processingResult, confirmation.ProcessingResult)
					serviceData = confirmation.SDR
				} else {
					serviceData = response.ServicesFrameData
				}
				if len(expected.services) == 0 {
					assert.Nil(t, serviceData)
					continue
				}
				records, ok := serviceData.(*packet.ServicesFrameData)
				assert.True(t, ok)
				if !ok {
					return
				}
				if !assert.Len(t, *records, len(expected.services)) {
					return
				}
				for i, service := range expected.services {
					record := (*records)[i]
					assert.False(t, seenNumbers[record.RecordNumber], "Repeated RN %d", record.RecordNumber)
					seenNumbers[record.RecordNumber] = true
					assert.Equal(t, service.serviceType, record.SourceServiceType)
					assert.Equal(t, service.serviceType, record.RecipientServiceType)
					if expected.packetType == packet.EGTS_PT_RESPONSE {
						if !assert.Len(t, record.RecordsData, len(service.confirmedRecords)) {
							return
						}
						for j, number := range service.confirmedRecords {
							assert.Equal(t, &subrecord.SRRecordResponse{ConfirmedRecordNumber: number}, record.RecordsData[j].SubrecordData)
						}
					} else {
						if !assert.Len(t, record.RecordsData, 1) {
							return
						}
						assert.Equal(t, &subrecord.SRResultCode{RCD: packet.EGTS_PC_OK}, record.RecordsData[0].SubrecordData)
					}
				}
			}
			assert.Empty(t, received)
		})
	}
}
