package main

import (
	// Uncomment this line to pass the first stage
	// "encoding/json"
	"bytes"
	"crypto/sha1"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"strconv"
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

type TrackerResponse struct {
	// Interval string `bencode:"interval"`
	// Complete string `bencode:"complete"`
	// Incomplete string `bencode:"incomplete"`
	Peers string `bencode:"peers"`
}

func buildTorrent(filename string) (Torrent, error) {
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
	return torrent, nil
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

		torrent, _ := buildTorrent(filename)
		// handle error

		fmt.Printf("Tracker URL: %s\n", torrent.Announce)
		fmt.Printf("Length: %v\n", torrent.Length)
		fmt.Printf("Info Hash: %s\n", hex.EncodeToString(torrent.InfoHash[:]))
		fmt.Printf("Piece Length: %d\n", torrent.PieceLength)
		for i := 0; i < len(torrent.PieceHashes); i++ {
			fmt.Printf("%s\n", hex.EncodeToString(torrent.PieceHashes[i][:]))

		}
	case "peers":
		filename := os.Args[2]
		torrent, _ := buildTorrent(filename)

		req, _ := http.NewRequest("GET", torrent.Announce, nil)

		q := req.URL.Query()
		q.Add("info_hash", string(torrent.InfoHash[:]))
		// q.Add("peer_id", "06225127954136140002")
		q.Add("peer_id", "00112233445566778899")
		q.Add("port", "6881")
		q.Add("uploaded", "0")
		q.Add("downloaded", "0")
		q.Add("left", strconv.Itoa(torrent.Length))
		q.Add("compact", "1")
		req.URL.RawQuery = q.Encode()

		// fmt.Println("url: ", req.URL.String())
		response, err := http.Get(req.URL.String())
		if err != nil {
			fmt.Println("Error processing the get request")
			os.Exit(1)
		}

		responseData, _ := io.ReadAll(response.Body)

		// decoded, _ :=bencode.Decode(strings.NewReader(string(responseData)))
		var resp TrackerResponse
		bencode.Unmarshal(strings.NewReader(string(responseData)), &resp)
		peersBytes := []byte(resp.Peers)
		for i := 0; i < len(peersBytes); i += 6 {
			ip := net.IP(peersBytes[i : i+4])
			port := binary.BigEndian.Uint16(peersBytes[i+4 : i+6])
			fmt.Printf("%s:%d\n", ip, port)
		}

	case "handshake":
		filename := os.Args[2]
		peer := os.Args[3]
		torrent, _ := buildTorrent(filename)

		conn, err := net.Dial("tcp", peer)
		if err != nil {
			fmt.Println("Error connecting to peer")
			os.Exit(1)
		}
		var buf []byte
		buf = append(buf, 19)
		buf = append(buf, []byte("BitTorrent protocol")...)
		buf = append(buf, make([]byte, 8)...)
		buf = append(buf, torrent.InfoHash[:]...)
		buf = append(buf, []byte("00112233445566778899")...)
		_, err = conn.Write(buf)
		if err != nil {
			fmt.Println("Error on protocol handshake", err)
			os.Exit(1)
		}

		answer := make([]byte, 68)
		io.ReadFull(conn, answer)

		// Read last 20 bytes (peerID)
		fmt.Printf("Peer ID: %x\n", answer[48:])

	default:
		fmt.Println("Unknown command: " + command)
		os.Exit(1)
	}

}
