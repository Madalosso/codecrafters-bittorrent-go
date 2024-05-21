package main

import (
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"

	bencode "github.com/jackpal/bencode-go"
)

func peersList(torrent Torrent) []string {

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

	var peers []string

	for i := 0; i < len(peersBytes); i += 6 {
		ip := net.IP(peersBytes[i : i+4])
		port := binary.BigEndian.Uint16(peersBytes[i+4 : i+6])
		peers = append(peers, fmt.Sprintf("%s:%d", ip, port))
	}
	return peers
}

type PeerConnection struct {
	conn     net.Conn
	peerID   [20]byte
	clientID []byte
}

type PeerMessage struct {
	lengthPrefix uint32
	id           uint8
	data         []byte
}

func (p *PeerConnection) readMessage(expectedMsgId byte) (PeerMessage, error) {
	// Read message length (4 bytes)
	lengthBuf := make([]byte, 4)
	n, err := io.ReadAtLeast(p.conn, lengthBuf, 4)
	fmt.Printf("readMessage: Read %d bytes \n", n)

	// _, err := io.ReadFull(p.conn, lengthBuf)
	if err != nil {
		fmt.Println("Error reading message length:", err)
		return PeerMessage{}, err
		// os.Exit(1)
	}
	length := binary.BigEndian.Uint32(lengthBuf)
	fmt.Println("msg length: ", length)
	// Read message ID (1 byte)
	payload := make([]byte, length)
	_, err = io.ReadFull(p.conn, payload)
	if err != nil {
		fmt.Println("Error reading message ID:", err)
		os.Exit(1)
	}
	msgId := payload[0]

	fmt.Println("msgId: ", msgId)
	// fmt.Println("payload: ", payload[1:])

	// If message ID is 5 (bitfield), break the loop
	if msgId != expectedMsgId {
		return PeerMessage{}, fmt.Errorf("Unexpected msg id")
	}

	msg := PeerMessage{
		lengthPrefix: length,
		id:           msgId,
		data:         payload[1:],
	}
	// fmt.Println("Msg data: ", msg.data)

	return msg, nil
}

// msgId -> enum
func (p *PeerConnection) writeMessage(msgId byte, payload []byte) {
	length := uint32(len(payload) + 1)
	msg := make([]byte, 4+length)

	// fill first 4 bytes with msg length
	binary.BigEndian.PutUint32(msg, length)

	msg[4] = msgId
	copy(msg[5:], payload)

	// Send the message over the connection
	_, err := p.conn.Write(msg)
	if err != nil {
		fmt.Println("Error writing message:", err)
		os.Exit(1)
	}
}

func newPeerConnection(peer string, infoHash []byte) PeerConnection {
	conn, err := net.Dial("tcp", peer)
	if err != nil {
		fmt.Println("Error connecting to peer")
		os.Exit(1)
	}

	clientId := []byte("00112233445566778899")
	// Protocol handshake msg
	var buf []byte
	buf = append(buf, 19)
	buf = append(buf, []byte("BitTorrent protocol")...)
	buf = append(buf, make([]byte, 8)...)
	buf = append(buf, infoHash...)
	buf = append(buf, clientId...)

	_, err = conn.Write(buf)
	if err != nil {
		fmt.Println("Error on protocol handshake", err)
		os.Exit(1)
	}

	// Read handshake reply
	answer := make([]byte, 68)
	io.ReadFull(conn, answer)

	// Read last 20 bytes (peerID)

	fmt.Printf("Connection established with Peer ID: %x\n", answer[48:])

	return PeerConnection{
		conn:     conn,
		peerID:   [20]byte(answer[48:]),
		clientID: clientId,
	}
}
