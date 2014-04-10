package raft

import (
	//"fmt"
	"log"
	"time"
	"io/ioutil"
	"encoding/json"
	sortutil "github.com/cznic/sortutil"
	zmq "github.com/pebbe/zmq4"
	"strconv"
	"sync"
	"github.com/syndtr/goleveldb/leveldb"
)

//states of server
const (
	Stopped   = "stopped"
	Follower  = "follower"
	Candidate = "candidate"
	Leader    = "leader"
)

type Server struct {
	id          int
	port        int
	state       string
	currentTerm int64
	peers       []int
	votedFor    int
	leader      int

	peerInfo    map[int]Peer
	connections map[int]*zmq.Socket

	inbox       chan Envelope
	outbox      chan Envelope
	raftInbox   chan Command
	raftOutbox  chan Response
	stopServer  chan chan bool
	stopInbox   chan chan bool
	stopOutbox  chan chan bool
	mutex       sync.RWMutex
	log         Log
	peersInSync map[int]bool
	db *leveldb.DB
	//kvstore *leveldb.DB
}

func (s *Server) Term() int64 {
	return s.currentTerm
}

func (s *Server) Leader() bool {
	return s.State() == Leader
}

func (s *Server) LeaderId() int{
	return s.leader
}

func (s *Server) Id() int {
	return s.id
}

func (s *Server) Peers() []int {
	return s.peers
}

func (s *Server) Outbox() chan Envelope {
	return s.outbox
}

func (s *Server) Inbox() chan Envelope {
	return s.inbox
}

func (s *Server) RaftOutbox() chan Response {
	return s.raftOutbox
}

func (s *Server) RaftInbox() chan Command {
	return s.raftInbox
}

func (s *Server) Port() int {
	return s.port
}

func (s *Server) Majority() int {
	return (len(s.peers)+1)/2 + 1
}

//TODO: error handling
func New(id int, configFile string) Server {
	//log.Println("Server.New(",id,configFile,")")
	conf := parseConfigFile(configFile)
	//log.Println(conf)
	//kvstore,_:=leveldb.OpenFile(strconv.Itoa(id)+"kvstore", nil)
	s := Server{
		id:          id,
		port:        conf.getPort(id),
		state:       Stopped,
		currentTerm: loadTermFromDisk(id),
		peers:       conf.getPeers(id),
		inbox:       make(chan Envelope,10),
		outbox:      make(chan Envelope,10),
		raftInbox:   make(chan Command,10),
		raftOutbox:  make(chan Response,10),
		stopServer:  make(chan chan bool),
		stopInbox:   make(chan chan bool),
		stopOutbox:  make(chan chan bool),
		votedFor:    0,
		leader:      0,
		peerInfo:    conf.getPeerInfo(id),
		log:         newLog(id),
		//kvstore:kvstore,
	}
	s.initializeSockets()
	return s
}


func (s *Server) Start() {
	//log.Println(s.Id(),s.Term(),s.State(),"Server.Start()")
	//TODO: check if server is already started
	registerGob()
	db,_:=leveldb.OpenFile(strconv.Itoa(s.Id())+"db", nil)
	s.db=db
	s.setState(Follower)
	go s.handleInbox()
	go s.handleOutbox()
	go s.mainLoop()
	//log.Println(s.Id(),s.State(),"started------------->")
}


func (s *Server) Stop() {
	s.db.Close()
	serverStopped := make(chan bool)	
	//log.Println(s.Id(),s.Term(),s.State(),s.State(),"Server.Stop()--------->")
	s.stopServer <- serverStopped
	//inboxStopped := make(chan bool)
	//s.stopInbox <- inboxStopped
	//outboxStopped := make(chan bool)
	//s.stopOutbox <- outboxStopped
	//wait till everything closes
	//<-serverStopped
	//<-inboxStopped
	//<-outboxStopped
	s.state = Stopped
	s.leader=0
	//keep discarting messages on raftinbox till server resumes
	//go s.handleStoppedState()
	//log.Println(s.Id(),s.State(),"server stopped")
}

