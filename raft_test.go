package raft

import (
	//"fmt"
	"testing"
	"time"
	"log"
	"strconv"
	"fmt"
)

func findLeader(servers map[int]Server) (Server,error){
	for true{
		for _,server:=range servers{
			if server.Leader(){
				return server,nil
			}
		}
	}
	//this should never execute
	var s Server
	return s, fmt.Errorf("Error:NoLeader")
}

func TestRaft(t *testing.T) {
	s1:=New(1001,PATH_TO_CONFIG)
	log.Println(s1.Majority())
	s1.Start()
	s2:=New(1002,PATH_TO_CONFIG)
	s2.Start()
	s3:=New(1003,PATH_TO_CONFIG)
	s3.Start()
	/*
	time.Sleep(10*time.Second)
	if s1.Leader(){
		s1.Stop()
		s1.Start()
	}
	
	if s2.Leader(){
		s2.Stop()
		s2.Start()
	}
	if s3.Leader(){
		s3.Stop()
		s3.Start()
	}
	*/
	
	time.Sleep(5*time.Second)
	servers:=make(map[int]Server)
	servers[1001]=s1
	servers[1002]=s2
	servers[1003]=s3
	for i:=0;i<10;i++{
		leader,err:=findLeader(servers)
		if err!=nil{
			panic("No leader")
		}
		inbox:=leader.RaftInbox()
		outbox:=leader.RaftOutbox()
		cmd:=newCommand(Put, "key"+strconv.Itoa(i),"val"+strconv.Itoa(i))
		inbox <- cmd
		rply:= <-outbox
		log.Println(rply)
	}
	time.Sleep(10*time.Second)
	time.Sleep(100*time.Second)
}
