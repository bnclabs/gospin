package main

import (
    "flag"
    "fmt"
    "github.com/goraft/raft"
    "github.com/prataprc/go-failsafe"
    "log"
    "math/rand"
    "os"
    "net/http"
    "net"
    "path/filepath"
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
    flag.BoolVar(&options.trace,  "trace", false, "Raft trace debugging")
    flag.BoolVar(&options.debug,  "debug", false, "Raft debugging")
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
    log.SetFlags(log.LstdFlags)
    nodeCount := 4

    leader := fmt.Sprintf("%s:%d", options.host, options.port)
    path := flag.Arg(0)
    paths := setupPaths(path, nodeCount)
    lis, _, fsd := startServer(paths[0], "", options.host, options.port)

    listeners := make([]net.Listener, 0, nodeCount-1)
    daemons   := make([]*failsafe.Server, 0, nodeCount-1)
    for i := 1; i < nodeCount; i++ {
        host, port := options.host, options.port+i
        lis, _, fsd := startServer(paths[i], leader, host, port)
        listeners = append(listeners, lis)
        daemons = append(daemons, fsd)
    }
    fmt.Println(lis, fsd, listeners, daemons)

    client := failsafe.NewSafeDictClient("http://" + leader)
    CAS, err := client.GetCAS()
    handleError(err)
    fmt.Printf("%v, Got initial CAS %v\n", leader, CAS)

    for {
        CAS, err = client.SetCAS("/eyeColor", "brown", CAS)
        handleError(err)
        fmt.Println("Set /eyeColor gave nextCAS as", CAS)
        time.Sleep(1*time.Second)
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

    mux   := http.NewServeMux()
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

func setupPaths(path string, count int) []string {
    if err := os.RemoveAll(path); err != nil {
        log.Fatalf("Unable to remove path %v: %v", path, err)
    }
    if err := os.MkdirAll(path, 0744); err != nil {
        log.Fatalf("Unable to create path %v: %v", path, err)
    }

    paths := make([]string, 0, count)
    for i := 0; i < count; i++ {
        serverPath := filepath.Join(path, fmt.Sprintf("%v", i))
        if err := os.MkdirAll(serverPath, 0744); err != nil {
            log.Fatalf("Unable to create path %v: %v", serverPath, err)
        }
        paths = append(paths, serverPath)
    }
    return paths
}