func (s *Server) handleStoppedState(){
	raftInbox:=s.RaftInbox()
	raftOutbox:=s.RaftOutbox()
	for s.State()==Stopped{
		select{
			case <-raftInbox:
				raftOutbox<-newResponse(Error,0,"")
			case <-time.After(100*time.Millisecond):
				
			}
	}
}

func (s *Server) initializeSockets() {
	//log.Println(s.Id(),s.Term(),s.State(),s.State(),"Server.initializeSockets()")
	s.connections = make(map[int]*zmq.Socket)
	for i := range s.peers {
		peerId := s.peers[i]

		sock, err := zmq.NewSocket(zmq.PUSH)
		if err != nil {
			log.Panicln("Error creating socket", err)
		}
		sockAddr := "tcp://" + s.peerInfo[peerId].Ip + ":" + strconv.Itoa(s.peerInfo[peerId].Port)
		sock.Connect(sockAddr)
		s.connections[peerId] = sock
		//s.peerInfo[peerId].soc= s.connections[peerId]
	}
}

func (s *Server) handleInbox() {
	//log.Println(s.Id(),s.Term(),s.State(),s.State(),"Server.handleInbox()")

	//create a socket to respond
	responder, err := zmq.NewSocket(zmq.PULL)
	if err != nil {
		log.Panicln("Error creating socket", err)
	}
	defer responder.Close()

	bindAddress := "tcp://*:" + strconv.Itoa(s.port)
	responder.Bind(bindAddress)
	//log.Println(s.Id(),"Started listening on port",s.port)
	responder.SetRcvtimeo(100 * time.Millisecond) //TODO:adjust this

	//keep looping till..
	for s.State() != Stopped {
		//log.Println(s.Id(),s.Term(),s.State(),s.State(),"Server.handleInbox()-->looping")

		select {
		//.. recieves kill signal
		/*
		case inboxStopped := <-s.stopInbox:
			//log.Println(s.Id(),"stopping inbox")
			//TODO: closing work

			// and then send confirmation
			inboxStopped <- true
			//log.Println(s.Id(),"returning")
			return
		*/
		//.. receives request
		default:
			req, err := responder.RecvBytes(0)
			//if something goes wrong while receiving.. neglect the request
			if err != nil {
				break
			}
			//otherwise.. (if everything goes well) ..forward request to inbox
			envelope := gobToEnvelope(req)
			s.inbox <- envelope
		}
	}
}

func (s *Server) handleOutbox() {
	//log.Println(s.Id(),s.Term(),s.State(),s.State(),"Server.handleOutbox()")
	//keep looping till...
	for s.State()!=Stopped{
		select {
		//receives kill signal
		//case outboxStopped := <-s.stopOutbox:
			//TODO: closing work

			//and then send confirmation
			//outboxStopped <- true
			////log.Println(s.Id(),s.Term(),,s.State(),"handleOutbox stopped")
			//return
		//if message is received on outbox
		case envelope := <-s.outbox:
			if envelope.DestId == BROADCAST {
				time.Sleep(50 * time.Millisecond)
				for _, conn := range s.connections {
					//TODO: insert individual pids in each envelope
					msg := envelopeToGob(envelope)
					conn.SendBytes(msg, 0)
				}
			}else{
				peerId := envelope.DestId
				conn := s.connections[peerId]
				msg := envelopeToGob(envelope)
				conn.SendBytes(msg, 0)
			}
		case <-time.After(200*time.Millisecond):
			
		}
	}
}


func(s *Server) persistTermOnDisk(term int64){
	fileName:=strconv.Itoa(s.Id())+".term"
	
	termJson:=TermJson{term}
	
	fileBytes, err := json.Marshal(termJson)
	if err!=nil{
		panic("Error while marshalling"+err.Error())
	}
	
	err=ioutil.WriteFile(fileName, fileBytes, 0644)
	if err!=nil{
		panic("Error writing to disk:"+ err.Error())
	}
	s.currentTerm=term
}


