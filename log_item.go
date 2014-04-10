package raft
//struct to store log entry
type LogItem struct {
	Index int64
	Term  int64
	Data  Command
}

func newLogItem(index int64, term int64, data Command) LogItem {
	return LogItem{
		Index: index,
		Term:  term,
		Data:  data,
	}
}
