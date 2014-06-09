package main

import (
	"flag"
	"fmt"
	"github.com/goraft/raft"
	"github.com/prataprc/go-failsafe"
	"log"
	"math/rand"
	"net"
	"net/http"
	"os"
	"time"
)

var options struct {
	trace bool
	debug bool
	host  string
	port  string
	join  string
}

func init() {
	flag.BoolVar(&options.trace, "trace", false, "Raft trace debugging")
	flag.BoolVar(&options.debug, "debug", false, "Raft debugging")
	flag.StringVar(&options.host, "h", "localhost", "hostname")
	flag.StringVar(&options.port, "p", "4001", "port")
	flag.StringVar(&options.join, "join", "", "host:port of leader to join")
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [arguments] <data-path> \n", os.Args[0])
		flag.PrintDefaults()
	}
}

func main() {
	flag.Parse()
	log.SetFlags(0)

	if options.trace {
		raft.SetLogLevel(raft.Trace)
		log.Print("Raft trace debugging enabled.")
	} else if options.debug {
		raft.SetLogLevel(raft.Debug)
		log.Print("Raft debugging enabled.")
	}

	rand.Seed(time.Now().UnixNano())

	// Setup commands.
	failsafe.RegisterCommands()

	// Set the data directory.
	if flag.NArg() == 0 {
		flag.Usage()
		log.Fatal("Data path argument required")
	}
	path := flag.Arg(0)
	if err := os.MkdirAll(path, 0744); err != nil {
		log.Fatalf("Unable to create path: %v", err)
	}
	log.SetFlags(log.LstdFlags)

	connAddr := fmt.Sprintf("%v:%v", options.host, options.port)
	lis, _, fsd := startServer(path, connAddr, options.host, options.port)

	client := failsafe.NewSafeDictClient("http://" + connAddr)
	CAS, err := client.GetCAS()
	handleError(err)
	fmt.Println("Got initial CAS", CAS)

	CAS, err = client.SetCAS("/eyeColor", "brown", CAS)
	handleError(err)
	fmt.Println("Set /eyeColor gave nextCAS as", CAS)

	value, CAS, err := client.Get("/eyeColor")
	handleError(err)
	fmt.Printf("Get /eyeColor returned %v with CAS %v\n", value, CAS)

	fsd.Stop()
	lis.Close()
}

func startServer(path, connAddr, host, port string) (net.Listener, *http.Server, *failsafe.Server) {
	mux := http.NewServeMux()
	httpd := &http.Server{
		Addr:    connAddr,
		Handler: mux,
	}
	lis, err := net.Listen("tcp", connAddr)
	if err != nil {
		log.Fatal(err)
	}

	fsd, err := failsafe.NewServer(path, host, port, mux)
	if err != nil {
		log.Fatal(err)
	}
	fsd.Install(options.join)

	// Server routine
	go func() {
		log.Printf("%s: http server starting ...\n", path)
		err := httpd.Serve(lis) // serve until listener is closed.
		if err != nil {
			log.Printf("%s, error: %v\n", path, err)
		}
	}()
	return lis, httpd, fsd
}

func handleError(err error) {
	if err != nil {
		log.Fatal(err)
	}
}
