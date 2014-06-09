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
var host = "localhost"
var port = "4000"
var servAddr = "http://" + host + ":" + port
var lisAddr = host + ":" + port
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

func startTestServer(servdir string) (*Server, net.Listener, *http.Server) {
	log.SetOutput(ioutil.Discard)
	if v, ok := activeServers[servdir]; ok {
		return v[0].(*Server), v[1].(net.Listener), v[2].(*http.Server)
	}

	if err := os.RemoveAll(servdir); err != nil {
		log.Fatal(err)
	}

	mux := http.NewServeMux()
	srv, err := NewServer(servdir, host, port, mux)
	if err != nil {
		log.Fatal(err)
	}
	srv.Install("")

	daemon := &http.Server{Addr: lisAddr, Handler: mux}

	lis, err := net.Listen("tcp", lisAddr)
	if err != nil {
		log.Fatal(err)
	}
	go daemon.Serve(lis)
	activeServers[servdir] = []interface{}{srv, lis, daemon}
	return srv, lis, daemon
}
