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

// RegisterCommands with raft for failsafe package.
func RegisterCommands() {
	raft.RegisterCommand(&SetCommand{})
	raft.RegisterCommand(&DeleteCommand{})
}

// Server is a combination of the Raft server and HTTP server which acts as
// the transport, also provides APIs for local application (as method
// recievers) and REST APIs for remote application to Set,Get,Delete
// failsafe data structure.
type Server struct {
	name        string
	path        string
	listernAddr string
	mux         raft.HTTPMuxer // mux can be used to chain HTTP handlers.
	raftServer  raft.Server
	db          *SafeDict
	// misc.
	logPrefix string
	stats     Stats
}

type Context struct {
	db *SafeDict
	s  *Server
}

var logLevel int = 0

// NewServer will instanstiate a new raft-server.
func NewServer(name, path, listernAddr string, mux raft.HTTPMuxer) (s *Server, err error) {
	if err = os.MkdirAll(path, 0700); err != nil {
		return nil, err
	}
	s = &Server{
		name:        name,
		path:        path,
		listernAddr: listernAddr,
		mux:         mux,
		logPrefix:   fmt.Sprintf("[failsafe.%v]", name),
		stats:       NewStats(),
	}

	if s.name == "" {
		nameFile := filepath.Join(path, "name")
		if b, err := ioutil.ReadFile(nameFile); err == nil {
			s.name = string(b)
		} else {
			s.name = fmt.Sprintf("%07x", rand.Int())[0:7]
			err := ioutil.WriteFile(nameFile, []byte(s.name), 0644)
			if err != nil {
				log.Fatal(err)
			}
		}
	}

	if s.db, err = NewSafeDict(nil, true); err != nil {
		log.Fatal(err)
	}
	return s, nil
}

// SetLogLevel to trace or debug
func SetLogLevel(level int) {
	logLevel = level
	raft.SetLogLevel(level)
}

// GetStats return statistics for this server node.
func (s *Server) GetStats() Stats {
	return s.stats
}

// GetRaftserver return raft server instance associated with this server node.
func (s *Server) GetRaftserver() raft.Server {
	return s.raftServer
}

// RemovePeers for a clean restart.
func (s *Server) RemovePeers() {
	for name, _ := range s.raftServer.Peers() {
		s.raftServer.RemovePeer(name)
	}
}

// Install to be called after NewServer(). This call will initialize and start
// a raft-server, start a cluster / join a cluster, subscribe http-handlers to
// muxer, add raft event callbacks.
func (s *Server) Install(leader string) (err error) {
	// Initialize and start Raft server.
	trans := raft.NewHTTPTransporter("/raft", 200*time.Millisecond)
	connStr := s.connectionString()
	s.raftServer, err = raft.NewServer(s.name, s.path, trans, s.db, s, connStr)
	if err != nil {
		log.Fatalf("%v, %v\n", s.logPrefix, err)
	}
	name := s.raftServer.Name()
	tracef("%s, initializing Raft Server\n", s.logPrefix)

	trans.Install(s.raftServer, s)

	// Read snapshot.
	if s.raftServer.LoadSnapshot() != nil {
		tracef("%v, loadingSnapshot %v\n", s.logPrefix, err)
	}
	s.RemovePeers()
	s.raftServer.Start()

	if leader != "" { // Join to leader if specified.
		tracef("%v, attempting to join leader %q\n", s.logPrefix, leader)
		if !s.raftServer.IsLogEmpty() {
			log.Fatalf("%v, cannot join with an existing log\n", s.logPrefix)
		}
		if err := s.selfJoin(leader); err != nil {
			log.Fatalf("%v, %v\n", s.logPrefix, err)
		}

	} else if s.raftServer.IsLogEmpty() {
		// Initialize the server by joining itself.
		tracef("%v, initializing new cluster\n", s.logPrefix)
		_, err := s.raftServer.Do(&raft.DefaultJoinCommand{
			Name:             s.raftServer.Name(),
			ConnectionString: s.connectionString(),
		})
		if err != nil {
			log.Fatalf("%v, %v\n", s.logPrefix, err)
		}

	} else {
		tracef("%v, recovered from log\n", name)
	}

	s.mux.HandleFunc("/dict", s.dbHandler)
	s.mux.HandleFunc("/join", s.joinHandler)
	s.mux.HandleFunc("/leave", s.leaveHandler)

	s.AddEventListeners()
	return
}

// HandleFunc callback for raft.
func (s *Server) HandleFunc(pattern string, handler func(http.ResponseWriter, *http.Request)) {
	s.mux.HandleFunc(pattern, handler)
}

func (s *Server) GetLeader() [2]string {
	if name := s.raftServer.Leader(); name != "" {
		if name == s.raftServer.Name() {
			return [2]string{name, s.connectionString()}
		} else if leader := s.raftServer.Peers()[name]; leader != nil {
			return [2]string{name, leader.ConnectionString}
		}
	}
	return [2]string{"", ""}
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
	s.raftServer.FlushCommitIndex()
	if err = s.raftServer.TakeSnapshot(); err != nil {
		return
	}
	s.raftServer.Stop()
	return
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

func (s *Server) connectionString() string {
	return fmt.Sprintf("http://%v", s.listernAddr)
}

func (s *Server) listenAddr() string {
	return s.listernAddr
}

func debugf(v ...interface{}) {
	if logLevel >= raft.Debug {
		format := v[0].(string)
		log.Printf(format, v[1:]...)
	}
}

func debugln(v ...interface{}) {
	if logLevel >= raft.Debug {
		log.Println(v...)
	}
}

func tracef(v ...interface{}) {
	if logLevel >= raft.Trace {
		format := v[0].(string)
		log.Printf(format, v[1:]...)
	}
}

func traceln(v ...interface{}) {
	if logLevel >= raft.Trace {
		log.Println(v...)
	}
}
