package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"net"
	"os"
	"os/signal"
	"runtime"
	"sync"
	"syscall"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var (
	ErrMaximumResponseLength = errors.New("maximum response length exceeded")
)

type Scanner interface {
	// Name of the scanner
	Name() string
	// Port to connect to
	Port() string
	// 'tcp' or 'udp'
	Network() string
	// Scans and returns the response and latency
	Scan(string, net.Conn) ([]byte, int64, error)
	// Saves the response to the database
	Save(string, int64, []byte, *mongo.Collection) error
}

func main() {
	ip := flag.String("ip", "", "IP address to start from")
	scannerName := flag.String("scanner", "http", "Scanner to use (default: http)")
	numWorkers := flag.Int("workers", runtime.NumCPU(), "Number of workers to use (default: number of threads)")
	tasksPerThread := flag.Int("tasks", 512, "Tasks per thread (default: 512)")
	flag.Parse()

	var scanner Scanner
	switch *scannerName {
	case "http":
		scanner = HTTPScanner{}
	case "minecraft":
		scanner = MinecraftScanner{}
	case "veloren":
		scanner = VelorenScanner{}
	default:
		fmt.Printf("Unknown scanner '%s'\n", *scannerName)
		os.Exit(1)
	}

	maxTasks := *tasksPerThread * 2 * *numWorkers
	ips := make(chan string, maxTasks)

	var wg sync.WaitGroup
	for i := 0; i < *numWorkers; i++ {
		wg.Add(1)
		go worker(scanner, *tasksPerThread, ips, &wg)
	}

	var seg_a, seg_b, seg_c, seg_d uint8

	if *ip != "" {
		fmt.Printf("Starting from IP '%s'\n", *ip)
		err := parseIP(*ip, &seg_a, &seg_b, &seg_c, &seg_d)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
	}

	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM)

	// Main loop
	for {
		select {
		case <-signals:
			fmt.Println("Received signal, shutting down...")
			close(ips)
			wg.Wait()
			fmt.Printf("Last IP: %d.%d.%d.%d\n", seg_a, seg_b, seg_c, seg_d)
			fmt.Printf("Done.\n")
			return

		default:
			if isReserved(&seg_a, &seg_b, &seg_c) {
				fmt.Println("Reserved range reached, skipping to next available range")
			}

			ips <- fmt.Sprintf("%d.%d.%d.%d", seg_a, seg_b, seg_c, seg_d)

			seg_d++
			if seg_d == 0 {
				seg_d = 0
				seg_c++

				if seg_c == 0 {
					seg_c = 0
					seg_b++

					if seg_b == 0 {
						seg_b = 0
						seg_a++

						if seg_a >= 224 {
							signals <- syscall.SIGTERM
							return
						}
					}
				}
			}
		}
	}
}

func worker(scanner Scanner, tasksPerThread int, ips <-chan string, wg *sync.WaitGroup) {
	defer wg.Done()

	client, err := mongo.Connect(context.TODO(), options.Client().ApplyURI("mongodb://localhost:27017"))
	if err != nil {
		fmt.Println(err)
		return
	}

	defer func() {
		err := client.Disconnect(context.TODO())
		if err != nil {
			fmt.Println(err)
		}
	}()

	collection := client.Database("hagelslag").Collection(scanner.Name())

	dialer := net.Dialer{Timeout: 1 * time.Second}
	semaphore := make(chan struct{}, tasksPerThread)

	port := scanner.Port()
	network := scanner.Network()

	for ip := range ips {
		// Get a semaphore slot
		semaphore <- struct{}{}

		go func(ip string) {
			// Release the slot
			defer func() { <-semaphore }()

			// Connect
			address := net.JoinHostPort(ip, port)
			conn, err := dialer.Dial(network, address)
			if err != nil {
				if errors.Is(err, os.ErrDeadlineExceeded) {
					fmt.Printf("TIMEOUT %s\n", ip)
					return
				}

				// Ignore logging for anything else
				return
			}
			defer conn.Close()

			// Read and Write deadline
			err = conn.SetDeadline(time.Now().Add(3 * time.Second))
			if err != nil {
				fmt.Printf("TIMEOUT %s\n", ip)
				return
			}

			response, latency, err := scanner.Scan(ip, conn)
			if len(response) == 0 && err == nil {
				// No response, or wrong response (not wanted, can be discarded)
				return
			}

			if err != nil {
				// Don't log EOF and connection reset errors
				if errors.Is(err, io.EOF) || errors.Is(err, net.ErrClosed) {
					return
				}

				fmt.Printf("ERROR %s | %s\n", ip, err)
				return
			}

			err = scanner.Save(ip, latency, response, collection)
			if err != nil {
				fmt.Printf("ERROR %s | error inserting to document: %s\n", ip, err)
				return
			}

			fmt.Printf("SUCCESS %s\n", ip)
		}(ip)
	}
}

