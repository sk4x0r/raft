package raft

import (
	"testing"
	"time"
	"strconv"
	"os"
	"log"
	"math/rand"
	"os/exec"
	"fmt"
	"github.com/syndtr/goleveldb/leveldb"
	"bytes"
	"strings"
)

//This function, takes input a map of servers, and return the leader among them
func findLeader(servers map[int]Server) Server{
	for true{
		for _,server:=range servers{
			//log.Println(server.Id(),server.State())
			if server.Leader(){
				//log.Println("leader found")
				return server
			}
			time.Sleep(100*time.Millisecond)
		}
	}
	//this should never execute
	var s Server
	return s
}

func cleanFiles(){
    os.RemoveAll("1001db")
    os.RemoveAll("1002db")
    os.RemoveAll("1003db")
    os.RemoveAll("1004db")
    os.RemoveAll("1005db")
    os.RemoveAll("1006db")
    os.RemoveAll("1001db.txt")
    os.RemoveAll("1002db.txt")
    os.RemoveAll("1003db.txt")
    os.RemoveAll("1004db.txt")
    os.RemoveAll("1005db.txt")
    os.RemoveAll("1006db.txt")
    os.Remove("1001.term")
    os.Remove("1002.term")
    os.Remove("1003.term")
    os.Remove("1004.term")
    os.Remove("1005.term")
    os.Remove("1006.term")
    os.Remove("1001.ci")
    os.Remove("1002.ci")
    os.Remove("1003.ci")
    os.Remove("1004.ci")
    os.Remove("1005.ci")
    os.Remove("1006.ci")
    os.Remove("1001_log_entries.log")
    os.Remove("1002_log_entries.log")
    os.Remove("1003_log_entries.log")
    os.Remove("1004_log_entries.log")
    os.Remove("1005_log_entries.log")
    os.Remove("1006_log_entries.log")
    
    /*
    cmd:= exec.Command("rm","-r","100*")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stdout
	cmd.Start()
	*/
}

//This function, takes input a map of servers and, returns a randomly selected server among them
func selectRandomServer(servers map[int]Server) Server{
	//log.Println("selecting random server")
	slice:=make([]int, 0)
	for key, _:=range servers{
		slice=append(slice,key)
	}
	rand.Seed(time.Now().UnixNano())
	randIdx:=rand.Intn(len(slice))
	idx:=slice[randIdx]
	s:=servers[idx]
	for s.State()==Stopped{
		rand.Seed(time.Now().UnixNano())
		randIdx:=rand.Intn(len(slice))
		idx:=slice[randIdx]
		s=servers[idx]
	}
	//log.Println("random server=",s.Id())
	return s
}

//This method takes a command and list of servers as arguemnt,
//and sends the command to leader.
func send(cmd Command, servers map[int]Server) (Response,Server){
	s:=selectRandomServer(servers)
	for true{
		outbox:=s.RaftOutbox()
		inbox:=s.RaftInbox()
		inbox <-cmd
		r:=Response{}
		select{
			case r= <-outbox:
			case <-time.After(1*time.Second):
				r=newResponse(Error,0,"")
		}
		//log.Println(s.Id(),r)
		if r.Status==Ok{
			return r,s
		}else if r.Status==Redirect{
			if r.LeaderId!=0{
				s=servers[r.LeaderId]
			}else{
				s=selectRandomServer(servers)
				time.Sleep(electionTimeout())
			}
		}else{
			s=selectRandomServer(servers)
			time.Sleep(electionTimeout())
		}
	}
	return Response{},s
}

func startServers(n int) map[int]Server{
	servers:=make(map[int]Server,n)
	for i:=1001;i<=1000+n;i++{
		s:=New(i,PATH_TO_CONFIG)
		s.Start()
		servers[i]=s
	}
	return servers
}

func killServers(servers map[int]Server){
	//log.Println("stopping")
	for _,s:=range servers{
		//log.Println("killing",s.Id())
		s.Stop()
	}
	//log.Println("stopped")
	time.Sleep(2*time.Second)
}

