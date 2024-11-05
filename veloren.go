package main

import (
	"context"
	"encoding/binary"
	"net"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type Veloren struct{}

func (s Veloren) Name() string {
	return "veloren"
}

func (s Veloren) Network() string {
	return "udp"
}

func (s Veloren) Port() string {
	return "14006"
}

func (s Veloren) Scan(ip string, conn net.Conn) ([]byte, int64, error) {
	request := make([]byte, 263)
	request[13] = 1
	header := []byte{'v', 'e', 'l', 'o', 'r', 'e', 'n'}
	for i, v := range header {
		request[256+i] = v
	}

	start := time.Now()

	// Init request
	_, err := conn.Write(request)
	if err != nil {
		return nil, 0, err
	}

	response := make([]byte, 14)
	_, err = conn.Read(response)
	if err != nil {
		return nil, 0, err
	}

	latency := time.Since(start).Milliseconds()

	p := binary.LittleEndian.Uint64(response[4:12])

	// version, not used for now
	// v := binary.BigEndian.Uint16(response[12:])

	// Set 'P' from response to the request
	binary.LittleEndian.PutUint64(request[2:12], p)

	// Change request to server info
	request[13] = 2

	// Server info request
	_, err = conn.Write(request)
	if err != nil {
		return nil, 0, err
	}

	response = make([]byte, 32)
	_, err = conn.Read(response)
	if err != nil {
		return nil, 0, err
	}

	return response, latency, nil
}

func (s Veloren) Save(ip string, latency int64, data []byte, collection *mongo.Collection) error {
	type serverInfo struct {
		Hash       uint32
		Timestamp  uint64
		Players    uint16
		Cap        uint16
		BattleMode uint8
	}

	info := serverInfo{
		Hash:       binary.BigEndian.Uint32(data[8:12]),
		Timestamp:  binary.BigEndian.Uint64(data[12:20]),
		Players:    binary.BigEndian.Uint16(data[20:22]),
		Cap:        binary.BigEndian.Uint16(data[22:24]),
		BattleMode: data[24],
	}

	document := bson.M{
		"_id":     ip,
		"latency": latency,
		"data":    info,
	}

	filter := bson.M{"_id": ip}
	opts := options.Replace().SetUpsert(true)

	_, err := collection.ReplaceOne(context.TODO(), filter, document, opts)
	if err != nil {
		return err
	}

	return nil
}
