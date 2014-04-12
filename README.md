#Raft

This is a `go` implementation of Raft consensus algorithm. Raft is a consensus algorithm for managing a log replication on distributed system. The details of Raft can be found in paper [In Search of an Understandable Consensus Algorithm](https://ramcloud.stanford.edu/wiki/download/attachments/11370504/raft.pdf)

#Installation and Test

```
// get the sourcecode
go get github.com/sk4x0r/raft
//switch current working directory to source directory
cd go/src/github.com/sk4x0r/raft
//build the executable for code in folder `start_server`
cd start_server
go build start_server.go
//get back to source code dictionary
cd ..
//execute the test
go test github.com/sk4x0r/raft
```

#Dependency
Library depends upon ZeroMQ 4, which can be installed from [github.com/pebbe/zmq4](github.com/pebbe/zmq4). It also requires library `sortutil` installed which can be found at [github.com/cznic/sortutil](github.com/cznic/sortutil).


#Usage
New instance of server can be created by method `New()`.
```
s1:=raft.New(serverId, configFile)
```
`New()` method takes two parameter, and returns object of type `Server`.

| Parameter		| Type		| Description  
| -------------|:---------:| -----------
| serverId		| `int` 	| unuque id assigned to each server
| configFile	| `string`  | path of the file containing configuration of all the servers

For example of configuration file, see _config.json_ file in source.

For starting newly created server instance, `Start()` method is used.
For instance, instance `s1` from previous example can be started using following command
```
s1.Start()
```

Running instance of a server can be stopped using `Stop()` method.
```
s1.Stop()
```

Commands can be sent to server over `inbox` and processed output is received over `outbox` as shown below:
```
inbox:=s1.RaftInbox()
outbox:=s1.RaftOutbox()
//if 'cmd' is a command I want to send then
inbox <-c1
//after processing the command by server, result is sent on outbox
resp:= <-outbox
//now 'resp' contains the output after processing command 'cmd'
```

# License

The package is available under GNU General Public License. See the _LICENSE_ file.

# References and contribution
1. Raft paper [In Search of an Understandable Consensus Algorithm](https://ramcloud.stanford.edu/wiki/download/attachments/11370504/raft.pdf)
2. [Golang Tutorials](http://tour.golang.org/)
3. [The Go Blog](http://blog.golang.org/)
4. [A Go implementation of the Raft distributed consensus protocol](https://github.com/goraft/raft)
5. Help of classmates of CS733. The nonexhausting list includes Sagar Sontakke, Pushkar Khadilkar, Amol Bhangdiya, Kallol Dey
