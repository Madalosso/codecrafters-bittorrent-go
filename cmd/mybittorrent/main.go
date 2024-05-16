package main

import (
	// Uncomment this line to pass the first stage
	// "encoding/json"
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
	"unicode"

	bencode "github.com/jackpal/bencode-go"
)

func decodeBencode(bencodedString string) (interface{}, error) {
	if unicode.IsDigit(rune(bencodedString[0])) {
		var firstColonIndex int

		for i := 0; i < len(bencodedString); i++ {
			if bencodedString[i] == ':' {
				firstColonIndex = i
				break
			}
		}

		lengthStr := bencodedString[:firstColonIndex]

		length, err := strconv.Atoi(lengthStr)
		if err != nil {
			return "", err
		}

		return bencodedString[firstColonIndex+1 : firstColonIndex+1+length], nil
	} else {
		return "", fmt.Errorf("Only strings are supported at the moment")
	}
}

type TorrentMetaInfo struct {
	Length int `bencode:"length"`
	Name string `benconde:"name"`
	// PieceLength int `bencode:"piece length"`
	// Pieces string `bencode:"pieces"`
}

type TorrentMeta struct {
	Announce string `bencode:"announce"`
	Info TorrentMetaInfo `bencode:"info"`
}

func main() {

	command := os.Args[1]

	switch command {
	case "decode":
		bencodedValue := os.Args[2]
		decoded, err := bencode.Decode(strings.NewReader(bencodedValue))
		// decoded, err := decodeBencode(bencodedValue)
		if err != nil {
			fmt.Println(err)
			return
		}

		jsonOutput, _ := json.Marshal(decoded)
		fmt.Println(string(jsonOutput))

	case "info":
		filename := os.Args[2]
		content, err := os.ReadFile(filename)
		if err != nil {
			fmt.Println("Error reading file content:", err)
			os.Exit(1)
		}
		reader := bytes.NewReader(content)
		var meta TorrentMeta
		err = bencode.Unmarshal(reader, &meta)
		if err != nil {
			fmt.Println("Error decoding file content:", err)
			os.Exit(1)
		}

		fmt.Printf("Tracker URL: %s", meta.Announce)
		fmt.Printf("Length: %v", meta.Info.Length)

	default:
		fmt.Println("Unknown command: " + command)
		os.Exit(1)
	}

}
