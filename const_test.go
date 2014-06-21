// contains common test fixtures.

package failsafe

import (
	"github.com/goraft/raft"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"
)

var testdir = "testdata"
var testRaftdir = filepath.Join(testdir, "server/0")
var listAddr = "localhost:4000"
var servAddr = "http://" + listAddr
var smallJSONfile = filepath.Join(testdir, "small.json")
var smallJSON, _ = ioutil.ReadFile(smallJSONfile)
var dummyFile = filepath.Join(testdir, "_dummy.json")

// servdir -> {*Server, net.Listener, *http.Server}
var activeServers map[string][]interface{}

// register commands for testing and start a server.
func init() {
	raft.RegisterCommand(&SetCommand{})
	raft.RegisterCommand(&DeleteCommand{})
	activeServers = make(map[string][]interface{})
}

func startTestServer(servdir string) (*Server, net.Listener, *http.Server, error) {
	log.SetOutput(ioutil.Discard)
	if v, ok := activeServers[servdir]; ok {
		return v[0].(*Server), v[1].(net.Listener), v[2].(*http.Server), nil
	}

	if err := os.RemoveAll(servdir); err != nil {
		log.Fatal(err)
	}

	mux := http.NewServeMux()
	srv, err := NewServer("test", servdir, listAddr, mux)
	if err != nil {
		log.Fatal(err)
	}
	if err := srv.Install(""); err != nil {
		return nil, nil, nil, err
	}

	daemon := &http.Server{Addr: listAddr, Handler: mux}

	lis, err := net.Listen("tcp", listAddr)
	if err != nil {
		log.Fatal(err)
	}
	go daemon.Serve(lis)
	activeServers[servdir] = []interface{}{srv, lis, daemon}
	return srv, lis, daemon, nil
}