//This method increments the current term.
//It doesn't handle leader and state changes.
//CAUTION:Should only be called from candidateLoop()
func (s *Server) incrementTerm(){
	//log.Println(s.Id(),s.Term(),s.State(),"Server.incrementTerm()")
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.currentTerm = s.currentTerm+1
	s.persistTermOnDisk(s.currentTerm)
}

//updates current term and changes state to follower
func (s *Server) updateCurrentTerm(term int64, leaderId int) {
	//log.Println(s.Id(),s.Term(),s.State(),"Server.updateCurrentTerm(",term,leaderId,")")
	s.mutex.Lock()
	defer s.mutex.Unlock()
	if s.state == Leader {
		//TODO:stop heartbeats
		//WORKAROUND: No need to stop heartbeats in current implementation, startHeartbeats() method takes care of checking leader
	}

	if s.state != Follower {
		s.setState(Follower)
	}

	s.currentTerm = term
	s.leader = leaderId
	s.votedFor = 0
	s.persistTermOnDisk(term)
}

func (s *Server) processAppendEntriesRequest(req AppendEntriesRequest) AppendEntriesResponse {
	//log.Println(s.Id(),s.Term(),s.State(),"Server.processAppendEntriesRequest()",req)
	if req.Term < s.currentTerm {
		//older term, rejecting
		return newAppendEntriesResponse(s.currentTerm, false, s.Id(), s.log.CommitIndex())
	}

	if req.Term == s.currentTerm {
		if s.state == Leader {
			//this should never happen
			panic("Multiple leaders for the same term")
		}
		//change state to follower, update the leader id
		s.mutex.Lock()
		s.state = Follower
		s.leader = req.LeaderId
		s.mutex.Unlock()
	} else {
		s.updateCurrentTerm(req.Term, req.LeaderId)
	}

	// Reject if log doesn't contain a matching previous entry
	if err := s.log.truncate(req.PrevLogIndex, req.PrevLogTerm); err != nil {
		return newAppendEntriesResponse(s.currentTerm, false, s.Id(), s.log.CommitIndex())
	}

	// Append entries to the log
	//Current implementation of appendEntries always returns nil
	if err := s.log.appendEntries(req.Entries); err != nil {
		return newAppendEntriesResponse(s.currentTerm, false, s.Id(), s.log.CommitIndex())
	}

	// Commit up to the commit index
	//Current implementation of setCommitIndex always returns nil
	//TODO: move the code below to func setCommitIndex()
	prevCommitIndex:=s.log.CommitIndex()
	commitIndex:=req.LeaderCommit
	if prevCommitIndex < commitIndex{
		//log.Println(s.Id(),s.State(),"Recvd commit idx:",commitIndex)
		sliceToCommit:=s.log.entries[prevCommitIndex+1:commitIndex+1]
		s.persistEntries(sliceToCommit)
	}
	if err := s.log.setCommitIndex(req.LeaderCommit); err != nil {
		return newAppendEntriesResponse(s.currentTerm, false, s.Id(), s.log.CommitIndex())
	}

	//if everything goes well, send success
	//log.Println(s.Id(),s.State(),"Sending back response commit idx:",s.log.CurrentIndex())
	return newAppendEntriesResponse(s.currentTerm, true, s.Id(), s.log.CurrentIndex())//CAUTION: sending currentIndex instead of commitIndex
}

