package failsafe

import (
	"github.com/goraft/raft"
	"log"
)

// TODO:
// - use this for, statistics.

// AddEventListeners to add callback for raft server.
func (s *Server) AddEventListeners() {
	rafts := s.raftServer
	rafts.AddEventListener(raft.StateChangeEventType, s.raftStateChange)
	rafts.AddEventListener(raft.LeaderChangeEventType, s.raftLeaderChange)
	rafts.AddEventListener(raft.TermChangeEventType, s.raftTermChange)
	rafts.AddEventListener(raft.CommitEventType, s.raftCommit)
	rafts.AddEventListener(raft.AddPeerEventType, s.raftAddPeer)
	rafts.AddEventListener(raft.RemovePeerEventType, s.raftRemovePeer)
	rafts.AddEventListener(raft.HeartbeatEventType, s.raftHeartbeat)
	rafts.AddEventListener(
		raft.HeartbeatIntervalEventType, s.raftHeartbeatInterval)
	rafts.AddEventListener(
		raft.ElectionTimeoutThresholdEventType, s.raftElectionTimeoutThreshold)
}

func (s *Server) raftStateChange(e raft.Event) {
	v, pv := e.Value(), e.PrevValue()
	log.Printf("%v, raftStateChange (%T) %v:%v", s.logPrefix, v, v, pv)
}

func (s *Server) raftLeaderChange(e raft.Event) {
	v, pv := e.Value(), e.PrevValue()
	log.Printf("%v, raftLeaderChange (%T) %v:%v", s.logPrefix, v, v, pv)
}

func (s *Server) raftTermChange(e raft.Event) {
	v, pv := e.Value(), e.PrevValue()
	log.Printf("%v, raftTermChange (%T) %v:%v", s.logPrefix, v, v, pv)
}

func (s *Server) raftCommit(e raft.Event) {
	value := e.Value()
	index, term, name, _ := logEntry(value.(*raft.LogEntry))
	log.Printf("%v, raftCommit %v, %v, %v", s.logPrefix, index, term, name)
}

func (s *Server) raftAddPeer(e raft.Event) {
	v, pv := e.Value(), e.PrevValue()
	log.Printf("%v, raftAddPeer (%T) %v:%v", s.logPrefix, v, v, pv)
}

func (s *Server) raftRemovePeer(e raft.Event) {
	v, pv := e.Value(), e.PrevValue()
	log.Printf("%v, raftRemovePeer (%T) %v:%v", s.logPrefix, v, v, pv)
}

func (s *Server) raftHeartbeat(e raft.Event) {
	v, pv := e.Value(), e.PrevValue()
	log.Printf("%v, raftHeartbeat (%T) %v:%v", s.logPrefix, v, v, pv)
}

func (s *Server) raftHeartbeatInterval(e raft.Event) {
	v, pv := e.Value(), e.PrevValue()
	log.Printf("%v, raftHeartbeatInterval (%T) %v:%v", s.logPrefix, v, v, pv)
}

func (s *Server) raftElectionTimeoutThreshold(e raft.Event) {
	v, pv := e.Value(), e.PrevValue()
	log.Printf(
		"%v, raftElectionTimeoutThreshold (%T) %v:%v", s.logPrefix, v, v, pv)
}

func logEntry(entry *raft.LogEntry) (uint64, uint64, string, []byte) {
	return entry.Index(), entry.Term(), entry.CommandName(), entry.Command()
}
