package main

import (
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"io"
	"net"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/LdDl/go-egts/crc"
	"github.com/LdDl/go-egts/egts/packet"
	"github.com/LdDl/go-egts/egts/subrecord"
	"github.com/stretchr/testify/assert"
)

type serverServiceResponseTestCase struct {
	confirmedRecords []uint16
	serviceType      uint8
}

type serverResponseTestCase struct {
	length           int
	packetType       uint8
	confirmedPacket  uint16
	processingResult uint8
	services         []serverServiceResponseTestCase
}

type serverConnectionTestCase struct {
	name          string
	chunks        [][]byte
	responses     []serverResponseTestCase
	expectedError error
}

type serverCounterTestCase struct {
	name    string
	counter *uint32
	next    func() uint16
}

type serverCounterBoundaryTestCase struct {
	start  uint32
	first  uint16
	second uint16
}

func TestHandleConnection(t *testing.T) {
	telemetry, err := hex.DecodeString("0100000b002300000001991800000001ef0000000202101500d2312b104fba3a9ed227bc35030000b200000000006a8d")
	assert.NoError(t, err)
	if err != nil {
		return
	}
	identity, err := hex.DecodeString("0100020b0020000000014f1900000010010101160000000000523836363130343032393639303030380004417f")
	assert.NoError(t, err)
	if err != nil {
		return
	}
	confirmation := packet.Packet{
		ProtocolVersion:   1,
		PRF:               "00",
		RTE:               "0",
		ENA:               "00",
		CMP:               "0",
		PR:                "00",
		HeaderLength:      11,
		FrameDataLength:   3,
		PacketID:          7,
		PacketType:        packet.EGTS_PT_RESPONSE,
		ServicesFrameData: &packet.PTResponse{ResponsePacketID: 1, ProcessingResult: packet.EGTS_PC_OK},
	}
	confirmationBytes, err := confirmation.Encode()
	assert.NoError(t, err)
	if err != nil {
		return
	}
	empty := []byte{1, 0, 0, 11, 0, 0, 0, 7, 0, 1, 0}
	empty[10] = byte(crc.Crc(8, empty[:10]))
	corrupted := append([]byte(nil), telemetry...)
	corrupted[len(corrupted)-1] ^= 1
	badHeader := append([]byte(nil), telemetry...)
	badHeader[10] ^= 1
	badData := append([]byte(nil), telemetry...)
	badData[23] = 0
	badData[24] = 0
	binary.LittleEndian.PutUint16(badData[len(badData)-2:], uint16(crc.Crc(16, badData[11:len(badData)-2])))
	mixed, err := packet.ReadPacket(identity)
	assert.NoError(t, err)
	if err != nil {
		return
	}
	position, err := packet.ReadPacket(telemetry)
	assert.NoError(t, err)
	if err != nil {
		return
	}
	mixedServices := mixed.ServicesFrameData.(*packet.ServicesFrameData)
	*mixedServices = append(*mixedServices, (*position.ServicesFrameData.(*packet.ServicesFrameData))[0])
	(*mixedServices)[0].RecordNumber = 11
	(*mixedServices)[1].RecordNumber = 22
	mixed.FrameDataLength = mixedServices.Len()
	mixedBytes, err := mixed.Encode()
	assert.NoError(t, err)
	if err != nil {
		return
	}
	telemetryResponse := serverResponseTestCase{
		length:          29,
		packetType:      packet.EGTS_PT_RESPONSE,
		confirmedPacket: 0,
		services:        []serverServiceResponseTestCase{{confirmedRecords: []uint16{0}, serviceType: packet.SERVICE_DATA}},
	}
	cases := []serverConnectionTestCase{
		{
			name:   "mixed services then next packet",
			chunks: [][]byte{append(mixedBytes, telemetry...)},
			responses: []serverResponseTestCase{
				{
					length:     42,
					packetType: packet.EGTS_PT_RESPONSE,
					services: []serverServiceResponseTestCase{
						{serviceType: packet.SERVICE_AUTH, confirmedRecords: []uint16{11}},
						{serviceType: packet.SERVICE_DATA, confirmedRecords: []uint16{22}},
					},
				},
				{length: 24, packetType: packet.EGTS_PT_APPDATA, services: []serverServiceResponseTestCase{{serviceType: packet.SERVICE_AUTH}}},
				telemetryResponse,
			},
		},
		{
			name:   "valid packet after bad service data",
			chunks: [][]byte{append(badData, telemetry...)},
			responses: []serverResponseTestCase{
				{length: 16, packetType: packet.EGTS_PT_RESPONSE, processingResult: packet.EGTS_PC_INC_DATAFORM},
				telemetryResponse,
			},
		},
		{
			name:      "valid packet after bad header checksum",
			chunks:    [][]byte{append(badHeader, telemetry...)},
			responses: []serverResponseTestCase{telemetryResponse},
		},
		{
			name:      "telemetry without authorization",
			chunks:    [][]byte{telemetry},
			responses: []serverResponseTestCase{telemetryResponse},
		},
		{
			name:      "two packets in one write",
			chunks:    [][]byte{append(append([]byte(nil), telemetry...), telemetry...)},
			responses: []serverResponseTestCase{telemetryResponse, telemetryResponse},
		},
		{
			name: "fragmented header and body",
			chunks: [][]byte{
				telemetry[:1], telemetry[1:5], telemetry[5:10], telemetry[10:11],
				telemetry[11 : len(telemetry)-2], telemetry[len(telemetry)-2:],
			},
			responses: []serverResponseTestCase{telemetryResponse},
		},
		{
			name:   "transport confirmation",
			chunks: [][]byte{confirmationBytes},
		},
		{
			name:      "telemetry after transport confirmation",
			chunks:    [][]byte{append(append([]byte(nil), confirmationBytes...), telemetry...)},
			responses: []serverResponseTestCase{telemetryResponse},
		},
		{
			name:   "identity then telemetry",
			chunks: [][]byte{append(append([]byte(nil), identity...), telemetry...)},
			responses: []serverResponseTestCase{
				{length: 29, packetType: packet.EGTS_PT_RESPONSE, services: []serverServiceResponseTestCase{{confirmedRecords: []uint16{0}, serviceType: packet.SERVICE_AUTH}}},
				{length: 24, packetType: packet.EGTS_PT_APPDATA, services: []serverServiceResponseTestCase{{serviceType: packet.SERVICE_AUTH}}},
				telemetryResponse,
			},
		},
		{
			name:      "empty appdata",
			chunks:    [][]byte{empty},
			responses: []serverResponseTestCase{{length: 16, packetType: packet.EGTS_PT_RESPONSE, confirmedPacket: 7}},
		},
		{
			name:   "valid packet after bad checksum",
			chunks: [][]byte{append(corrupted, telemetry...)},
			responses: []serverResponseTestCase{
				{length: 16, packetType: packet.EGTS_PT_RESPONSE, processingResult: packet.EGTS_PC_DATACRC_ERROR},
				telemetryResponse,
			},
		},
		{
			name:          "wrong protocol",
			chunks:        [][]byte{{2, 0, 0, 11, 0, 0, 0, 0, 0, 1}},
			expectedError: ErrNoEGTSPacket,
		},
	}
	for length := 0; length < 10; length++ {
		tc := serverConnectionTestCase{
			name:   fmt.Sprintf("header interrupted at %d", length),
			chunks: [][]byte{telemetry[:length]},
		}
		if length > 0 {
			tc.expectedError = ErrAcceptEGTSPacket
		}
		cases = append(cases, tc)
	}
	for _, length := range []int{10, 11, 20, len(telemetry) - 1} {
		cases = append(cases, serverConnectionTestCase{
			name:          fmt.Sprintf("body interrupted at %d", length),
			chunks:        [][]byte{telemetry[:length]},
			expectedError: ErrAcceptEGTSPacket,
		})
	}
	for _, length := range []byte{0, 9, 10, 12, 16, 255} {
		cases = append(cases, serverConnectionTestCase{
			name:          fmt.Sprintf("invalid header length %d", length),
			chunks:        [][]byte{{1, 0, 0, length, 0, 0, 0, 0, 0, 1}},
			expectedError: ErrAcceptEGTSPacket,
		})
	}
	cases = append(cases, serverConnectionTestCase{
		name:          "route flag without route fields",
		chunks:        [][]byte{{1, 0, 0x20, 11, 0, 0, 0, 0, 0, 1}},
		expectedError: ErrAcceptEGTSPacket,
	})
	for _, length := range []uint16{65523, 65535} {
		header := []byte{1, 0, 0, 11, 0, 0, 0, 0, 0, 1}
		binary.LittleEndian.PutUint16(header[5:7], length)
		cases = append(cases, serverConnectionTestCase{
			name:          fmt.Sprintf("packet length overflow %d", length),
			chunks:        [][]byte{header},
			expectedError: ErrAcceptEGTSPacket,
		})
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
				response, err := packet.ReadPacket(received[:length])
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

func TestPacketCounters(t *testing.T) {
	counters := []serverCounterTestCase{
		{name: "PID", counter: &pidCounter, next: getNextPid},
		{name: "RN", counter: &rnCounter, next: getNextRN},
	}
	boundaries := []serverCounterBoundaryTestCase{
		{start: 0, first: 1, second: 2},
		{start: 65534, first: 65535, second: 0},
		{start: 65535, first: 0, second: 1},
		{start: 0xfffffffe, first: 65535, second: 0},
		{start: 0xffffffff, first: 0, second: 1},
	}
	for _, tc := range counters {
		t.Run(tc.name, func(t *testing.T) {
			previous := atomic.LoadUint32(tc.counter)
			defer atomic.StoreUint32(tc.counter, previous)
			for _, boundary := range boundaries {
				atomic.StoreUint32(tc.counter, boundary.start)
				assert.Equal(t, boundary.first, tc.next())
				assert.Equal(t, boundary.second, tc.next())
			}
			atomic.StoreUint32(tc.counter, 65530)
			values := make(chan uint16, 1024)
			var workers sync.WaitGroup
			for i := 0; i < 16; i++ {
				workers.Add(1)
				go func() {
					defer workers.Done()
					for j := 0; j < 64; j++ {
						values <- tc.next()
					}
				}()
			}
			workers.Wait()
			close(values)
			var received []uint16
			for value := range values {
				received = append(received, value)
			}
			expected := make([]uint16, 1024)
			for i := range expected {
				expected[i] = uint16(65531 + i)
			}
			assert.ElementsMatch(t, expected, received)
		})
	}
}
