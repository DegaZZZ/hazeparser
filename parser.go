package main

import (
	"encoding/binary"
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/DegaZZZ/hazeparser/proto/citadel_gcmessages_common_go"
	"github.com/DegaZZZ/hazeparser/proto/citadel_usermessages_go"
	"github.com/DegaZZZ/hazeparser/proto/demoproto"
	"github.com/golang/snappy"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

const (
	DEM_IsCompressed = 0x40 // Replace with actual value if different
)

var (
	header_ouput = false //THE MOST TEMPORARY SOLUTION, I PROMISE
	demo_stamp   = ""    //THE MOST TEMPORARY SOLUTION, I PROMISE PART 2
	quick_mode   = false //THE MOST TEMPORARY SOLUTION, I PROMISE PART 3
)

func main() {
	// Define flags for the input file path and quick mode
	filePath := flag.String("f", "", "Path to the demo file")
	flag.BoolVar(&quick_mode, "qm", false, "Toggle quick mode on or off")
	flag.Parse()

	// Check if the file path is provided
	if *filePath == "" {
		fmt.Println("Please provide a path to the demo file using the -file flag.")
		return
	}

	// Call the parseDEMFile function with the provided file path
	err := parseDEMFile(*filePath)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
	}
}

func readVarUInt32(r io.Reader) (uint32, error) {
	var result uint32
	var shift uint
	for {
		var byteVal byte
		if err := binary.Read(r, binary.LittleEndian, &byteVal); err != nil {
			return 0, err
		}
		result |= uint32(byteVal&0x7f) << shift
		if byteVal&0x80 == 0 {
			break
		}
		shift += 7
	}
	return result, nil
}

func readVarInt32(r io.Reader) (int32, error) {
	var result int32
	var shift uint
	for {
		var byteVal byte
		if err := binary.Read(r, binary.LittleEndian, &byteVal); err != nil {
			return 0, err
		}
		result |= int32(byteVal&0x7f) << shift
		if byteVal&0x80 == 0 {
			break
		}
		shift += 7
	}
	return result, nil
}

func parseDEMFile(filePath string) error {
	file, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	// Skip the first 16 bytes (Valve file header)
	if _, err := file.Seek(16, io.SeekStart); err != nil {
		return err
	}

	for {
		command, err := readVarInt32(file)
		if err == io.EOF {
			fmt.Println("EOF file reached, parsing done.")
			fmt.Println("Output written to file successfully.")
			return nil
		} else if err != nil {
			fmt.Println("Output written to file successfully.")
			return fmt.Errorf("error reading command: %w", err)
		}

		tick, err := readVarUInt32(file)
		if err != nil {
			return fmt.Errorf("error reading tick: %w", err)
		}

		size, err := readVarInt32(file)
		if err != nil {
			return fmt.Errorf("error reading size: %w", err)
		}

		// Handle tick value
		if tick == 0xFFFFFFFF {
			tick = 0
		}

		// Determine message type and compression flag
		msgType := command &^ DEM_IsCompressed
		isCompressed := (command & DEM_IsCompressed) == DEM_IsCompressed

		// Read the message based on the Frame Size
		messageData := make([]byte, size)
		_, err = io.ReadFull(file, messageData)
		if err != nil {
			return fmt.Errorf("error reading message data: %w", err)
		}

		// If the message is compressed, decompress it
		if isCompressed {
			messageData, err = snappy.Decode(nil, messageData)
			if err != nil {
				return fmt.Errorf("error decompressing data: %w", err)
			}
		}

		// Open or create the file
		file, err := os.OpenFile("output.txt", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			fmt.Println("Error opening file:", err)
		}
		defer file.Close()

		// Create the formatted string
		output := fmt.Sprintf("Command Number: %d (msgType: %d), Tick Number: %d, Frame Size: %d, Compressed: %t\n",
			command, msgType, tick, size, isCompressed)

		// Write the string to the file
		if _, err := file.WriteString(output); err != nil {
			fmt.Println("Error writing to file:", err)
		}

		var header = ""

		if msgType == 1 {
			var demoHeader demoproto.CDemoFileHeader
			err := proto.Unmarshal(messageData, &demoHeader)
			if err != nil {
				return fmt.Errorf("failed to parse CDemoFileHeader: %w", err)
			}
			//print the header
			// Create the formatted string
			header_ouput = true
			header = fmt.Sprintf("DemoFileStump: %d \nNetworkProtocol: %d \nServerName: %s \nClientName: %s \nMapName: %s", demoHeader.DemoFileStamp, *demoHeader.NetworkProtocol, *demoHeader.ServerName, *demoHeader.ClientName, *demoHeader.MapName)
			demo_stamp = fmt.Sprintf("%d", demoHeader.DemoFileStamp)
		}

		// Handle CDemoPacket messages
		if msgType == 7 || msgType == 8 {
			err = parseCDemoPacket(messageData)

			if err != nil {
				continue
			}
		}

		if header_ouput {
			// Write the string to the file
			if _, err := file.WriteString(header); err != nil {
				fmt.Println("Error writing to file:", err)
			}
			header_ouput = false
		}

	}

}

func parseCDemoPacket(messageData []byte) error {
	var demoPacket demoproto.CDemoPacket
	err := proto.Unmarshal(messageData, &demoPacket)
	if err != nil {
		return fmt.Errorf("")
	}

	packetData := demoPacket.GetData()
	bitReader := NewBitReader(packetData)

	for {
		ubit, err := bitReader.ReadUbit()
		if err != nil {
			return err
		}

		msgSize, err := bitReader.ReadVarInt32()
		if err != nil {
			return err
		}

		msgData, err := bitReader.ReadBytes(int(msgSize))
		if err != nil {
			return err
		}

		if !quick_mode {
			// Open or create the file
			file, err := os.OpenFile("output.txt", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
			if err != nil {
				fmt.Println("Error opening file:", err)
			}
			defer file.Close()

			// Write the string to the file
			if _, err := file.WriteString(fmt.Sprintf("Ubit: %d, MsgSize: %d\n", ubit, msgSize)); err != nil {
				fmt.Println("Error writing to file:", err)
			}
		}

		// Handle the message based on the Ubit value
		if ubit == 316 {

			var postMatch citadel_usermessages_go.CCitadelUserMsg_PostMatchDetails //citadel_usermessages.CCitadelUserMsg_PostMatchDetails
			err := proto.Unmarshal(msgData, &postMatch)
			if err != nil {
				return err
			}

			data := postMatch.GetMatchDetails()
			var metadata citadel_gcmessages_common_go.CMsgMatchMetaDataContents
			err = proto.Unmarshal(data, &metadata)
			if err != nil {
				return err
			}

			// Convert the Protobuf message to JSON and save it
			jsonData, err := protojson.Marshal(&metadata)
			if err != nil {
				return err
			}

			err = os.WriteFile(fmt.Sprintf("match_data_%s.json", demo_stamp), jsonData, 0644)
			if err != nil {
				return err
			}
		}
	}

}
