package failsafe

import (
	"github.com/goraft/raft"
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
	s.stats["raftStateChange"] = s.stats["raftStateChange"].(int) + 1
}

func (s *Server) raftLeaderChange(e raft.Event) {
	s.stats["raftLeaderChange"] = s.stats["raftLeaderChange"].(int) + 1
}

func (s *Server) raftTermChange(e raft.Event) {
	s.stats["raftTermChange"] = s.stats["raftTermChange"].(int) + 1
}

func (s *Server) raftCommit(e raft.Event) {
	s.stats["raftCommit"] = s.stats["raftCommit"].(int) + 1
}

func (s *Server) raftAddPeer(e raft.Event) {
	s.stats["raftAddPeer"] = s.stats["raftAddPeer"].(int) + 1
}

func (s *Server) raftRemovePeer(e raft.Event) {
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
	v := s.stats["raftElectionTimeoutThreshold"].(int) + 1
	s.stats["raftElectionTimeoutThreshold"] = v
}
