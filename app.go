package failsafe

import (
    "fmt"
    "time"
    "net"
    "log"
    "net/http"
    "os"
    "path/filepath"
)

const (
    DemoCmdShutdown byte = iota + 1
    DemoCmdHarakiri
    DemoCmdQuit
)

func StartDemoServer(
    path, leader, host string, port int,
    quitch chan<- []interface{},
    killch <-chan []interface{}) {

    go func() {
    loop:
        for {
            if leader != "" {
                //cleanServer(path)
            }
            portStr  := fmt.Sprintf("%d", port)
            connAddr := fmt.Sprintf("%v:%v", host, port)
            mux      := http.NewServeMux()
            httpd    := &http.Server{Addr: connAddr, Handler: mux}

            lis, err := net.Listen("tcp", connAddr)
            if err != nil {
                log.Fatal(path, err)
            }
            fsd, err := NewServer(path, host, portStr, mux)
            if err != nil {
                log.Fatal(path, err)
            }
            fsd.Install(leader)

            go startDemo(lis, httpd, fsd)

            msg := <-killch
            switch msg[0].(byte) {
            case DemoCmdShutdown:
                timeout := msg[1].(int)
                leader   = msg[2].(string)
                shutdownDemo(lis, fsd)
                time.Sleep(time.Duration(timeout) * time.Millisecond)
            case DemoCmdHarakiri:
                lis.Close()
            case DemoCmdQuit:
                shutdownDemo(lis, fsd)
                break loop
            }
        }
        time.Sleep(900*time.Millisecond)
        quitch <- []interface{}{true}
    }()
}

func shutdownDemo(lis net.Listener, fsd *Server) {
    fsd.Stop()
    lis.Close()
}

func startDemo(lis net.Listener, httpd *http.Server, fsd *Server) {
    name := fsd.raftServer.Name()
    // Server routine
    log.Printf("%s: http server starting ...\n", name)
    err := httpd.Serve(lis) // serve until listener is closed.
    if err != nil {
        log.Printf("%s, error: %v\n", name, err)
    }
}

func cleanServer(path string) {
    log.Printf("%v, cleaning path\n", path)
    confFile := filepath.Join(path, "conf")
    if err := os.Remove(confFile); err != nil {
        log.Printf("%v, %v\n", path, err)
    }
    logFile := filepath.Join(path, "log")
    if err := os.Remove(logFile); err != nil {
        log.Printf("%v, %v\n", path, err)
    }
    //snapshotDir := filepath.Join(path, "snapshot")
    if err := os.RemoveAll(path); err != nil {
        log.Printf("%v, %v\n", path, err)
    }
}

func printServerState(fsd *Server) {
    s := fsd.GetRaftserver()
    // state of the server
    fmt.Printf("\tName: %v\n", s.Name())
    fmt.Printf("\tPath: %v\n", s.Path())
    fmt.Printf("\tQuorumsize: %v\n", s.QuorumSize())
    fmt.Printf("\tState: %v\n", s.State())
    fmt.Printf("\tIsRunning: %v\n", s.Running())
    fmt.Printf("\tLogPath: %v\n", s.LogPath())
    fmt.Printf("\tSnapshotPath: %v\n", s.SnapshotPath(s.Term(), s.CommitIndex()))
    fmt.Printf("\tLeader: %v, term: %v commitIndex: %v votedFor: %v\n",
        s.Leader(), s.Term(), s.CommitIndex(), s.VotedFor())
    fmt.Printf("\tmembers: %v, islogEmpty: %v\n", s.MemberCount(), s.IsLogEmpty())
    fmt.Printf("\tpeers: %v\n", s.Peers())
    fmt.Printf("\tElectionTimeout: %v, HeartbeatInterval: %v\n",
        s.ElectionTimeout(), s.HeartbeatInterval())
}
