package main

import (
	// Uncomment this line to pass the first stage
	// "encoding/json"
	"bytes"
	"crypto/sha1"
	"encoding/hex"
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
	PieceLength int `bencode:"piece length"`
	Pieces string `bencode:"pieces"`
}

type TorrentMeta struct {
	Announce string `bencode:"announce"`
	Info TorrentMetaInfo `bencode:"info"`
}

type Torrent struct {
	Announce string
	Name string
	Length int
	InfoHash [20]byte
	PieceLength int
	PieceHashes [][20]byte
}

func (tr *TorrentMeta) toTorrent() Torrent {
	infoHash := tr.Info.hash()
	pieceHashes := tr.Info.pieceHashes()

	return Torrent {
		Announce: tr.Announce,
		Name: tr.Info.Name,
		Length: tr.Info.Length,
		InfoHash: infoHash,
		PieceHashes: pieceHashes,
		PieceLength: tr.Info.PieceLength,
	}
}

func (meta *TorrentMetaInfo) hash() [20]byte {
	fmt.Println(meta)
	var buf bytes.Buffer
	bencode.Marshal(&buf, *meta)
	h := sha1.Sum(buf.Bytes())
	return h
}

func (meta *TorrentMetaInfo) pieceHashes() [][20]byte {
	hashLen :=20
	buf := []byte(meta.Pieces)

	numHashes := len(buf) / hashLen
	hashes:= make([][20]byte, numHashes)

	for i:=0;i<numHashes;i++{
		copy(hashes[i][:], buf[i*hashLen:(i+1)*hashLen])
	}
	return hashes
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

		torrent:= meta.toTorrent()

		// fmt.Printf("Tracker URL: %s", torrent.Announce)
		// fmt.Printf("Length: %v", torrent.Length)
		fmt.Printf("Info Hash: %s", hex.EncodeToString(torrent.InfoHash[:]))

	default:
		fmt.Println("Unknown command: " + command)
		os.Exit(1)
	}

}
