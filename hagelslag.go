package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"net"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/writeconcern"
)

type Hagelslag struct {
	// This channel is embedded since its pretty contained in this struct
	connections chan string

	Scanner Scanner

	StartingIP  string
	Port        string
	URI         string
	OnlyConnect bool
	Rate        int
}

type Scanner interface {
	// Name of the scanner
	Name() string
	// Port to connect to
	Port() string
	// 'tcp' or 'udp'
	Network() string
	// Responsible for sending and receiving all the necessary data for saving
	Scan(ip string, conn net.Conn) ([]byte, int64, error)
	// Saves the response to the database
	Save(ip string, latency int64, data []byte, collection *mongo.Collection) error
}

func NewHagelslag() (Hagelslag, error) {
	ip := flag.String("ip", "", "IP address to start from, without port")
	scannerName := flag.String("scanner", "http", "Scanner to use (default: http)")
	port := flag.String("port", "", "Override the scanners port")
	uri := flag.String("uri", "mongodb://localhost:27017", "MongoDB URI (default: mongodb://localhost:27017)")
	connect := flag.Bool("only-connect", false, "Skip scanning, connect and save if successful (default: false)")
	rate := flag.Int("rate", 1000, "Limit of connections, be careful with this value (default: 1000)")
	flag.Parse()

	h := Hagelslag{
		StartingIP:  *ip,
		URI:         *uri,
		OnlyConnect: *connect,
		Rate:        *rate,
	}

	// Checking if the database is reachable
	client, err := mongo.Connect(context.TODO(), options.Client().SetServerSelectionTimeout(3*time.Second).ApplyURI(h.URI))
	if err != nil {
		return Hagelslag{}, fmt.Errorf("failed to connect to database: %s", err)
	}

	err = client.Ping(context.TODO(), nil)
	if err != nil {
		return Hagelslag{}, fmt.Errorf("failed to ping database: %s", err)
	}

	err = client.Disconnect(context.TODO())
	if err != nil {
		return Hagelslag{}, fmt.Errorf("failed to disconnect from database: %s", err)
	}

	scanner := strings.ToLower(*scannerName)

	switch scanner {
	case "http":
		h.Scanner = HTTP{}
	case "minecraft":
		h.Scanner = Minecraft{}
	case "veloren":
		h.Scanner = Veloren{}
	default:
		return Hagelslag{}, fmt.Errorf("unknown scanner '%s'", scanner)
	}

	if *port != "" {
		h.Port = *port
	} else {
		h.Port = h.Scanner.Port()
	}

	if h.OnlyConnect {
		file, err := os.OpenFile("connections.out", os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0644)
		if err != nil {
			return Hagelslag{}, fmt.Errorf("failed to open file: %s", err)
		}

		h.connections = make(chan string)
		go h.saveConnections(file)
	}

	return h, nil
}

func (h Hagelslag) worker(addresses chan string, semaphore chan struct{}, wg *sync.WaitGroup) {
	defer wg.Done()

	options := options.Client().
		ApplyURI(h.URI).
		SetWriteConcern(&writeconcern.WriteConcern{})

	client, err := mongo.Connect(context.TODO(), options)
	if err != nil {
		fmt.Printf("failed to connect to database: %s\n", err)
		return
	}

	err = client.Ping(context.TODO(), nil)
	if err != nil {
		fmt.Printf("failed to ping database: %s\n", err)
		return
	}

	name := h.Scanner.Name()
	network := h.Scanner.Network()

	dialer := net.Dialer{
		KeepAlive: -1,
		Timeout:   1 * time.Second,
	}

	collection := client.Database("hagelslag").Collection(name)

	for address := range addresses {
		go h.spawn(semaphore, address, network, dialer, collection)
	}

	err = client.Disconnect(context.TODO())
	if err != nil {
		fmt.Printf("failed to disconnect from database: %s\n", err)
	}
}

func (h Hagelslag) spawn(semaphore chan struct{}, address string, network string, dialer net.Dialer, collection *mongo.Collection) {
	// Release the slot when done
	defer func() { <-semaphore }()

	// Connection
	conn, err := dialer.Dial(network, address)
	if err != nil {
		// Don't log anything
		return
	}

	defer conn.Close()

	if h.OnlyConnect {
		atomic.AddInt64(&SUCCESS, 1)
		h.connections <- address
		return
	}

	// Read and Write deadline
	err = conn.SetDeadline(time.Now().Add(3 * time.Second))
	if err != nil {
		return
	}

	response, latency, err := h.Scanner.Scan(address, conn)
	if len(response) == 0 && err == nil {
		// No response, or wrong response (not wanted, can be discarded)
		return
	}

	if err != nil {
		// Don't log these errors
		if errors.Is(err, os.ErrDeadlineExceeded) || errors.Is(err, syscall.ECONNRESET) || errors.Is(err, io.EOF) {
			return
		}

		if SHUTTING_DOWN {
			return
		}

		os.Stderr.WriteString("\nERROR SCAN " + address + ": " + err.Error() + "\n")
		return
	}

	err = h.Scanner.Save(address, latency, response, collection)
	if err != nil {
		if SHUTTING_DOWN {
			return
		}

		os.Stderr.WriteString("\nERROR SAVE " + address + ": " + err.Error() + "\n")
		return
	}

	atomic.AddInt64(&SUCCESS, 1)
}

// Only used when OnlyConnect is true
//
// Wait for addresses and append them to a file
func (h Hagelslag) saveConnections(file *os.File) {
	defer file.Close()

	portLen := len(h.Port) + 1

	for address := range h.connections {
		// Removing the port
		address = address[:len(address)-portLen]

		_, err := file.WriteString(address + "\n")
		if err != nil {
			os.Stderr.WriteString("\nERROR SAVE " + address + ": " + err.Error() + "\n")
		}
	}
}
