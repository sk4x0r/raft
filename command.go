package raft
const(
	Get="get"
	Put="put"
	Delete="delete"
)

type Command struct {
	Cmd string
	Key string
	Value string
}

func newCommand(cmd string, key string, value string) Command{
	return Command{
		Cmd:cmd,
		Key:key,
		Value:value,
		}
}

const(
	Ok="ok"
	Error="error"
	Redirect="redirect"
)

type Response struct{
	Status string
	LeaderId int
	Value string
}

func newResponse(status string, leaderId int, value string) Response{
	return Response{
		Status:status,
		LeaderId:leaderId,
		Value:value,
	}
}
