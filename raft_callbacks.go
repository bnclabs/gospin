package failsafe

import (
    "github.com/goraft/raft"
    "fmt"
    "time"
)

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
    state, oldState := e.Value().(string), e.PrevValue().(string)
    name := s.raftServer.Name()
    fmt.Printf("%v, changes state from %q to %q\n", name, oldState, state)
    s.stats["raftStateChange"] = s.stats["raftStateChange"].(int) + 1
}

func (s *Server) raftLeaderChange(e raft.Event) {
    leader, oldLeader := e.Value().(string), e.PrevValue().(string)
    name := s.raftServer.Name()
    fmt.Printf("%v, leader changed from %q to %q\n", name, oldLeader, leader)
    s.stats["raftLeaderChange"] = s.stats["raftLeaderChange"].(int) + 1
}

func (s *Server) raftTermChange(e raft.Event) {
    term, oldTerm := e.Value().(string), e.PrevValue().(string)
    name := s.raftServer.Name()
    fmt.Printf("%v, term changed from %q to %q\n", name, oldTerm, term)
    s.stats["raftTermChange"] = s.stats["raftTermChange"].(int) + 1
}

func (s *Server) raftCommit(e raft.Event) {
    s.stats["raftCommit"] = s.stats["raftCommit"].(int) + 1
}

func (s *Server) raftAddPeer(e raft.Event) {
    peer := e.Value().(string)
    name := s.raftServer.Name()
    fmt.Printf("%v, add peer %q\n", name, peer)
    s.stats["raftAddPeer"] = s.stats["raftAddPeer"].(int) + 1
}

func (s *Server) raftRemovePeer(e raft.Event) {
    peer := e.Value().(string)
    name := s.raftServer.Name()
    fmt.Printf("%v, add peer %q\n", name, peer)
    s.stats["raftRemovePeer"] = s.stats["raftRemovePeer"].(int) + 1
}

func (s *Server) raftHeartbeat(e raft.Event) {
    s.stats["raftHeartbeat"] = s.stats["raftHeartbeat"].(int) + 1
}

func (s *Server) raftHeartbeatInterval(e raft.Event) {
    v := s.stats["raftHeartbeatInterval"].(int) + 1
    s.stats["raftHeartbeatInterval"] = v
}

func (s *Server) raftElectionTimeoutThreshold(e raft.Event) {
    elapsedTime := e.Value().(time.Duration)
    name := s.raftServer.Name()
    fmt.Printf("%v, elapsed time %v\n", name, elapsedTime)
    v := s.stats["raftElectionTimeoutThreshold"].(int) + 1
    s.stats["raftElectionTimeoutThreshold"] = v
}
