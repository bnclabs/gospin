package failsafe

import (
    "bytes"
    "os"
    "encoding/json"
    "fmt"
    "github.com/goraft/raft"
    "io/ioutil"
    "log"
    "math/rand"
    "net/http"
    "path/filepath"
    "time"
)

// Server is a combination of the Raft server and HTTP server which acts as
// the transport, also provides APIs for local application (as method
// recievers) and REST APIs for remote application to Set,Get,Delete
// failsafe-dictionart.
type Server struct {
    name       string
    path       string
    host       string
    port       string
    mux        raft.HTTPMuxer
    raftServer raft.Server
    db         *SafeDict
    // gen-server
    reqch      chan []interface{}
    finch      chan bool
    // misc.
    logPrefix  string
    stats      Stats
}

// Create a new server.
func NewServer(path, host, port string, mux raft.HTTPMuxer) (s *Server, err error) {
    if err := os.MkdirAll(path, 0700); err != nil {
        return nil, err
    }

    nameFile := filepath.Join(path, "name")
    s = &Server{
        path:      path,
        host:      host,
        port:      port,
        mux:       mux,
        reqch:     make(chan []interface{}, 64), // TODO: avoid magic number
        finch:     make(chan bool),
        logPrefix: fmt.Sprintf("SafeDict server %q", path),
        stats:     make(Stats),
    }

    // Read existing name or generate a new one.
    if b, err := ioutil.ReadFile(nameFile); err == nil {
        s.name = string(b)
    } else {
        s.name = fmt.Sprintf("%07x", rand.Int())[0:7]
        if err = ioutil.WriteFile(nameFile, []byte(s.name), 0644); err != nil {
            panic(err)
        }
    }
    if s.db, err = NewSafeDict(nil, true); err != nil {
        return nil, err
    }
    return s, nil
}

// Install to be called after server NewServer().
func (s *Server) Install(leader string) (err error) {
    log.Printf("Initializing Raft Server: %s", s.path)

    // Initialize and start Raft server.
    trans := raft.NewHTTPTransporter("/raft", 200*time.Millisecond)
    s.raftServer, err = raft.NewServer(s.name, s.path, trans, nil, s.db, "")
    if err != nil {
        log.Fatal(err)
    }
    trans.Install(s.raftServer, s)
    s.raftServer.Start()

    if leader != "" { // Join to leader if specified.
        log.Println("Attempting to join leader:", leader)
        if !s.raftServer.IsLogEmpty() {
            log.Fatal("Cannot join with an existing log")
        }
        if err := s.selfJoin(leader); err != nil {
            log.Fatal(err)
        }

    } else if s.raftServer.IsLogEmpty() {
        // Initialize the server by joining itself.
        log.Println("Initializing new cluster")
        _, err := s.raftServer.Do(&raft.DefaultJoinCommand{
            Name:             s.raftServer.Name(),
            ConnectionString: s.connectionString(),
        })
        if err != nil {
            log.Fatal(err)
        }

    } else {
        log.Println("Recovered from log")
    }

    log.Println("Initializing HTTP server")

    s.mux.HandleFunc("/dict",  s.dbHandler)
    s.mux.HandleFunc("/join",  s.joinHandler)
    s.mux.HandleFunc("/leave", s.leaveHandler)

    s.AddEventListeners()

    //snapshotFile := filepath.Join(s.path, "snapshot")
    //log.Printf("%v, restoring from %q\n", s.logPrefix, snapshotFile)
    //s.db.restore(snapshotFile)
    return
}

// HandleFunc callback for raft.
func (s *Server) HandleFunc(pattern string, handler func(http.ResponseWriter, *http.Request)) {
    s.mux.HandleFunc(pattern, handler)
}

// DBGet field value located by `path` jsonpointer, does not go through
// gen-server.
func (s *Server) DBGet(path string) (value interface{}, CAS float64, err error) {
    return s.db.Get(path)
}

func (s *Server) DBSet(path string, value interface{}) (nextCAS float64, err error) {
    val, err := s.raftServer.Do(NewSetCommand(path, value, nullCAS))
    return val.(float64), err
}

func (s *Server) DBSetCAS(path string, value interface{}, CAS float64) (nextCAS float64, err error) {
    val, err := s.raftServer.Do(NewSetCommand(path, value, CAS))
    return val.(float64), err
}

func (s *Server) DBDelete(path string) (nextCAS float64, err error) {
    val, err := s.raftServer.Do(NewDeleteCommand(path, nullCAS))
    return val.(float64), err
}

func (s *Server) DBDeleteCAS(path string, CAS float64) (nextCAS float64, err error) {
    val, err := s.raftServer.Do(NewDeleteCommand(path, CAS))
    return val.(float64), err
}

func (s *Server) Stop() {
    s.raftServer.Stop()
}

func (s *Server) connectionString() string {
    return fmt.Sprintf("http://%s:%d", s.host, s.port)
}

func (s *Server) selfJoin(leader string) error {
    var b bytes.Buffer

    command := &raft.DefaultJoinCommand{
        Name:             s.raftServer.Name(),
        ConnectionString: s.connectionString(),
    }
    json.NewEncoder(&b).Encode(command)
    url := fmt.Sprintf("http://%s/join", leader)
    resp, err := http.Post(url, "application/json", &b)
    if err != nil {
        return err
    }
    resp.Body.Close()
    return nil
}
