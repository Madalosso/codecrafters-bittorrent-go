package main

import (
	"bytes"
	"crypto/sha1"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	bencode "github.com/jackpal/bencode-go"
)

type TrackerResponse struct {
	// Interval string `bencode:"interval"`
	// Complete string `bencode:"complete"`
	// Incomplete string `bencode:"incomplete"`
	Peers string `bencode:"peers"`
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
		peersList := peersList(torrent)
		for _, val := range peersList {
			fmt.Printf("%s\n", val)
		}

	case "handshake":
		filename := os.Args[2]
		peer := os.Args[3]
		torrent, _ := buildTorrent(filename)

		peerConnection := newPeerConnection(peer, torrent.InfoHash[:])
		fmt.Printf("Peer ID: %x\n", peerConnection.peerID)

	case "download_piece":
		// -o usually represents file output destination
		fileDestination := os.Args[3]
		torrentFile := os.Args[4]
		pieceStr := os.Args[5]
		piece, _ := strconv.Atoi(pieceStr)

		torrent, _ := buildTorrent(torrentFile)
		peers := peersList(torrent)
		fmt.Println(fileDestination, torrentFile, piece)
		fmt.Println(torrent.Announce)
		// for _, peer := range peers
		for j:=0 ; j <len(peers); j++{
			peer := peers[j]
			fmt.Println("Trying to download piece ", piece, "from peer ", peer)
			peerConnection := newPeerConnection(peer, torrent.InfoHash[:])

			// fmt.Println("Waiting for bitfield")
			peerConnection.readMessage(5)

			var payload []byte
			// fmt.Println("Sending 'interested'")
			peerConnection.writeMessage(2, payload)

			// fmt.Println("Waiting for unchoke")
			peerConnection.readMessage(1)

			// fmt.Println("Waiting for request download")

			// Download piece
			// const blockSize int = 3 * 1024
			const blockSize int = 16 * 1024
			var pieceData []byte
			fmt.Println("Number of peers: ", len(peers))
			fmt.Printf("Download piece %d with size %d through %d blocks of %d length\n", piece, torrent.PieceLength, torrent.PieceLength/blockSize, blockSize)
			for i := 0; i < torrent.PieceLength; i += blockSize {
				requestPayload := make([]byte, 12) // 4 bytes for piece index, 4 bytes for offset, 4 bytes for length

				binary.BigEndian.PutUint32(requestPayload[0:], uint32(piece))
				binary.BigEndian.PutUint32(requestPayload[4:], uint32(i))
				if i+blockSize <= torrent.PieceLength {
					fmt.Println("requesting data from ", i, " to ", i+blockSize)
					binary.BigEndian.PutUint32(requestPayload[8:], uint32(blockSize))
				} else {
					fmt.Printf("Last piece, length: %v - %v : %v\n", torrent.PieceLength, i, torrent.PieceLength-i)
					binary.BigEndian.PutUint32(requestPayload[8:], uint32(torrent.PieceLength-i))
				}

				peerConnection.writeMessage(6, requestPayload)

				//wait for msg id 7 (Piece)
				// fmt.Println("Waiting for msg with data")

				msg, err := peerConnection.readMessage(7)
				if err != nil {
					if err == io.EOF {
						fmt.Println("Connection was closed? Trying to reconnect and keep downloading")
						//try to reconnect?
						peerConnection.conn.Close()
						if j+1 < len(peers){
							j++
							fmt.Println("Selecting next peer: ", peers[j])
						} else {
							fmt.Println("No other peer available, closing...")
							os.Exit(1)
						}
						peer = peers[j]

						peerConnection = newPeerConnection(peer, torrent.InfoHash[:])

						// fmt.Println("Waiting for bitfield")
						peerConnection.readMessage(5)

						var payload []byte
						// fmt.Println("Sending 'interested'")
						peerConnection.writeMessage(2, payload)

						// fmt.Println("Waiting for unchoke")
						peerConnection.readMessage(1)
						// os.Exit(1)
					}
					// try again
					i -= blockSize
					continue
				}

				index := make([]byte, 4)
				begin := make([]byte, 4)
				copy(index, msg.data[0:4])
				copy(begin, msg.data[4:8])
				fmt.Printf("index: %d; begin: %d\n", binary.BigEndian.Uint32(index), binary.BigEndian.Uint32(begin))
				data := msg.data[8:]
				// fmt.Println("Data: ", data)
				pieceData = append(pieceData, data...)
				fmt.Println("Length of downloaded piece data so far: ", len(pieceData), "/", torrent.PieceLength)

			}
			peerConnection.conn.Close()
			// fmt.Println("Total length of downloaded piece data: ", len(pieceData))

			pieceHash := torrent.PieceHashes[piece]
			hash := sha1.Sum(pieceData)
			// fmt.Println("hashTarget: ", pieceHash)
			// fmt.Println("Hash from data: ", hash)
			if check := bytes.Equal(pieceHash[:], hash[:]); check {
				// fmt.Println("Checksum correct! breaking out of the loop")
				// write to disk
				file, err := os.Create(fileDestination)
				if err != nil {
					fmt.Println(err)
					return
				}
				defer file.Close()
				n, err := file.Write(pieceData)
				if err != nil {
					fmt.Println(err)
					return
				}
				fmt.Printf("%d bytes written to %s.\n", n, fileDestination)
				fmt.Printf("Piece %d downloaded to %s.\n", piece, fileDestination)
				os.Exit(0)
				break
			}
			fmt.Println("Checksum failed, testing fetch data from another peer")
		}

	default:
		fmt.Println("Unknown command: " + command)
		os.Exit(1)
	}

}
