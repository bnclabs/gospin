package failsafe

// Stats for failsafe dictionary.
type Stats map[string]interface{}

func NewStats() Stats {
	stats := make(Stats)
	stats["raftStateChange"] = 0
	stats["raftLeaderChange"] = 0
	stats["raftTermChange"] = 0
	stats["raftCommit"] = 0
	stats["raftAddPeer"] = 0
	stats["raftRemovePeer"] = 0
	stats["raftHeartbeat"] = 0
	stats["raftHeartbeatInterval"] = 0
	stats["raftElectionTimeoutThreshold"] = 0
	return stats
}
