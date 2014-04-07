package raft

import (
	"math"
	"math/rand"
	"time"
)

type Envelope struct {
	SourceId int
	DestId   int
	MsgId    int64
	Msg      interface{}
}

func newEnvelope(sourceId int, destId int, msg interface{}) Envelope {
	//generate message id
	rand.Seed(time.Now().UnixNano())
	msgId := rand.Int63n(math.MaxInt64)
	return Envelope{
		SourceId: sourceId,
		DestId:   destId,
		MsgId:    msgId,
		Msg:      msg,
	}
}
