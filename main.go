package main

import (
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func main() {
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix

	hagelslag, err := NewHagelslag()
	if err != nil {
		panic(err)
	}

	ips := make(chan string, hagelslag.MaxTasks)

	var wg sync.WaitGroup
	for range hagelslag.NumWorkers {
		wg.Add(1)
		go hagelslag.worker(ips, &wg)
	}

	segA, segB, segC, segD, err := parseIP(hagelslag.StartingIP)
	if err != nil {
		panic(err)
	}

	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM)

	counter := int64(0)

	go getIPPerSecond(&counter)

	// Main loop
	for {
		select {
		case <-signals:
			fmt.Println("Received signal, shutting down...")
			close(ips)
			wg.Wait()
			fmt.Printf("Last IP: %d.%d.%d.%d\n", segA, segB, segC, segD)
			fmt.Printf("Done.\n")
			return

		default:
			ip := strconv.Itoa(segA) + "." + strconv.Itoa(segB) + "." + strconv.Itoa(segC) + "." + strconv.Itoa(segD)

			if isReserved(&segA, &segB, &segC) {
				log.Log().Str("ip", ip).Msg("Reserved range reached, skipping to next available range")
			}

			ips <- ip
			atomic.AddInt64(&counter, 1)

			segD++
			if segD >= 256 {
				segD = 0
				segC++

				if segC >= 256 {
					segC = 0
					segB++

					if segB >= 256 {
						segB = 0
						segA++

						if segA >= 224 {
							signals <- syscall.SIGTERM
							return
						}
					}
				}
			}
		}
	}
}

func getIPPerSecond(counter *int64) {
	secondTicker := time.NewTicker(1 * time.Second)

	for range secondTicker.C {
		ipsPerSecond := atomic.LoadInt64(counter)
		log.Log().Int64("ips", ipsPerSecond).Msg("IPs per second")
		atomic.StoreInt64(counter, 0)
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