func (s *Server) processRequestVoteRequest(req RequestVoteRequest) RequestVoteResponse {
	//log.Println(s.Id(),s.Term(),s.State(),"Server.processRequestVoteRequest()",req)

	// If the request is coming from an old term then reject the vote
	if req.Term < s.Term() {
		//log.Println(s.Id(),s.Term(),s.State(),"processRqVoteRqst() Request is coming from an older term",req)
		return newRequestVoteResponse(s.currentTerm, false)
	}

	//If term from request is greater than current term then update current term
	if req.Term > s.Term() {
		s.updateCurrentTerm(req.Term, 0)
	} else if s.votedFor != 0 {
		//else if vote is already casted for current term then reject the vote
		//log.Println(s.Id(),s.Term(),s.State(),"processRqVoteRqst() Already voted for this term",req)
		return newRequestVoteResponse(s.currentTerm, false)
	}

	// If the candidate's log is not at least as up to date as our last log then reject the vote
	//TODO: remove lastInfo()
	lastIndex, lastTerm := s.log.lastInfo()
	if lastIndex > req.LastLogIndex || lastTerm > req.LastLogTerm {
		//log.Println(s.Id(),s.Term(),s.State(),"processRqVoteRqst() Candidate's log is not upto date",req)
		return newRequestVoteResponse(s.currentTerm, false)
	}

	// if no error happens till this place, then cast the vote
	s.votedFor = req.CandidateId
	return newRequestVoteResponse(s.currentTerm, true)
}

func (s *Server) processRequestVoteResponse(req RequestVoteResponse) bool {
	//log.Println(s.Id(),s.Term(),s.State(),"Server.processRequestVoteResponse()",req)
	if req.Term == s.Term() && req.VoteGranted == true {
		return true
	}
	return false
}

func (s *Server) followerLoop() {
	//log.Println(s.Id(),s.Term(),s.State(),"Server.followerLoop()")
	for s.State() == Follower {
		//log.Println(s.Id(),s.Term(),s.State(),"Server.followerLoop()-->looping")
		select {
		
		case serverStopped := <-s.stopServer:
			//TODO: do cleaning
			s.setState(Stopped)
			//and send confirmation
			serverStopped <- true
			return
		
		case envelope := <-s.inbox:
			//log.Println(s.Id(),s.Term(),s.State(),"Server.followerLoop(), recvd msg on inbox",envelope)
			switch req := envelope.Msg.(type) {
			case AppendEntriesRequest:
				//log.Println(s.Id(),s.Term(),s.State(),"Server.followerLoop(), appendEntriesRequest recvd on inbox",req)
				resp := s.processAppendEntriesRequest(req)
				//log.Println(s.Id(),s.Term(),s.State(),"Server.followerLoop(), sending appendEntriesResponse",resp)
				e := newEnvelope(s.Id(), envelope.SourceId, resp)
				outbox := s.Outbox()
				outbox <- e
			case RequestVoteRequest:
				//log.Println(s.Id(),s.Term(),s.State(),"Server.followerLoop(), RequestVoteRequest recvd on inbox",req)
				resp := s.processRequestVoteRequest(req)
				//log.Println(s.Id(),s.Term(),s.State(),"Server.followerLoop(), sending RequestVoteResponse",resp)
				e := newEnvelope(s.Id(), envelope.SourceId, resp)
				outbox := s.Outbox()
				outbox <- e
			}
		case command:= <-s.raftInbox:
				//log.Println(s.Id(),s.Term(),s.State(),"Server.followerLoop(), Command recvd on raftinbox",command)
				resp:=s.processCommand(command)
				//log.Println(s.Id(),s.Term(),s.State(),"Server.followerLoop(), sending Response for command",resp)
				raftOutbox:=s.RaftOutbox()
				raftOutbox <- resp
		case <-time.After(electionTimeout()): //TODO: check whether this is electionTimeout or heartbeatTimeout
			//log.Println(s.Id(),s.Term(),s.State(),"Server.followerLoop(), timed out, going for election")
			s.setState(Candidate)
		}
	}
}

