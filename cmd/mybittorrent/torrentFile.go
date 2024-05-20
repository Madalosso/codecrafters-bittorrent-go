package main

import (
	"bytes"
	"crypto/sha1"
	"fmt"
	"os"

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
	var asd [20]byte
	copy(asd[:], h[:20])
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
