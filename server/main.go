package main

import (
    "flag"
    "fmt"
    "github.com/goraft/raft"
    "github.com/prataprc/go-failsafe"
    "log"
    "math/rand"
    "os"
    "time"
)

var options struct {
    host    string
    port    int
    join    string
    seed    int
    verbose bool
    trace   bool
    debug   bool
}

func init() {
    seed := int(time.Now().UnixNano())
    flag.StringVar(&options.host, "h", "localhost", "hostname")
    flag.IntVar(&options.port, "p", 4001, "port")
    flag.StringVar(&options.join, "join", "", "host:port of leader to join")
    flag.IntVar(&options.seed, "seed", seed, "seed value")
    flag.BoolVar(&options.verbose, "v", false, "verbose logging")
    flag.BoolVar(&options.trace, "trace", false, "Raft trace debugging")
    flag.BoolVar(&options.debug, "debug", false, "Raft debugging")
    flag.Usage = func() {
        fmt.Fprintf(os.Stderr, "Usage: %s [arguments] <data-path> \n", os.Args[0])
        flag.PrintDefaults()
    }
}

func main() {
    log.SetFlags(0)

    flag.Parse()
    if flag.NArg() == 0 { // Set the data directory.
        flag.Usage()
        log.Fatal("Data path argument required")
    }
    rand.Seed(int64(options.seed))

    if options.verbose {
        log.Print("Verbose logging enabled.")
    }

    if options.trace {
        raft.SetLogLevel(raft.Trace)
        log.Print("Raft trace debugging enabled.")
    } else if options.debug {
        raft.SetLogLevel(raft.Debug)
        log.Print("Raft debugging enabled.")
    }

    registerCommands()

    path := flag.Arg(0)
    if err := os.MkdirAll(path, 0744); err != nil {
        log.Fatalf("Unable to create path: %v", err)
    }

    log.SetFlags(log.LstdFlags)
    s := NewHTTPServer(path, options.host, options.port)
    log.Fatal(s.ListenAndServe(options.join))
}

// Setup commands for go-failsafe.
func registerCommands() {
    raft.RegisterCommand(&failsafe.SetCommand{})
    raft.RegisterCommand(&failsafe.DeleteCommand{})
}