func (s *Server) createVoteRequest() Envelope {
	//log.Println(s.Id(),s.Term(),s.State(),"Server.createVoteRequest()")
	//create a broadcast message asking for vote
	lastLogIndex, lastLogTerm := s.log.lastInfo()
	req := newRequestVoteRequest(s.Term(), s.Id(), lastLogIndex, lastLogTerm)
	e := newEnvelope(s.Id(), BROADCAST, req)
	return e
}

func (s *Server) candidateLoop() {
	//log.Println(s.Id(),s.Term(),s.State(),"Server.candidateLoop()")
	firstIteration := true
	var voteCount int
	var timeout <-chan time.Time
	for s.State() == Candidate {
		//log.Println(s.Id(),s.Term(),s.State(),"Server.candidateLoop()-->looping")
		if firstIteration {
			//increment current term, and set leader to none
			s.incrementTerm()
			//vote for self
			s.votedFor = s.Id()

			//create vote requests
			//log.Println(s.Id(),s.State(), "Creating vote requests")
			voteRequest := s.createVoteRequest()

			//send vote requests
			outbox := s.Outbox()
			outbox <- voteRequest
			
			//log.Println(s.Id(),s.State(), "Sent vote requests")
			//initialize vote count with self vote
			voteCount = 1
			//start timer
			timeout = time.After(electionTimeout())
			firstIteration = false
		}

		select {
		case serverStopped := <-s.stopServer:
			//TODO: do cleaning
			s.setState(Stopped)
			//and send confirmation
			serverStopped <- true
			return

		case envelope := <-s.inbox:
			switch req := envelope.Msg.(type) {
			case AppendEntriesRequest:
				//log.Println(s.Id(),s.Term(),s.State(),"Server.candidateLoop(), appendEntriesRqst recvd on inbox",req)
				resp := s.processAppendEntriesRequest(req)
				//log.Println(s.Id(),s.Term(),s.State(),"Server.candidateLoop(), sending appendEntriesResponse",resp)
				e := newEnvelope(s.Id(), envelope.SourceId, resp)
				outbox := s.Outbox()
				outbox <- e
			case RequestVoteRequest:
				//log.Println(s.Id(),s.Term(),s.State(),"Server.candidateLoop(), RequestVoteRqst recvd on inbox",req)
				s.processRequestVoteRequest(req)
				resp := s.processRequestVoteRequest(req)
				//log.Println(s.Id(),s.Term(),s.State(),"Server.candidateLoop(), sending RequestVoteResponse",resp)
				e := newEnvelope(s.Id(), envelope.SourceId, resp)
				outbox := s.Outbox()
				outbox <- e
			case RequestVoteResponse:
				//log.Println(s.Id(),s.Term(),s.State(),"Server.candidateLoop(), RequestVoteResponse recvd on inbox",req)
				if s.processRequestVoteResponse(req) {
					voteCount++
					if voteCount >= s.Majority() {
						s.setState(Leader)
						return
					}
				}
			}
		case command:= <-s.raftInbox:
				//log.Println(s.Id(),s.Term(),s.State(),"Server.candidateLoop(), Command recvd on raftinbox",command)
				resp:=s.processCommand(command)
				//log.Println(s.Id(),s.Term(),s.State(),"Server.candidateLoop(), sending command response",resp)
				raftOutbox:=s.RaftOutbox()
				raftOutbox <- resp
		case <-timeout:
			//log.Println(s.Id(),s.Term(),s.State(),"Server.candidateLoopLoop(), timed out, going for reelection")
			firstIteration = true
		}
	}
}