//cite: Sagar sontakke
func readLevelDb(n int){
	for i:=0;i<n;i++{
		dbname:=strconv.Itoa(i+1001)+"db"
		db,err:=leveldb.OpenFile(dbname, nil)
		if err!=nil{
			fmt.Println("Could not open db:",err)
		}
		iter := db.NewIterator(nil, nil)
		outfile:=dbname+".txt"
		_, err = os.Create(outfile)
		if err != nil {
			fmt.Println("File", outfile, "not created")
		}

		fobj, err := os.OpenFile(outfile, os.O_RDWR, 0600)
		if err != nil {
			fmt.Println("File", outfile, "not opened")
		}
		for iter.Next() {
			key := iter.Key()
			value := iter.Value()
			fobj.Write([]byte(string(key)+string(value)+"\n"))
		}
		iter.Release()
		err = iter.Error()
		fobj.Close()
		db.Close()
	}
}

//cite: Sagar Sontakke
func checkConsistency(n int) bool{
	consistent:=false
	readLevelDb(n)
	
	consistentCount:=0
	majority:=n/2+1
	var output bytes.Buffer
	for i:=0; i<n;i++ {
		consistentCount=1
		for j:=i+1;j<n;j++ {
			file1 := strconv.Itoa(i+1001) + "db.txt"
			file2 := strconv.Itoa(j+1001) + "db.txt"
			//fmt.Println("comparing",file1,file2)
			diff := exec.Command("diff", file1, file2)
			diff.Stdout = &output

			diff.Start()
			diff.Wait()

			out := string(output.Bytes())
			match := strings.TrimSpace(string(out))
			if match == "" {
				consistentCount++
				if consistentCount>=majority{
					consistent:=true
					return consistent
				}
			}
			output.Reset()
		}
		
	}
	fmt.Println("Consistency count;",consistentCount)
	return consistent
}


func _TestOne(t *testing.T){
		//log.Println("testone")
		cleanFiles()
		//log.Println("Starting servers")
		servers:=startServers(5)
		//log.Println("Servers started")
		time.Sleep(30*time.Second)
		
		
		for i:=1;i<=5;i++{
		//log.Println("i=",i)
		cmd:=newCommand(Put, "key"+strconv.Itoa(i),"val"+strconv.Itoa(i))
		//log.Println("command:",cmd)
		_,_=send(cmd, servers)
		//log.Println("reply:",rply)
		//if i%2==0{
		//leader.Stop()
		//time.Sleep(2*time.Second)
		//leader.Start()
		//}
		time.Sleep(500*time.Millisecond)
		}
		time.Sleep(2*time.Second)
		killServers(servers)
		success:=checkConsistency(len(servers))
		if !success{
		t.Fatalf("Test Multiple Restarts failed")
	}
}


func TestNormal(t *testing.T) {
	cleanFiles()	
	var cmd []*exec.Cmd
	cmd = make([]*exec.Cmd, 5)
	
	for i:=0; i<5; i++ {
		command := strconv.Itoa(1001 + i)
		cmd[i] = exec.Command("./start_server/start_server", command)
		cmd[i].Stdout = os.Stdout
		cmd[i].Stderr = os.Stdout
		cmd[i].Start()
	}																																																																																									
	
	time.Sleep(20*time.Second)
	
	for i:=0; i<5;i++ {
		cmd[i].Process.Kill()
		//log.Println("Killed")
	}
	time.Sleep(2*time.Second)
	success:=checkConsistency(5)
	if !success{
		t.Fatalf("Test Normal Operations - Failed")
	}else{
		log.Println("Test Normal Operations - Success")
	}
}



func TestMultipleRestarts(t *testing.T) {
	cleanFiles()	
	var cmd []*exec.Cmd
	cmd = make([]*exec.Cmd, 5)
	
	for i:=0; i<5; i++ {
		arg := strconv.Itoa(1001 + i)
		cmd[i] = exec.Command("./start_server/start_server", arg)
		cmd[i].Stdout = os.Stdout
		cmd[i].Stderr = os.Stdout
		cmd[i].Start()
	}
		
	time.Sleep(5*time.Second)
	
	for i:=0; i<10;i++ {
		cmd[i%5].Process.Kill()
		//log.Println("Killed",i%5)
		time.Sleep(1*time.Second)
		arg := strconv.Itoa(1001 + (i%5))
		cmd[i%5] = exec.Command("./start_server/start_server", arg)
		cmd[i%5].Stdout = os.Stdout
		cmd[i%5].Stderr = os.Stdout
		cmd[i%5].Start()
	}
	
	for i:=0; i<5;i++ {
		cmd[i].Process.Kill()
		//log.Println("Killed")
	}
	
	time.Sleep(2*time.Second)
	success:=checkConsistency(5)
	if !success{
		t.Fatalf("Test Multiple Restarts - Failed")
	}else{
		log.Println("Test Multiple Restarts - Success")
	}
}
