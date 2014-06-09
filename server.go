package failsafe

import (
    "bytes"
    "encoding/json"
    "fmt"
    "github.com/goraft/raft"
    "io/ioutil"
    "log"
    "math/rand"
    "net/http"
    "os"
    "path/filepath"
    "time"
)

// Server is a combination of the Raft server and HTTP server which acts as
// the transport, also provides APIs for local application (as method
// recievers) and REST APIs for remote application to Set,Get,Delete
// failsafe data structure.
type Server struct {
    name       string
    path       string
    host       string
    port       string
    mux        raft.HTTPMuxer // mux can be used to chain HTTP handlers.
    raftServer raft.Server
    db         *SafeDict
    // misc.
    logPrefix string
    stats     Stats
}

// NewServer will instanstiate a new raft-server.
func NewServer(path, host, port string, mux raft.HTTPMuxer) (s *Server, err error) {
    if err = os.MkdirAll(path, 0700); err != nil {
        return nil, err
    }

    nameFile := filepath.Join(path, "name")
    s = &Server{
        path:      path,
        host:      host,
        port:      port,
        mux:       mux,
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

    // Read snapshot, if it exists
    filename := snapshotFile(path)
    b, err := ioutil.ReadFile(filename)
    if err != nil {
        b = nil
    }
    if s.db, err = NewSafeDict(b, true); err != nil {
        panic(err)
    }
    return s, err
}

// Install to be called after NewServer(). This call will initialize and start
// a raft-server, start a cluster / join a cluster, subscribe http-handlers to
// muxer, add raft event callbacks.
func (s *Server) Install(leader string) (err error) {
    log.Printf("Initializing Raft Server: %s", s.path)

    // Initialize and start Raft server.
    trans := raft.NewHTTPTransporter("/raft", 200*time.Millisecond)
    s.raftServer, err = raft.NewServer(s.name, s.path, trans, s.db, s.db, "")
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

    s.mux.HandleFunc("/dict", s.dbHandler)
    s.mux.HandleFunc("/join", s.joinHandler)
    s.mux.HandleFunc("/leave", s.leaveHandler)

    s.AddEventListeners()

    log.Printf("%v, restored from snapshot CAS:%v\n", s.logPrefix, s.db.GetCAS())
    return
}

// HandleFunc callback for raft.
func (s *Server) HandleFunc(pattern string, handler func(http.ResponseWriter, *http.Request)) {
    s.mux.HandleFunc(pattern, handler)
}

// DBGet field value located by `path` jsonpointer, full json-pointer spec is
// allowed.
func (s *Server) DBGet(path string) (value interface{}, CAS float64, err error) {
    return s.db.Get(path)
}

// DBSet value at the specified path, full json-pointer spec. is allowed. CAS
// is ignored.
func (s *Server) DBSet(path string, value interface{}) (nextCAS float64, err error) {
    val, err := s.raftServer.Do(NewSetCommand(path, value, nullCAS))
    if err == nil {
        return val.(float64), err
    }
    return nullCAS, err
}

// DBSetCAS value at the specified path with matching CAS, full json-pointer
// spec. is allowed.
func (s *Server) DBSetCAS(path string, value interface{}, CAS float64) (nextCAS float64, err error) {
    val, err := s.raftServer.Do(NewSetCommand(path, value, CAS))
    if err == nil {
        return val.(float64), err
    }
    return nullCAS, err
}

// DBDelete value at the specified path, last segment shall always index
// into json property. CAS is ignored.
func (s *Server) DBDelete(path string) (nextCAS float64, err error) {
    val, err := s.raftServer.Do(NewDeleteCommand(path, nullCAS))
    if err == nil {
        return val.(float64), err
    }
    return nullCAS, err
}

// DBDeleteCAS value at the specified path with matching CAS, last segment
// shall always index into json property.
func (s *Server) DBDeleteCAS(path string, CAS float64) (nextCAS float64, err error) {
    val, err := s.raftServer.Do(NewDeleteCommand(path, CAS))
    if err == nil {
        return val.(float64), err
    }
    return nullCAS, err
}

// Stop will stop the server and persist the dictionary on the disk.
func (s *Server) Stop() (err error) {
    var data []byte

    if data, err = s.db.Save(); err != nil {
        return
    }
    filename := snapshotFile(s.path)
    if err = ioutil.WriteFile(filename, data, 0644); err != nil {
        return
    }
    s.raftServer.Stop()
    return
}

func (s *Server) connectionString() string {
    return fmt.Sprintf("http://%s:%s", s.host, s.port)
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

func RegisterCommands() {
    raft.RegisterCommand(&SetCommand{})
    raft.RegisterCommand(&DeleteCommand{})
}