func (s *Server) processCommand(cmd Command) Response {
	//log.Println(s.Id(),s.Term(),s.State(),s.State(),"Server.processCommand(",cmd,")")
	if s.State()!=Leader{
		//log.Println(s.Id(),s.Term(),s.State(),"Server.processCommand()--> not a leader")
		return newResponse(Redirect, s.LeaderId(), "")
	}
	switch cmd.Cmd{
		//if command is get, send the result
		case Get:
			data, err := s.db.Get([]byte(cmd.Key), nil)
			if err!=nil{
				return newResponse(Error, s.LeaderId(), "")
			}else{
				return newResponse(Ok, s.LeaderId(), string(data))
			}
		// for put and delete, command should be executed after it gets replicated on majority of the servers
		case Put:
			entry := s.log.newLogEntry(s.Term(), cmd) //TODO: error handling
			s.log.appendEntry(entry)                  //TODO:error handling
			return newResponse(Ok, s.LeaderId(), "")//TODO: return only after entry is committed
		case Delete:
			entry := s.log.newLogEntry(s.Term(), cmd) //TODO: error handling
			s.log.appendEntry(entry)                  //TODO:error handling
			return newResponse(Ok, s.LeaderId(), "")//TODO: return only after entry is committed
	}
	//TODO:handle this
	return newResponse(Error, s.LeaderId(), "")
}

func (s *Server) processAppendEntriesResponse(resp AppendEntriesResponse) {
	//log.Println(s.Id(),s.Term(),s.State(),"Server.processAppendEntriesResponse()", resp)
	if resp.Term > s.Term() {
		s.updateCurrentTerm(resp.Term, 0)
		return
	}
	//TODO: check this code for correctness
	if resp.Success == false {
		prevCommit := resp.CommitIndex
		entries := s.log.getEntriesAfter(prevCommit)
		var prevLogTerm int64
		if prevCommit==-1 {
			prevLogTerm=-1
		}else{
			prevLogTerm = s.log.entries[prevCommit].Term //TODO: check this
		}
		req := newAppendEntriesRequest(s.Term(), s.Id(), prevCommit, prevLogTerm, s.log.CommitIndex(), entries)
		e := newEnvelope(s.Id(), resp.PeerId, req)
		outbox := s.Outbox()
		outbox <- e
		return
	}

	//if resp.Success == true 
	//log.Println("Setting peer in sync for",resp.PeerId)
	s.peersInSync[resp.PeerId] = true
	p := s.peerInfo[resp.PeerId]
	//log.Println("Setting prevlogindex for",resp.PeerId,"as",resp.CommitIndex)
	p.setPrevLogIndex(resp.CommitIndex)
	s.peerInfo[resp.PeerId]=p
	
	//log.Println("confirmation:p.prevLogIndex=")
	p=s.peerInfo[resp.PeerId]
	//log.Println(p.getPrevLogIndex())
	if len(s.peersInSync) < s.Majority() {
		//some more peers need before starting commits
		return
	}
	//else..commit

	//TODO: write sort interface for int64 and remove sortutil
	indices := make(sortutil.Int64Slice, 0)
	indices = append(indices, s.log.CurrentIndex())
	//log.Println(s.log.CurrentIndex())
	for _, peer := range s.peerInfo {
		indices = append(indices, peer.getPrevLogIndex())
	}
	//log.Println(s.Id(),indices)
	indices.Sort()
	
	//log.Println(s.Id(),indices)
	commitIndex := indices[s.Majority()-1]
	prevCommitIndex := s.log.commitIndex

	if commitIndex > prevCommitIndex {
		s.log.setCommitIndex(commitIndex)
		sliceToCommit:=s.log.entries[prevCommitIndex+1:commitIndex+1]
		s.persistEntries(sliceToCommit)
	}
}

//TODO: error handling
func (s *Server) persistEntries(entries []LogItem){
	//log.Println(s.Id(), "adding entries to db:",entries)
	for _, entry:=range entries{
		entry_str,_:=json.Marshal(entry)
		s.db.Put([]byte(strconv.FormatInt(entry.Index,10)), []byte(entry_str), nil)
	}
}

func (s *Server) startHeartbeats() {
	//log.Println(s.Id(),s.Term(),s.State(),"Server.startHeartbeats()")
	for s.State() == Leader {
		select {
		case <-time.After(heartbeatInterval()):
			//log.Println(s.Id(),s.Term(),s.State(),"Sending Heartbeats")
			s.mutex.RLock()
			for _, peer := range s.peerInfo {
				hb := peer.generateHeartbeat()
				e := newEnvelope(s.Id(), peer.Pid, hb)
				outbox := s.Outbox()
				outbox <- e
			}
			
			s.mutex.RUnlock()
		}
	}
}

