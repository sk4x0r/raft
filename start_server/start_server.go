package main
import (
	raft "github.com/sk4x0r/raft"
	"os"
	"strconv"
	"fmt"
)
const 	PATH_TO_CONFIG     = "config.json"
func main(){
	
	
	id, _ := strconv.Atoi((os.Args[1]))
	fmt.Println("Server id at start_server:",id)
	s:=raft.New(id,PATH_TO_CONFIG)
	s.Start()
}
