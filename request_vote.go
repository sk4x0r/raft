package raft

type RequestVoteRequest struct {
	Term         int64 //candidate's term
	CandidateId  int   //candidate requesting vote
	LastLogIndex int64 //index of candidate's last log entry
	LastLogTerm  int64 //term of candidate's last log entry
}

type RequestVoteResponse struct {
	Term        int64 //currentTerm, for candidate to update itself
	VoteGranted bool  //true means candidate received vote
}

func newRequestVoteRequest(term int64, candidateId int, lastLogIndex int64, lastLogTerm int64) RequestVoteRequest {
	return RequestVoteRequest{
		Term:         term,
		CandidateId:  candidateId,
		LastLogIndex: lastLogIndex,
		LastLogTerm:  lastLogTerm,
	}
}

func newRequestVoteResponse(term int64, voteGranted bool) RequestVoteResponse {
	return RequestVoteResponse{
		Term:        term,
		VoteGranted: voteGranted,
	}
}
