package raft
import (
	"log"
	"encoding/json"
	"io/ioutil"
	"strconv"
)

//used to create json object of term for storing on disk
type TermJson struct {
	Term int64
}

func loadTermFromDisk(serverId int) int64 {
	var term int64
	fileName := strconv.Itoa(serverId) + ".term"
	fileBytes, err := ioutil.ReadFile(fileName)
	if err != nil {
		//log.Println("Unable to find term stored on disk")
	}
	var termJson TermJson
	err = json.Unmarshal(fileBytes, &termJson)
	if err != nil {
		//log.Print("Error while unmarshalling. Initializing the term to zero")
		term = 0
	} else {
		term = termJson.Term
	}
	return term
}

//struct used to marshal/unmarshal json objects of configuration file
type Config struct {
	Timeout int
	Peers   []Peer
}

func (c *Config) getPeers(myId int) []int {
	l := len(c.Peers)
	pids := make([]int, l-1)

	i := 0
	for _, peer := range c.Peers {
		if peer.Pid != myId {
			pids[i] = peer.Pid
			i = i + 1
		}
	}
	return pids
}

func (c *Config) getPort(myId int) int {
	for _, peer := range c.Peers {
		if peer.Pid == myId {
			return peer.Port
		}
	}
	panic("myId" + string(myId) + "doesn't exist")
}

func (c *Config) getPeerInfo(myId int) map[int]Peer {
	peerInfo := make(map[int]Peer)
	for _, peer := range c.Peers {
		if peer.Pid != myId {
			peer.prevLogIndex=-1 //WORKAROUND: consider proper initialization
			peerInfo[peer.Pid] = peer
		}
	}
	return peerInfo
}

func parseConfigFile(configFile string) Config {
	content, err := ioutil.ReadFile(configFile)
	if err != nil {
		log.Println("Error parsing the config file")
		panic(err)
	}
	var conf Config
	err = json.Unmarshal(content, &conf)
	if err != nil {
		log.Println("Error parsing the config file")
		panic(err)
	}
	return conf
}
