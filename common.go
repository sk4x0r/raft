package raft

import (
	//"flag"
	//"fmt"
	//"testing"
	"bytes"
	"encoding/gob"
	"log"
	"math/rand"
	"time"
	//"github.com/sk4x0r/cluster"
)

const (
	BROADCAST          = -1
	PATH_TO_CONFIG     = "config.json"
	ELECTION_TIMEOUT   = 300 //in millisecond
	HEARTBEAT_INTERVAL = 50 //in millisecond
)

func heartbeatInterval() time.Duration {
	return time.Duration(HEARTBEAT_INTERVAL) * time.Millisecond
}

func electionTimeout() time.Duration {
	rand.Seed(time.Now().UnixNano())
	return time.Duration((ELECTION_TIMEOUT + rand.Intn(ELECTION_TIMEOUT))) * time.Millisecond
}

//cite:http://blog.golang.org/gobs-of-data
//cite: Pushkar Khadilkar
func gobToEnvelope(gobbed []byte) Envelope {
	buf := bytes.NewBuffer(gobbed)
	dec := gob.NewDecoder(buf)
	var envelope Envelope
	dec.Decode(&envelope)
	return envelope
}

//cite: http://blog.golang.org/gobs-of-data
//cite: Pushkar Khadilkar
func envelopeToGob(envelope Envelope) []byte {
	//log.Println("envelopeToGob-->",envelope)
	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	gob.Register(AppendEntriesRequest{})
	gob.Register(AppendEntriesResponse{})
	gob.Register(RequestVoteRequest{})
	gob.Register(RequestVoteResponse{})
	gob.Register(Command{})
	gob.Register(LogItem{})
	err := enc.Encode(envelope)
	if err != nil {
		log.Fatal("encode error:", err,envelope)
	}
	return buf.Bytes()
}
