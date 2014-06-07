package failsafe

import(
    "github.com/goraft/raft"
    "log"
)

func (s *Server) AddEventListeners() {
    rafts := s.raftServer
    rafts.AddEventListener(raft.StateChangeEventType, s.RaftStateChange)
    rafts.AddEventListener(raft.LeaderChangeEventType, s.RaftLeaderChange)
    rafts.AddEventListener(raft.TermChangeEventType, s.RaftTermChange)
    rafts.AddEventListener(raft.CommitEventType, s.RaftCommit)
    rafts.AddEventListener(raft.AddPeerEventType, s.RaftAddPeer)
    rafts.AddEventListener(raft.RemovePeerEventType, s.RaftRemovePeer)
    rafts.AddEventListener(raft.HeartbeatEventType, s.RaftHeartbeat)
    rafts.AddEventListener(
        raft.HeartbeatIntervalEventType, s.RaftHeartbeatInterval)
    rafts.AddEventListener(
        raft.ElectionTimeoutThresholdEventType, s.RaftElectionTimeoutThreshold)
}

func (s *Server) RaftStateChange(e raft.Event) {
    v, pv := e.Value(), e.PrevValue()
    log.Printf("%v, RaftStateChange (%T) %v:%v", s.logPrefix, v, v, pv)
}

func (s *Server) RaftLeaderChange(e raft.Event) {
    v, pv := e.Value(), e.PrevValue()
    log.Printf("%v, RaftLeaderChange (%T) %v:%v", s.logPrefix, v, v, pv)
}

func (s *Server) RaftTermChange(e raft.Event) {
    v, pv := e.Value(), e.PrevValue()
    log.Printf("%v, RaftTermChange (%T) %v:%v", s.logPrefix, v, v, pv)
}

func (s *Server) RaftCommit(e raft.Event) {
    value := e.Value()
    index, term, name, _ := logEntry(value.(*raft.LogEntry))
    log.Printf("%v, RaftCommit %v, %v, %v", s.logPrefix, index, term, name)
}

func (s *Server) RaftAddPeer(e raft.Event) {
    v, pv := e.Value(), e.PrevValue()
    log.Printf("%v, RaftAddPeer (%T) %v:%v", s.logPrefix, v, v, pv)
}

func (s *Server) RaftRemovePeer(e raft.Event) {
    v, pv := e.Value(), e.PrevValue()
    log.Printf("%v, RaftRemovePeer (%T) %v:%v", s.logPrefix, v, v, pv)
}

func (s *Server) RaftHeartbeat(e raft.Event) {
    v, pv := e.Value(), e.PrevValue()
    log.Printf("%v, RaftHeartbeat (%T) %v:%v", s.logPrefix, v, v, pv)
}

func (s *Server) RaftHeartbeatInterval(e raft.Event) {
    v, pv := e.Value(), e.PrevValue()
    log.Printf("%v, RaftHeartbeatInterval (%T) %v:%v", s.logPrefix, v, v, pv)
}

func (s *Server) RaftElectionTimeoutThreshold(e raft.Event) {
    v, pv := e.Value(), e.PrevValue()
    log.Printf(
        "%v, RaftElectionTimeoutThreshold (%T) %v:%v", s.logPrefix, v, v, pv)
}

func logEntry(entry *raft.LogEntry) (uint64, uint64, string, []byte) {
    return entry.Index(), entry.Term(), entry.CommandName(), entry.Command()
}
