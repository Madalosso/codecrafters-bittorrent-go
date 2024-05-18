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
	"strings"

	bencode "github.com/jackpal/bencode-go"
)

type TorrentMetaInfo struct {
	Length      int    `bencode:"length"`
	Name        string `bencode:"name"`
	PieceLength int    `bencode:"piece length"`
	Pieces      string `bencode:"pieces"`
}
type TorrentFile struct {
	Announce  string          `bencode:"announce"`
	CreatedBy string          `bencode:"created by"`
	Info      TorrentMetaInfo `bencode:"info"`
}

type Torrent struct {
	Announce    string
	Name        string
	Length      int
	InfoHash    [20]byte
	PieceLength int
	PieceHashes [][20]byte
}

func (tr *TorrentFile) toTorrent() Torrent {
	infoHash := tr.Info.hash()
	pieceHashes := tr.Info.pieceHashes()

	return Torrent{
		Announce:    tr.Announce,
		Name:        tr.Info.Name,
		Length:      tr.Info.Length,
		InfoHash:    infoHash,
		PieceHashes: pieceHashes,
		PieceLength: tr.Info.PieceLength,
	}
}

func (meta *TorrentMetaInfo) hash() [20]byte {
	sha := sha1.New()
	bencode.Marshal(sha, *meta)
	h := sha.Sum(nil)
	// fmt.Println(h)
	var asd [20]byte
	copy(asd[:], h[:20])
	// fmt.Println(asd)
	return asd
}

func (meta *TorrentMetaInfo) pieceHashes() [][20]byte {
	hashLen := 20
	buf := []byte(meta.Pieces)

	numHashes := len(buf) / hashLen
	hashes := make([][20]byte, numHashes)

	for i := 0; i < numHashes; i++ {
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
		var meta TorrentFile
		err = bencode.Unmarshal(reader, &meta)
		if err != nil {
			fmt.Println("Error decoding file content:", err)
			os.Exit(1)
		}
		torrent := meta.toTorrent()

		fmt.Printf("Tracker URL: %s\n", torrent.Announce)
		fmt.Printf("Length: %v\n", torrent.Length)
		fmt.Printf("Info Hash: %s\n", hex.EncodeToString(torrent.InfoHash[:]))
		fmt.Printf("Piece Length: %d\n", torrent.PieceLength)
		for i:=0; i< len(torrent.PieceHashes); i++{
			fmt.Printf("%s\n", hex.EncodeToString(torrent.PieceHashes[i][:]))

		}

	default:
		fmt.Println("Unknown command: " + command)
		os.Exit(1)
	}

}
