package raft

import (
	//zmq "github.com/pebbe/zmq4"
	"sync"
	//"log"
)

type Peer struct {
	Pid    int
	Ip     string
	Port   int
	server Server
	mutex  sync.RWMutex
	//soc  *zmq.Socket
	prevLogIndex int64
}

func newPeer(pid int, ip string, port int, server Server) Peer {
	return Peer{
		Pid:    pid,
		Ip:     ip,
		Port:   port,
		server: server,
	}
}

func (p *Peer) getPrevLogIndex() int64 {
	//log.Println(p.server.Id(),"Peer.getPrevLogIndex()")
	p.mutex.RLock()
	defer p.mutex.RUnlock()
	return p.prevLogIndex
}

func (p *Peer) setPrevLogIndex(value int64) {
	//log.Println(p.server.Id(),"Peer.setPrevLogIndex()")
	p.mutex.Lock()
	defer p.mutex.Unlock()
	p.prevLogIndex = value
}

func (p *Peer) getPrevLogTerm() int64 {
	//log.Println(p.server.Id(),"Peer.getPrevTerm()")
	if len(p.server.log.entries)==0{
		return -1
	}else{
		return p.server.log.entries[p.getPrevLogIndex()].Term
	}
}

func (p *Peer) generateHeartbeat() AppendEntriesRequest {
	//log.Println(p.server.Id(),"Peer.generateHeartbeat()")
	entries:=[]LogItem{}
	if len(p.server.log.entries)==0{
		entries=[]LogItem{}
	} else{
		entries= p.server.log.entries[p.getPrevLogIndex()+1:]
	}
	return newAppendEntriesRequest(p.server.Term(), p.server.Id(),
		p.getPrevLogIndex(), p.getPrevLogTerm(), p.server.log.CommitIndex(), entries)
}
