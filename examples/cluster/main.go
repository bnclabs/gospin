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
	"path/filepath"
	"time"
)

var options struct {
	trace bool
	debug bool
	host  string
	port  int
	join  string
	nodes int
}

func init() {
	flag.BoolVar(&options.trace, "trace", false, "Raft trace debugging")
	flag.BoolVar(&options.debug, "debug", false, "Raft debugging")
	flag.StringVar(&options.host, "h", "localhost", "hostname")
	flag.IntVar(&options.port, "p", 4001, "port")
	flag.StringVar(&options.join, "join", "", "host:port of leader to join")
	flag.IntVar(&options.nodes, "nodes", 4, "number nodes in the cluster")
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [arguments] <data-path-dir> \n", os.Args[0])
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
	log.SetFlags(log.LstdFlags)

	host, path := options.host, flag.Arg(0)
	addrs, paths, ports := setupNodes(path)

	listeners := make([]net.Listener, options.nodes)
	daemons := make([]*failsafe.Server, options.nodes)

	// leader
	leaderAddr := addrs[0]
	listeners[0], _, daemons[0] = startServer(paths[0], "", host, ports[0])

	// followers
	for i := 1; i < options.nodes; i++ {
		lis, _, fsd := startServer(paths[i], leaderAddr, host, ports[i])
		listeners[i] = lis
		daemons[i] = fsd
	}

	client := failsafe.NewSafeDictClient("http://" + leaderAddr)
	CAS, err := client.GetCAS()
	handleError(err)
	fmt.Printf("Got initial CAS %v\n", CAS)

	for {
		CAS, err = client.SetCAS("/eyeColor", "brown", CAS)
		handleError(err)
		fmt.Println("Set /eyeColor gave nextCAS as", CAS)
		time.Sleep(1 * time.Second)
	}

	//value, CAS, err := client.Get("/eyeColor")
	//handleError(err)
	//fmt.Printf("Get /eyeColor returned %v with CAS %v\n", value, CAS)

	//fsd.Stop()
	//lis.Close()
}

func startServer(
	path, join, host string,
	port int) (net.Listener, *http.Server, *failsafe.Server) {

	connAddr := fmt.Sprintf("%s:%d", host, port)

	mux := http.NewServeMux()
	httpd := &http.Server{
		Addr:    connAddr,
		Handler: mux,
	}
	lis, err := net.Listen("tcp", connAddr)
	if err != nil {
		log.Fatal(err)
	}

	fsd, err := failsafe.NewServer(path, host, fmt.Sprintf("%d", port), mux)
	if err != nil {
		log.Fatal(err)
	}
	if options.trace {
		raft.SetLogLevel(raft.Trace)
		log.Print("Raft trace debugging enabled.")
	} else if options.debug {
		raft.SetLogLevel(raft.Debug)
		log.Print("Raft debugging enabled.")
	}
	fsd.Install(join)

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

func setupNodes(pathdir string) ([]string, []string, []int) {
	addrs := make([]string, 0, options.nodes)
	paths := make([]string, 0, options.nodes)
	ports := make([]int, 0, options.nodes)
	for i := 0; i < options.nodes; i++ {
		path := filepath.Join(pathdir, fmt.Sprintf("%v", i))
		addrs = append(addrs, fmt.Sprintf("%s:%d", options.host, options.port))
		paths = append(paths, path)
		ports = append(ports, options.port+i)
	}
	return addrs, paths, ports
}
