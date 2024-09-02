package main

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"

	"hazeparser/proto/citadel_gcmessages_common_go"
	"hazeparser/proto/citadel_usermessages_go"
	"hazeparser/proto/demoproto"

	"github.com/golang/snappy"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

type DemoParser struct {
	reader     *bufio.Reader
	outputFile *os.File
	buffer     []byte
	demoGuid   string
}

func NewDemoParser(filePath string, outputToFile bool) (*DemoParser, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}

	var outputFile *os.File
	if outputToFile {
		outputFile, err = os.OpenFile("output.txt", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			file.Close()
			return nil, fmt.Errorf("error opening output file: %w", err)
		}
	}

	return &DemoParser{
		reader:     bufio.NewReaderSize(file, DEMO_BUFFER_SIZE),
		outputFile: outputFile,
		buffer:     make([]byte, DEMO_BUFFER_SIZE),
	}, nil
}

func (dp *DemoParser) Close() {
	if dp.outputFile != nil {
		dp.outputFile.Close()
	}
}

func (dp *DemoParser) Parse() error {
	if _, err := dp.reader.Discard(VALVE_HEADER_SIZE); err != nil {
		return err
	}

	for {
		command, err := dp.readVarInt32()
		if err == io.EOF {
			fmt.Println("EOF file reached, parsing done.")
			return nil
		} else if err != nil {
			return fmt.Errorf("error reading command: %w", err)
		}

		tick, err := dp.readVarUInt32()
		if err != nil {
			return fmt.Errorf("error reading tick: %w", err)
		}

		size, err := dp.readVarInt32()
		if err != nil {
			return fmt.Errorf("error reading size: %w", err)
		}

		if tick == 0xFFFFFFFF {
			tick = 0
		}

		msgType := command &^ DEM_IsCompressed
		isCompressed := (command & DEM_IsCompressed) == DEM_IsCompressed

		if int(size) > len(dp.buffer) {
			dp.buffer = make([]byte, size)
		}
		messageData := dp.buffer[:size]
		_, err = io.ReadFull(dp.reader, messageData)
		if err != nil {
			return fmt.Errorf("error reading message data: %w", err)
		}

		if isCompressed {
			messageData, err = snappy.Decode(nil, messageData)
			if err != nil {
				return fmt.Errorf("error decompressing data: %w", err)
			}
		}

		output := fmt.Sprintf("Command Number: %d (msgType: %d), Tick Number: %d, Frame Size: %d, Compressed: %t\n",
			command, msgType, tick, size, isCompressed)
		if dp.outputFile != nil {
			if _, err := dp.outputFile.WriteString(output); err != nil {
				return fmt.Errorf("error writing to output file: %w", err)
			}
		}

		if msgType == 1 {
			if err := dp.handleDemoHeader(messageData); err != nil {
				return err
			}
		}

		if msgType == 7 || msgType == 8 {
			if err := dp.parseCDemoPacket(messageData); err != nil {
				continue
			}
		}
	}
}

func (dp *DemoParser) handleDemoHeader(messageData []byte) error {
	var demoHeader demoproto.CDemoFileHeader
	if err := proto.Unmarshal(messageData, &demoHeader); err != nil {
		return fmt.Errorf("failed to parse CDemoFileHeader: %w", err)
	}

	dp.demoGuid = demoHeader.GetDemoVersionGuid()

	header := fmt.Sprintf("DemoGuid: %s\nNetworkProtocol: %d\nServerName: %s\nClientName: %s\nMapName: %s\n",
		dp.demoGuid, demoHeader.GetNetworkProtocol(), demoHeader.GetServerName(), demoHeader.GetClientName(), demoHeader.GetMapName())
	if dp.outputFile != nil {
		if _, err := dp.outputFile.WriteString(header); err != nil {
			return fmt.Errorf("error writing header to output file: %w", err)
		}
	}

	return nil
}

func (dp *DemoParser) parseCDemoPacket(messageData []byte) error {
	var demoPacket demoproto.CDemoPacket
	if err := proto.Unmarshal(messageData, &demoPacket); err != nil {
		return fmt.Errorf("failed to parse CDemoPacket: %w", err)
	}

	packetData := demoPacket.GetData()
	bitReader := NewBitReader(packetData)

	for {
		ubit, err := bitReader.ReadUbit()
		if err == io.EOF {
			break // End of packet data reached
		}
		if err != nil {
			return fmt.Errorf("error reading Ubit: %w", err)
		}

		msgSize, err := bitReader.ReadVarInt32()
		if err != nil {
			if err == io.EOF {
				break // End of packet data reached
			}
			return fmt.Errorf("error reading message size: %w", err)
		}

		msgData, err := bitReader.ReadBytes(int(msgSize))
		if err != nil {
			if err == io.EOF {
				break // End of packet data reached
			}
			return fmt.Errorf("error reading message data: %w", err)
		}

		if dp.outputFile != nil {
			if _, err := dp.outputFile.WriteString(fmt.Sprintf("Ubit: %d, MsgSize: %d\n", ubit, msgSize)); err != nil {
				return fmt.Errorf("error writing to output file: %w", err)
			}
		}

		if ubit == 316 {
			if err := dp.handlePostMatchDetails(msgData); err != nil {
				return err
			}
		}
	}
	return nil
}

func (dp *DemoParser) handlePostMatchDetails(msgData []byte) error {
	var postMatch citadel_usermessages_go.CCitadelUserMsg_PostMatchDetails
	if err := proto.Unmarshal(msgData, &postMatch); err != nil {
		return err
	}

	data := postMatch.GetMatchDetails()
	var metadata citadel_gcmessages_common_go.CMsgMatchMetaDataContents
	if err := proto.Unmarshal(data, &metadata); err != nil {
		return err
	}

	jsonData, err := protojson.Marshal(&metadata)
	if err != nil {
		return err
	}

	// Write to file
	filename := fmt.Sprintf("match_data_%s.json", dp.demoGuid)
	if err := os.WriteFile(filename, jsonData, 0644); err != nil {
		return err
	}

	// Log to console
	fmt.Printf("JSON data written to %s\n", filename)

	return nil
}

func (dp *DemoParser) readVarInt32() (int32, error) {
	var result int32
	var shift uint
	for {
		b, err := dp.reader.ReadByte()
		if err != nil {
			return 0, err
		}
		result |= int32(b&0x7f) << shift
		if b&0x80 == 0 {
			break
		}
		shift += 7
		if shift >= 35 {
			return 0, errors.New("VarInt32 is too long")
		}
	}
	return result, nil
}

func (dp *DemoParser) readVarUInt32() (uint32, error) {
	var result uint32
	var shift uint
	for {
		b, err := dp.reader.ReadByte()
		if err != nil {
			return 0, err
		}
		result |= uint32(b&0x7f) << shift
		if b&0x80 == 0 {
			break
		}
		shift += 7
		if shift >= 35 {
			return 0, errors.New("VarUInt32 is too long")
		}
	}
	return result, nil
}
