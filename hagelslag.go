package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"net"
	"os"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/rs/zerolog/log"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var (
	ErrMaximumResponseLength = errors.New("maximum response length exceeded")
)

type Hagelslag struct {
	// IP address to start scanning from
	StartingIP string
	// Number of workers to use (default: number of threads)
	NumWorkers int
	// Maximum number of tasks that the main channel can hold (tasks-per-thread * 2 * workers)
	MaxTasks int
	// Tasks per thread (default: 512)
	TasksPerThread int
	// Scanner being used (default: http)
	Scanner Scanner
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
	numWorkers := flag.Int("workers", runtime.NumCPU(), "Number of workers to use (default: number of threads)")
	tasksPerThread := flag.Int("tasks-per-thread", 512, "Tasks per thread (default: 512)")
	flag.Parse()

	h := Hagelslag{
		StartingIP:     *ip,
		NumWorkers:     *numWorkers,
		MaxTasks:       *tasksPerThread * 2 * *numWorkers,
		TasksPerThread: *tasksPerThread,
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

func (h Hagelslag) worker(ips <-chan string, wg *sync.WaitGroup) {
	defer wg.Done()

	options := options.Client().ApplyURI("mongodb://localhost:27017")

	client, err := mongo.Connect(context.TODO(), options)
	if err != nil {
		panic(fmt.Sprintf("Error connecting to database: %s\n", err))
	}

	err = client.Ping(context.TODO(), nil)
	if err != nil {
		panic(fmt.Sprintf("Error pinging database: %s\n", err))
	}

	name := h.Scanner.Name()
	port := h.Scanner.Port()
	network := h.Scanner.Network()

	dialer := net.Dialer{Timeout: 1 * time.Second}
	collection := client.Database("hagelslag").Collection(name)

	// Responsible for controlling how many tasks can be processed
	semaphore := make(chan struct{}, h.TasksPerThread)

	for ip := range ips {
		// Get a slot to work on a task
		semaphore <- struct{}{}
		go h.spawn(semaphore, ip, port, network, dialer, collection)
	}

	err = client.Disconnect(context.TODO())
	if err != nil {
		panic(fmt.Sprintf("Error disconnecting from database: %s\n", err))
	}
}

func (h Hagelslag) spawn(semaphore <-chan struct{}, ip string, port string, network string, dialer net.Dialer, collection *mongo.Collection) {
	// Release the slot when done
	defer func() { <-semaphore }()

	// Connect
	address := net.JoinHostPort(ip, port)
	conn, err := dialer.Dial(network, address)
	if err != nil {
		if errors.Is(err, os.ErrDeadlineExceeded) {
			return
		}

		// Ignore logging for anything else
		return
	}
	defer conn.Close()

	// Read and Write deadline
	err = conn.SetDeadline(time.Now().Add(3 * time.Second))
	if err != nil {
		return
	}

	response, latency, err := h.Scanner.Scan(ip, conn)
	if len(response) == 0 && err == nil {
		// No response, or wrong response (not wanted, can be discarded)
		return
	}

	if err != nil {
		// Don't log EOF and connection reset errors
		if errors.Is(err, io.EOF) || errors.Is(err, net.ErrClosed) {
			return
		}

		log.Log().Str("ip", ip).Err(err).Msg("SCAN")
		return
	}

	err = h.Scanner.Save(ip, latency, response, collection)
	if err != nil {
		log.Log().Str("ip", ip).Err(err).Msg("SAVE")
		return
	}

	log.Log().Str("ip", ip).Msg("SAVED")
}
