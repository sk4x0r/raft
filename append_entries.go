package raft

type AppendEntriesRequest struct {
	Term         int64      //leader's term
	LeaderId     int        //so follower can redirect clients
	PrevLogIndex int64      //index of log entry immediately preceding new ones
	PrevLogTerm  int64      //term of prevLogIndex entry
	Entries      []LogItem //log items to store
	LeaderCommit int64      //leader’s commitIndex
}

func newAppendEntriesRequest(term int64, leaderId int, prevLogIndex int64, prevLogTerm int64,
	leaderCommit int64, entries []LogItem) AppendEntriesRequest {
	return AppendEntriesRequest{
		Term:         term,
		LeaderId:     leaderId,
		PrevLogIndex: prevLogIndex,
		PrevLogTerm:  prevLogTerm,
		LeaderCommit: leaderCommit,
		Entries:      entries,
	}
}

type AppendEntriesResponse struct {
	Term        int64 //currentTerm, for leader to update itself
	Success     bool  //true if follower contained entry matching prevLogIndex and prevLogTerm
	PeerId      int //id of client
	CommitIndex int64 // commit index of client
}

func newAppendEntriesResponse(term int64, success bool, peerId int, commitIndex int64) AppendEntriesResponse {
	return AppendEntriesResponse{
		Term:        term,
		Success:     success,
		PeerId:      peerId,
		CommitIndex: commitIndex,
	}
}
