package main
import (
	raft "github.com/sk4x0r/raft"
	"os"
	//"log"
	"strconv"
	"time"
	"math/rand"
)
const 	PATH_TO_CONFIG     = "config.json"

func main(){
	id, _ := strconv.Atoi((os.Args[1]))
	//log.Println("start_server:",id)
	s:=raft.New(id,PATH_TO_CONFIG)
	s.Start()
	inbox:=s.RaftInbox()
	outbox:=s.RaftOutbox()
	for i:=0;i<10000;i++{
		cmd:=newRandomCommand()
		//log.Println("sending command", cmd)
		inbox <-cmd
		<-outbox
		//log.Println("Received reply",rply)
		time.Sleep(500*time.Millisecond)
	}
}

func newRandomCmd() string{
	commands:=[]string{raft.Get,raft.Put,raft.Delete}
	rand.Seed(time.Now().UnixNano())
	idx:=rand.Intn(len(commands))
	cmd:=commands[idx]
	return cmd
}

func newRandomKey() string{
	rand.Seed(time.Now().UnixNano())
	idx:=rand.Intn(1000)
	key:="key"+strconv.Itoa(idx)
	return key
}

func newRandomValue() string{
	rand.Seed(time.Now().UnixNano())
	idx:=rand.Intn(1000)
	value:="value"+strconv.Itoa(idx)
	return value
}

func newRandomCommand() raft.Command{
	cmd:=newRandomCmd()
	key:=newRandomKey()
	value:=newRandomValue()
	return raft.Command{cmd,key,value}
}