func (s *Server) leaderLoop() {
	//log.Println(s.Id(),s.Term(),s.State(),"Server.leaderLoop()")
	logIndex, _ := s.log.lastInfo()
	for _, peer := range s.peerInfo {
		peer.setPrevLogIndex(logIndex)
	}
	go s.startHeartbeats()
	for s.State() == Leader {
		//log.Println(s.Id(),s.Term(),s.State(),"Server.leaderLoop() --> looping")
		select {
		case serverStopped := <-s.stopServer:
			//log.Println(s.Id(),s.Term(),s.State(),"Server.leaderLoop() --> case serverStopped")
			//TODO: do cleaning
			s.setState(Stopped)
			//and send confirmation
			serverStopped <- true
			//log.Println(s.Id(),s.Term(),s.State(),"exiting leaderLoop")
			return
		case envelope := <-s.inbox:
			//log.Println(s.Id(),s.Term(),s.State(),"Server.leaderLoop() --> case envelope on inbox")
			switch req := envelope.Msg.(type) {
			case AppendEntriesRequest:
				//log.Println(s.Id(),s.Term(),s.State(),"Server.leaderLoop(), appendEntriesRqst recvd on inbox",req)
				resp := s.processAppendEntriesRequest(req)
				//log.Println(s.Id(),s.Term(),s.State(),"Server.leaderLoop(), sending reply to appendEntriesRequest",resp)
				e := newEnvelope(s.Id(), envelope.SourceId, resp)
				outbox := s.Outbox()
				outbox <- e
			case RequestVoteRequest:
				//log.Println(s.Id(),s.Term(),s.State(),"Server.leaderLoop(), RqstVoteRqst recvd on inbox",req)
				resp := s.processRequestVoteRequest(req)
				//log.Println(s.Id(),s.Term(),s.State(),"Server.leaderLoop(), sending reply to RequestVoteRequest",resp)
				e := newEnvelope(s.Id(), envelope.SourceId, resp)
				outbox := s.Outbox()
				outbox <- e
				s.processRequestVoteRequest(req)
			case AppendEntriesResponse:
				//log.Println(s.Id(),s.Term(),s.State(),"Server.leaderLoop(), appendEntriesResponse recvd on inbox",req)
				s.processAppendEntriesResponse(req)
			}
		case command:= <-s.raftInbox:
			//log.Println(s.Id(),s.Term(),s.State(),"Server.leaderLoop(), Command recvd on raftinbox",command)
			resp:=s.processCommand(command)
			//log.Println(s.Id(),s.Term(),s.State(),"Server.leaderLoop(), sending response to raftoutbox",resp)
			raftOutbox:=s.RaftOutbox()
			raftOutbox <- resp
		}
	}
	//log.Println(s.Id(),s.Term(),s.State(),"exiting leaderLoop()")
}

func (s *Server) mainLoop() {
	//log.Println(s.Id(),s.Term(),s.State(),"Server.mainLoop()")
	for s.State() != Stopped {
		switch s.State() {
		case Follower:
			s.followerLoop()
			break
		case Candidate:
			s.candidateLoop()
			break
		case Leader:
			s.leaderLoop()
			break
		}
	}
}

func (s *Server) State() string {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	state := s.state
	return state
}

//TODO: check whenever state is being updated, whether leader is also being updated or not
func (s *Server) setState(state string) {
	//log.Println(s.Id(),s.Term(),s.State(),"Server.setState(",state,")")
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.state = state
	if state == Leader {
		s.leader = s.Id()
		s.peersInSync = make(map[int]bool)
		s.peersInSync[s.Id()] = true
	}
}
