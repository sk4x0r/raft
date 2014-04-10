package raft

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"strconv"
	"sync"
)

type Log struct {
	entries     []LogItem
	commitIndex int64
	mutex       sync.RWMutex
	fileName    string
	path        string
	serverId    int
}

func newLog(serverId int) Log {
	l := Log{
		fileName:    strconv.Itoa(serverId) + "_log_entries.log",
		serverId: serverId,
	}
	l.loadEntriesFromDisk()
	l.loadCommitIndexFromDisk()
	return l
}

func (l *Log) CurrentIndex() int64 {
	l.mutex.RLock()
	defer l.mutex.RUnlock()
	if len(l.entries) == 0 {
		return -1
	}
	return l.entries[len(l.entries)-1].Index
}

func (l *Log) CurrentTerm() int64 {
	l.mutex.RLock()
	defer l.mutex.RUnlock()

	if len(l.entries) == 0 {
		return -1
	}
	return l.entries[len(l.entries)-1].Term
}

func (l *Log) CommitIndex() int64 {
	return l.commitIndex
}

//cite: https://github.com/goraft/raft
func (l *Log) lastInfo() (index int64, term int64) {
	l.mutex.RLock()
	defer l.mutex.RUnlock()

	if len(l.entries) == 0 {
		return -1, -1 //TODO:check whether the value should be 0 or -1
	}

	entry := l.entries[len(l.entries)-1]
	return entry.Index, entry.Term
}

func (l *Log) appendEntries(entries []LogItem) error {
	//TODO: error handling
	l.mutex.Lock()
	defer l.mutex.Unlock()
	l.entries = append(l.entries, entries...)
	//WORKAROUND: handle persistence
	l.saveEntriesToDisk()
	return nil
}

//WORKAROUND: func saveToDisk() is temporary workaround, consider using leveldb
func (l *Log) saveEntriesToDisk() {
	fileBytes, err := json.Marshal(l.entries)
	if err != nil {
		panic(err) //TODO: proper error handling
	}
	_ = ioutil.WriteFile(l.fileName, fileBytes, 0644)
}

//WORKAROUND: func readFromDisk() is temporary workaround, consider using leveldb
func (l *Log) loadEntriesFromDisk() {
	fileBytes, err := ioutil.ReadFile(l.fileName)
	if err != nil {
		l.entries = make([]LogItem, 0)
	}
	_ = json.Unmarshal(fileBytes, &(l.entries))
}



type CommitIndexJson struct {
	CommitIndex int64
}

func (l *Log) loadCommitIndexFromDisk(){
	var commitIndex int64
	fileName := strconv.Itoa(l.serverId) + ".ci"
	fileBytes, err := ioutil.ReadFile(fileName)
	if err != nil {
		//log.Println("Unable to find term stored on disk")
	}
	var ciJson CommitIndexJson
	err = json.Unmarshal(fileBytes, &ciJson)
	if err != nil {
		//log.Print("Error while unmarshalling. Initializing the term to zero")
		commitIndex=-1
	} else {
		commitIndex = ciJson.CommitIndex
	}
	l.mutex.Lock()
	l.commitIndex=commitIndex
	l.mutex.Unlock()
}


func (l *Log) saveCommitIndexToDisk(){
	fileName:=strconv.Itoa(l.serverId)+".ci"
	ciJson:=CommitIndexJson{l.CommitIndex()}
	
	fileBytes, err := json.Marshal(ciJson)
	if err!=nil{
		panic("Error while marshalling"+err.Error())
	}
	
	err=ioutil.WriteFile(fileName, fileBytes, 0644)
	if err!=nil{
		panic("Error writing to disk:"+ err.Error())
	}
}

func (l *Log) appendEntry(entry LogItem) error {
	//TODO: error handling
	//WORKAROUND: handle persistence
	l.mutex.Lock()
	defer l.mutex.Unlock()
	l.entries = append(l.entries, entry)
	l.saveEntriesToDisk()
	return nil
}

func (l *Log) newLogEntry(term int64, cmd Command) LogItem {
	return newLogItem(l.CurrentIndex()+1, term, cmd) //TODO: consider defining l.nextIndex() instead of l.currentIndex()+1
}

func (l *Log) setCommitIndex(index int64) error {
	//TODO: error handling
	l.mutex.Lock()
	defer l.mutex.Unlock()
	if index > int64(len(l.entries)) {
		//commitIndex is greater than length of log
		// going to commit all entries from log
		index = int64(len(l.entries) - 1)
	}

	if index < l.commitIndex {
		//already committed till commitindex, nothing needs to be changed
		return nil
	}

	l.commitIndex = index
	l.saveCommitIndexToDisk()
	return nil
}

//cite: https://github.com/goraft/raft
//This method truncates the log till previous matching entry
//Entries after matched previousLogIndex are discarded using this method
func (l *Log) truncate(index int64, term int64) error {
	l.mutex.Lock()
	defer l.mutex.Unlock()

	if index < l.commitIndex {
		//committed entries are being tried to truncate
		//TODO: Explore this
		return fmt.Errorf("Error:IndexAlreadyCommitted")
	}

	if index > int64(len(l.entries)) {
		//nonexisting index is being tried to truncate
		return fmt.Errorf("Error:IndexDoesntExist")
	}

	if index == -1 {
		//TODO: check whether this should be 0 or -1
		// Truncate everything
		l.entries = []LogItem{}
		//WORKAROUND: handle persistence
		l.saveEntriesToDisk()
	} else {
		entry := l.entries[index] //TODO: check whether this should be index or index-1
		if len(l.entries) > 0 && entry.Term != term {
			// Do not truncate if the entry at index do not have matching term
			return fmt.Errorf("Error:TermMismatch")
		}

		// Otherwise truncate up to the desired entry.
		if index < int64(len(l.entries)) {
			l.entries = l.entries[:index+1]
			l.saveEntriesToDisk()
		}
	}
	return nil
}

//TODO: remove panic, return error
func (l *Log) getEntriesAfter(index int64) []LogItem {
	l.mutex.RLock()
	defer l.mutex.RUnlock()
	if index < -1 {
		panic("Index underflow")
	}
	if index > int64(len(l.entries)) {
		panic("Index overflow")
	}
	return l.entries[index+1:]
}
