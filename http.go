package main

import (
	"context"
	"io"
	"net"
	"strings"
	"time"
	"unsafe"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type HTTP struct{}

func (s HTTP) Name() string {
	return "http"
}

func (s HTTP) Network() string {
	return "tcp"
}

func (s HTTP) Port() string {
	return "80"
}

func (s HTTP) Scan(ip string, conn net.Conn) ([]byte, int64, error) {
	request := []string{"GET / HTTP/1.1\r\nHost: ", ip, "\r\nConnection: close\r\n\r\n"}
	get := strings.Join(request, "")

	start := time.Now()
	_, err := conn.Write(unsafe.Slice(unsafe.StringData(get), len(get)))
	if err != nil {
		return nil, 0, err
	}

	response := make([]byte, 17)

	_, err = io.ReadFull(conn, response)
	if err != nil {
		return nil, 0, err
	}

	latency := time.Since(start).Milliseconds()

	// Check if the status code is 2xx.
	if response[9] != '2' {
		return nil, 0, nil
	}

	response, err = read(conn, MAX_RESPONSE_LENGTH)
	if err != nil {
		return nil, 0, err
	}

	return response, latency, nil
}

func (s HTTP) Save(ip string, latency int64, data []byte, collection *mongo.Collection) error {
	document := bson.M{
		"_id":     ip,
		"latency": latency,
		"data":    *(*string)(unsafe.Pointer(&data)),
	}

	filter := bson.M{"_id": ip}
	opts := options.Replace().SetUpsert(true)

	_, err := collection.ReplaceOne(context.TODO(), filter, document, opts)
	if err != nil {
		return err
	}

	return nil
}
