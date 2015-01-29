package main

import (
	"container/list"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/signal"
	"runtime"
	"syscall"
	"time"
	"github.com/michaelklishin/rabbit-hole"
)

var (
	addr = flag.String("addr", ":7000", "Server port")
	rmqc, _ = NewClient("http://127.0.0.1:15672", "guest", "guest") // use default config [test only]
)

func ChannelsKeeper(clients chan chan string) {
	channels := list.New()
	go func() {
		for {
			select {
			case c := <-clients:
				channels.PushBack(c)
				fmt.Printf("New client: %d\n", channels.Len())
			}
		}
	}()
}

func InstallSignalHandlers(signals chan os.Signal) {
	go func() {
		sig := <-signals
		switch sig {
		case syscall.SIGINT:
			fmt.Printf("\nCtrl-C signalled\n")
			os.Exit(0)
		}
	}()
}

func CreatePidfile() {
	pid := []byte(fmt.Sprintf("%d", os.Getpid()))
	ioutil.WriteFile("long_polling.pid", pid, 0755)
}

func MakeLPHandler(clients chan chan string) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		message := make(chan string, 1)
		clients <- message

		select {
		case <-time.After(60e9):
			io.WriteString(w, "Timeout!\n")
		case msg := <-message:
			io.WriteString(w, msg)
		}
	}
}

func FetchHendler(clients chan chan string) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, "Hello Fetch")
	}
}

func CreateHttpServer(clients chan chan string) {
	http.HandleFunc("/", MakeLPHandler(clients))
	http.HandleFunc("/fetch", MakeLPHandler(clients))
	log.Println("Listen On" + *addr)
	err := http.ListenAndServe(*addr, nil)
	if err != nil {
		log.Fatal("ListenAndServe: ", err.Error())
	}
}

func main() {
	runtime.GOMAXPROCS(runtime.NumCPU() * 2 + 1)
	flag.Parse()

	clients := make(chan chan string, 1)
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGINT, syscall.SIGUSR1)

	CreatePidfile()
	ChannelsKeeper(clients)
	InstallSignalHandlers(signals)
	CreateHttpServer(clients)
} 

