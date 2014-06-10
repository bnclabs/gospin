package main

import (
    "flag"
    "fmt"
    "github.com/goraft/raft"
    "github.com/prataprc/go-failsafe"
    "github.com/dustin/go-jsonpointer"
    "log"
    "math/rand"
    "os"
    "path/filepath"
    "io/ioutil"
    "encoding/json"
    "time"
    "reflect"
)

var options struct {
    name      string
    listAddr  string
    join      string
    nodes     int
    trace     bool
    debug     bool
}

var smallJSON, _ = ioutil.ReadFile("./testdata/small.json")
var ops = [4]string{"GETCAS", "GET", "SET", "DELETE"}

func init() {
    flag.StringVar(&options.name, "name", "failsafe", "server's unique name")
    flag.StringVar(&options.listAddr, "s", "localhost:4001", "host:port listen")
    flag.StringVar(&options.join, "join", "", "host:port of leader to join")
    flag.IntVar(&options.nodes, "nodes", 4, "number nodes in the cluster")
    flag.BoolVar(&options.trace, "trace", false, "Raft trace debugging")
    flag.BoolVar(&options.debug, "debug", false, "Raft debugging")
    flag.Usage = func() {
        fmt.Fprintf(os.Stderr, "Usage: %s [arguments] <data-path-dir> \n", os.Args[0])
        flag.PrintDefaults()
    }
}

func main() {
    flag.Parse()
    rand.Seed(time.Now().UnixNano())
    if options.trace {
        failsafe.SetLogLevel(raft.Trace)
    } else if options.debug {
        failsafe.SetLogLevel(raft.Debug)
    }

    failsafe.RegisterCommands() // Setup commands.

    // Set the data directory.
    if flag.NArg() == 0 {
        flag.Usage()
        log.Fatal("Data path argument required")
    }
    namePrefix := options.name
    listAddr, path := options.listAddr, flag.Arg(0)
    log.SetFlags(log.LstdFlags)

    names, addrs, paths, ports := setupNodes(namePrefix, path)
    quitch := make(chan []interface{})

    // leader
    leaderAddr := addrs[0]
    killchs := make([]chan []interface{}, 0, options.nodes)
    killch  := make(chan []interface{})
    failsafe.StartDemoServer(names[0], paths[0], listAddr, "", quitch, killch)
    killchs = append(killchs, killch)
    time.Sleep(1 * time.Second) // wait for it to become leader, in case.

    // Initialize leader with data
    client := failsafe.NewSafeDictClient("http://" + leaderAddr)
    if CAS, err := client.GetCAS(); err != nil {
        log.Fatal(err)
    } else {
        var m map[string]interface{}
        json.Unmarshal(smallJSON, &m)
        client.SetCAS("", m, CAS)
    }

    // followers
    for i := 1; i < options.nodes; i++ {
        killch = make(chan []interface{})
        failsafe.StartDemoServer(
            names[i], paths[i], listAddr, leaderAddr, quitch, killch)
        killchs = append(killchs, killch)
    }

    pointers, err := jsonpointer.ListPointers(smallJSON)
    if err != nil {
        log.Fatal(err)
    }
    sd, _ := failsafe.NewSafeDict(smallJSON, true)

    CAS, ch := float64(1), make(chan int)

    go clientRoutine(addrs, pointers, time.After(3*time.Second), ch)
    go clientRoutine(addrs, pointers, time.After(3*time.Second), ch)
    go clientRoutine(addrs, pointers, time.After(3*time.Second), ch)

    done := make([]int, 0)
    for {
        i := <-ch
        if i == 0 {
            done = append(done, i)
        }
        if len(done) == 3 {
            break
        }
        pointer := pointers[i%len(pointers)]
        op := ops[i%len(ops)]
        switch op {
        case "SET":
            CAS, _ = sd.Set(pointer, float64(i), CAS)
        case "DELETE":
            CAS, _ = sd.Delete(pointer, CAS)
        }
        if i % 5 == 0 {
            idx := i % options.nodes
            fmt.Printf("%v, shutting down\n", paths[idx])
            killchs[idx] <- []interface{}{failsafe.DemoCmdShutdown, 300, ""}
        }
    }
    leader := getLeader(addrs)
    val1, _, _ := leader.Get("")
    val2, _, _ := sd.Get("")
    fmt.Println(reflect.DeepEqual(val1, val2))

    for _, killch := range killchs {
        killch <- []interface{}{failsafe.DemoCmdQuit}
    }
}

func clientRoutine(addrs []string, pointers []string, timeout <-chan time.Time, ch chan int) {
    leader := getLeader(addrs)
    for {
        select {
        case <-timeout:
            ch <- 0
            return
        default:
            i := rand.Int()
            pointer := pointers[i%len(pointers)]
            op := ops[i%len(ops)]
            doOp(leader, op, pointer, float64(i), addrs)
            ch <- i
        }
    }
}

func doOp(leader *failsafe.SafeDictClient, op, pointer string, val interface{}, addrs []string) {
    var CAS uint64
    var err error

    for {
        i := rand.Int()
        switch op {
        case "GETCAS":
            _, err = leader.GetCAS()
        case "GET":
            _, _, err = leader.Get(pointer)
        case "SET":
            if i % 2 == 0 {
                if CAS, err = leader.GetCAS(); err == nil {
                    _, err = leader.Set(pointer, val)
                }
            } else {
                _, err = leader.SetCAS(pointer, val, CAS)
            }
        case "DELETE":
            if i % 2 == 0 {
                if CAS, err = leader.GetCAS(); err == nil {
                    _, err = leader.Delete(pointer)
                }
            } else {
                _, err = leader.DeleteCAS(pointer, CAS)
            }
        }
        if err == raft.NotLeaderError {
            leader = getLeader(addrs)
        } else {
            break
        }
    }
}

func getLeader(addrs []string) *failsafe.SafeDictClient {
    for _, addr := range addrs {
        client := failsafe.NewSafeDictClient("http://" + addr)
        name, leaderAddr, _ := client.GetLeader()
        if name != "" && leaderAddr != "" {
            return failsafe.NewSafeDictClient(leaderAddr)
        }
    }
    log.Fatal("no leader ", addrs)
    return nil
}

func handleError(path string, err error) {
    if err != nil {
        log.Fatalf("%v, %v\n", path, err)
    }
}

func setupNodes(namePrefix, pathdir string) (names, addrs, paths, ports []string) {
    names := make([]string, 0, options.nodes)
    addrs := make([]string, 0, options.nodes)
    paths := make([]string, 0, options.nodes)
    ports := make([]int, 0, options.nodes)
    for i := 0; i < options.nodes; i++ {
        path := filepath.Join(pathdir, fmt.Sprintf("%v", i))
        names = append(names, fmt.Sprintf("%s%d", name, i))
        addrs = append(addrs, fmt.Sprintf("%s:%d", options.host, options.port))
        paths = append(paths, path)
        ports = append(ports, options.port+i)
    }
    return names, addrs, paths, ports
}

