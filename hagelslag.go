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

var (
	ErrMaximumResponseLength = errors.New("maximum response length exceeded")
)

type Hagelslag struct {
	Scanner     Scanner
	StartingIP  string
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
	// Scans and returns the response and latency
	Scan(ip string, conn net.Conn) ([]byte, int64, error)
	// Saves the response to the database
	Save(ip string, latency int64, data []byte, collection *mongo.Collection) error
}

func NewHagelslag() (Hagelslag, error) {
	ip := flag.String("ip", "", "IP address to start from")
	scannerName := flag.String("scanner", "http", "Scanner to use (default: http)")
	uri := flag.String("uri", "mongodb://localhost:27017", "MongoDB URI (default: mongodb://localhost:27017)")
	onlyConnect := flag.Bool("only-connect", false, "Only connect to IPs, skipping scan/save (default: false)")
	rate := flag.Int("rate", 1000, "Limit of connections (default: 1000)")
	flag.Parse()

	h := Hagelslag{
		StartingIP:  *ip,
		URI:         *uri,
		OnlyConnect: *onlyConnect,
		Rate:        *rate,
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

	return h, nil
}

func (h Hagelslag) worker(addresses chan string, semaphore chan struct{}, wg *sync.WaitGroup) {
	defer wg.Done()

	options := options.Client().
		ApplyURI(h.URI).
		SetWriteConcern(&writeconcern.WriteConcern{})

	client, err := mongo.Connect(context.TODO(), options)
	if err != nil {
		panic(fmt.Sprintf("Error connecting to database: %s\n", err))
	}

	err = client.Ping(context.TODO(), nil)
	if err != nil {
		panic(fmt.Sprintf("Error pinging database: %s\n", err))
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
		panic(fmt.Sprintf("Error disconnecting from database: %s\n", err))
	}
}

func (h Hagelslag) spawn(semaphore chan struct{}, address string, network string, dialer net.Dialer, collection *mongo.Collection) {
	// Release the slot when done
	defer func() { <-semaphore }()

	// Connect
	conn, err := dialer.Dial(network, address)
	if err != nil {
		// Don't log timeouts
		if errors.Is(err, os.ErrDeadlineExceeded) {
			return
		}

		// Ignore logging for anything else
		return
	}
	defer conn.Close()

	if h.OnlyConnect {
		atomic.AddInt64(&successCount, 1)
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
		// Don't log timeouts, connection reset errors or  EOF
		if errors.Is(err, os.ErrDeadlineExceeded) || errors.Is(err, syscall.ECONNRESET) || errors.Is(err, io.EOF) {
			return
		}

		if shuttingDown {
			return
		}

		os.Stderr.WriteString("\nERROR SCAN " + address + ": " + err.Error())
		return
	}

	err = h.Scanner.Save(address, latency, response, collection)
	if err != nil {
		if shuttingDown {
			return
		}

		os.Stderr.WriteString("\nERROR SAVE " + address + ": " + err.Error())
		return
	}

	atomic.AddInt64(&successCount, 1)
}