// ************#*#******####*########***#####################%%%%@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@%%@@@@%%%@@@@@@@@@@@@@@@@@@@
// ************************************##################%##%%%@@@@@@@@@@@@@@@@@@@@@@@@@%%#@@@%#**+++++**#%@@@@%%@@@@%%%@@@@%@@@@@@@@@@@@@@
// **++++++**************************#####%%%%%%%%%%%%@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@+:.:.:..              :-+##%%%@#*#%@@@%@@@@@@@@@@@@@@
// ++==++++******######%%%%%%%%%%%%%%%%%%%###########%%%%%%@@@@@@@@@@@@@@@@@@@@@@@@@%-    ...          .        .=*-... .+@@%@@@@@@@@@@@@@@
// ####%%%%%####*****++++++++++***########%%%%%%%%%%%%%%%@@@@@@@@@@@@@@@@@@@@@@@@@@#:   .:...          ....        .      .*%@@@@@@@@@@@@@@
// +++++++*****###########%##################%%%%%%%%%%%%%%@@@@@@@@@@@@@@@@@@@@@@@=    -..      :        .:.                *@@@@@@@@@@@@@@
// ..::......::..::--=+***********###################%%%%%%@@@@@@@@@@@@@@@@@@@@@%:   .=.       .::        .::               .#%%%%%%@@@@@@@
//             ...:::.:=++++**######*#***######%%%%%%%%%%%@@@@@@@@@@@@@@@@@@@@@%.    :      .... :          =                =###%%%%%%%%%%
// :::-::::--------==*#%%%#*******########%%####%%%%%%%%%%@@@@@@@@@@@@@@@@@@@@%.    :      ..:.   :..        :                +***#%%%@@@@@
// +--::::::::::::--=#%@#***###***####*######%%%%@@@@@@@@@@@@@@@@@@@@@@@@@@@@@-    ..    .::.      :.:       :          .     =%@@@@@@@@@@@
// @%+:::::::::::::-**#@##*#####%%%%%@@@@@%%%%%%%%%%%%%%%@@@@@@@@@@@@@@@@@@@@%     :   :--:         :.:.      :         ..    .#%@@@@@@@@@@
// %@@@#=-::--::.:-=**#@###%%#########**######%%%%%%%%%@@@@@@@@@@@@@@@@@@@@@@=    ...:+=:.:.        ::::::    :          .     *%@@@@@@@@@@
// %%@@@@@%+---:.-#=***%########*#*##*######%%%%%%%@@@@@@@@@@@@@@@@@@@@@@@@@@:    -::.               ..:::..  :        : .     +@@@@@@@@@@@
// ##*#%@@@@%+-::*%+***%##**######%%%%%%%%%%%%%%@@@@@@@@@@@@@@@@@@@@@@@@@@@@@     :    ...::        .:... ..::-        .:.   . -%@@@@@@@@@@
// +*****#%@%++***%+**#@##*%%%%%%%%%%%%%%%%%%@@@@@@@@@@@@@@@@@@@@@@@@@@%----+     .  .:--.          .         :       .:-.   . :%@@@@@@@@@@
// *********#@++*#%+**#@##****#######%%%%@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@%=..+ .   :.-+%##*-.        :=*##*=+:.:       ...  ... .=*#%@@@@@@@
// **********##++===**#@##+***####%%%%%%@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@%=..-   +.                  -:::.:-:-       :    : :..::.=@@@@@@@
// **********#%#%%**=##@***#####%%%%%%@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@#=-. .+-        ..                :    . .    .    .=#@@@@@@@@@
// **********#%#%#*++##@**#%%%%%%@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@-==..*-.       ::               :::  .=:....   .   *@@@@@@@@@@
// *********#%+%@@++:##@###%%%%@@@@%%##*%@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@ .::.+.                        .-:..:-++.....      #@@@@@@@@@@
// ********#%+#@@*++-+#@#**#####%%%%@@@@+@@@@@@@@@@@@@@@@@@@@@@@@@@@@%%%%%%%* :..#==       .                  -..=-=:-          #@@@@@@@@@@
// *****##%**%%*+++++++=#%@@@@@@@@@@@@@@#*@%%%%%%%%%%@@@@@@@@@@@@@@%%%%%%%%%- : -*.-:.      ..........         .#+.#:.          =@@@@@@%%%#
// ****######%*#*+++++*+#@@@@@@@@@@@@@@@@+%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%* .: -*.-.-:                      .+@@=-*-           :%########*
// %@@@@@@@@%%#@@@*-#@@**@@@@@@@@@@@@@@@@@+%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%#. :. .: - :.--.                 .=%%@@:+*:           .**********
// @@@@@@%%@@@@@@%#-#*+:  -%@@@@@@@@@@@@@@*#%%%%%%%%%%%%%%%%%%%%%%%%%%%%%#=  :     ...:.+#+-.            .  +%%%#.-=            .=+*****#**
// @@@@@*  +%%%%%%%***::-. :%@@@@@@@@@@@@@@+%############################*   ..     : :.-#***+-.......      =***=..             -:*********
// @@@@@@+:+*##%%%%%%#####*.-@@@@@@@@@@@@@@%+%#################%%########-    :     :  :.*##**=             :++*:.              = ++++**##%
// @@@@+:-**=---+##*%%#####:.@@@@@@@@@@@@@@@**%%%%%%#####################.    :.     :  -=###*+              :+=.               + =+*###%%%
// @@@@+: =###****+-#+-=-+#- @@@@@@@@@@@@@@@@+%##########################:.    :     ..  +=--:-               :                .+ =*###%##%
// @@@@%**#####***#*%***###= ::==+*##%%%%%%@@#+#######**********##**+==:. .::  .:     :  .:   :              .                 -= -*****###
// @@@@-  +*#*****#*#****##+                        .:=+**#+=-:.            .:  :.     :  :   .             :                 :-  -==++++**
// @@@@%+=*##****#####***##*                                                  :  :     :  :                ..                .  .---=======
// @@@@#:-###**######*#**###                                                   :  :    :  -....    . ......:                .:------=======
// @@@@@@#*#######%%%#*##%#%-:..                                               .: ..   .. :       .       :       .          ---------=-=--
// @@@@@@*##%%###%########**@@@@@%#*+=:.                                  .:::. :  :    : :              ..      :           :-----=-------
// @@@@@%*###*##########**--%@@@@@@@@@@@@%#*+=-:.                 .:--=+++++=+-:.: ..   : :              :   .  ..           :--------=----
// @@@@@%#################+#@@@@@@@@@@@@@@@@@@@@@@***++=========++++++++++++++++.+  :   : :             ...     :  .         :--------:----
// @@@@@@@@%%%@@@@@@@@@@%%@@@@@@@@@@@@@@@@@@@@@@@@@+**********+++++++++++++++=   -  :   : :             : : :  :. .          ...:::....::::
// @@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@%****++++++********++++++++...:. -   :.=             :.- :  : ..                      ..
// @@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@***************++*******++    : +  .::  ............:--..  : :
// @@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@+#***********+++++++++++-    :.-  - :....   .......-:::  ....
// @@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@#****##******++****++===-    -:: .*.-              :--:  : :
// @@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@#*+==***++=+***###*+++++++++:   -::.--.=              ::-:  . :
// @@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@#*++====+*##+:    .=+*+****+**++++=   =  ::.-:              ::::  : :
// @@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@%*++=-=+++#%%@@@@@%.         :-=********=      :- : .             .=::. -.-
// @@@@@@@@@@@@@@@@@@@@@@@@@@@@@%#*+====++*#%%@@@@@@@@@@=             .---+**+-       :   .              =.:: :::
// @@@@@@@@@@@@@@@@@@@@@@@%#*+====++*#%%@@@@@@@@@@@@@@@#.                     .            :               :- :.=                         :
// @@@@@@@@@@@@@@@@@%#**+===++*#%%@@@@@@@@@@@@@@@@@@@@@:.                    .              :               =:. +.                     :==-
// @@@@@@@@@@@@%**+====+*#%%@@@@@@@@@@@@@@@@@@@@@@@@@@#=*=::                 .               .              ::  -                  ..:----=
// @@@@@@%**+=-==+*#%%@@@@@@@@@@@@@@@@@@@@@@@@@@@@%%##%%**+*#-               :               .              : . -               -=+=--=====
// %#*+=-=+++*%%@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@%@#+=%%#%%%%%%#:              ...              .               . :           . -+===========
