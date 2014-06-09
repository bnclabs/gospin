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
	port  int
	join  string
}

func init() {
	flag.BoolVar(&options.trace, "trace", false, "Raft trace debugging")
	flag.BoolVar(&options.debug, "debug", false, "Raft debugging")
	flag.StringVar(&options.host, "h", "localhost", "hostname")
	flag.IntVar(&options.port, "p", 4001, "port")
	flag.StringVar(&options.join, "join", "", "host:port of leader to join")
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [arguments] <data-path> \n", os.Args[0])
		flag.PrintDefaults()
	}
}

func main() {
	flag.Parse()
	log.SetFlags(0)

	rand.Seed(time.Now().UnixNano())

	// Setup commands.
	failsafe.RegisterCommands()

	// Set the data directory.
	if flag.NArg() == 0 {
		flag.Usage()
		log.Fatal("Data path argument required")
	}
	path := flag.Arg(0)
	log.SetFlags(log.LstdFlags)

	connAddr := fmt.Sprintf("%v:%v", options.host, options.port)
	port := fmt.Sprintf("%d", options.port)
	lis, _, fsd := startServer(path, connAddr, options.host, port)
	if options.trace {
		fsd.SetLogLevel(raft.Trace)
	} else if options.debug {
		fsd.SetLogLevel(raft.Debug)
	}

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

func startServer(path, connAddr, host,
	port string) (lis net.Listener, httpd *http.Server, fsd *failsafe.Server) {

	var err error

	mux := http.NewServeMux()
	httpd = &http.Server{
		Addr:    connAddr,
		Handler: mux,
	}
	if lis, err = net.Listen("tcp", connAddr); err != nil {
		log.Fatal(err)
	}

	if fsd, err = failsafe.NewServer(path, host, port, mux); err != nil {
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
