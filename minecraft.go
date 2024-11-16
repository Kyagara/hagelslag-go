package main

import (
	"context"
	"fmt"
	"io"
	"net"
	"time"
	"unsafe"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type Minecraft struct{}

func (s Minecraft) Name() string {
	return "minecraft"
}

func (s Minecraft) Network() string {
	return "tcp"
}

func (s Minecraft) Port() string {
	return "25565"
}

func (s Minecraft) Scan(ip string, conn net.Conn) ([]byte, int64, error) {
	// Handshake
	hostLen := len(ip)
	packetLen := 7 + hostLen

	request := make([]byte, packetLen+1)

	request[0] = byte(packetLen)
	request[2] = 0xFF
	request[3] = 1
	request[4] = byte(hostLen)

	i := 5
	copy(request[i:], ip)
	i += hostLen

	request[i] = byte(255)
	i++
	request[i] = byte(65)
	i++

	request[i] = 1

	n, err := conn.Write(request)
	if n == 0 || err != nil {
		return nil, 0, err
	}

	start := time.Now()

	// Status request
	request = []byte{0x1, 0x0}
	_, err = conn.Write(request)
	if err != nil {
		return nil, 0, err
	}

	// Read Status response
	packetLen, err = s.readVarInt(conn)
	if err != nil {
		return nil, 0, err
	}

	latency := time.Since(start).Milliseconds()

	if packetLen <= 0 {
		return nil, 0, nil
	}

	packetID, err := s.readByte(conn)
	if err != nil {
		return nil, 0, err
	}

	if packetID != 0x0 {
		return nil, 0, fmt.Errorf("unexpected packet ID: %d", packetID)
	}

	jsonLen, err := s.readVarInt(conn)
	if err != nil {
		return nil, 0, err
	}

	if jsonLen <= 0 {
		return nil, 0, nil
	}

	if jsonLen > MAX_RESPONSE_LENGTH {
		jsonLen = MAX_RESPONSE_LENGTH
	}

	response := make([]byte, jsonLen)
	_, err = io.ReadFull(conn, response)
	if err != nil {
		return nil, 0, err
	}

	return response, latency, nil
}

func (s Minecraft) Save(ip string, latency int64, data []byte, collection *mongo.Collection) error {
	document := bson.M{
		"_id":     ip,
		"latency": latency,
	}

	var result bson.D
	err := bson.UnmarshalExtJSON(data, true, &result)
	if err != nil {
		// If the data is not valid JSON, just save it as a string
		document["data"] = *(*string)(unsafe.Pointer(&data))
	}

	filter := bson.M{"_id": ip}
	opts := options.Replace().SetUpsert(true)

	_, err = collection.ReplaceOne(context.TODO(), filter, document, opts)
	if err != nil {
		return fmt.Errorf("failed to insert document '%s': %s", ip, err)
	}

	return nil
}

func (s Minecraft) readByte(r io.Reader) (byte, error) {
	b := []byte{0xff}

	_, err := io.ReadFull(r, b)
	if err != nil {
		return 0, err
	}

	return b[0], nil
}

func (s Minecraft) readVarInt(r io.Reader) (int, error) {
	var result int
	var position uint

	for {
		val, err := s.readByte(r)
		if err != nil {
			return 0, err
		}

		result |= int(val&0x7F) << position
		if val&0x80 == 0 {
			break
		}

		position += 7
		if position >= 64 {
			return 0, fmt.Errorf("varint too large")
		}
	}

	return result, nil
}
