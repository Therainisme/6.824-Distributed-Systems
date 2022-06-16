package mr

//
// RPC definitions.
//
// remember to capitalize all names.
//

import (
	"os"
	"strconv"
)

// Add your RPC definitions here.

// WorkerRegister
type WorkerRegisterArgs struct {
}
type WorkerRegisterReply struct {
	WorkerId int
	NMap     int
	NReduce  int
}

// WorkerGetTask
type WorkerGetTaskArgs struct {
	WorkerId int
}
type WorkerGetTaskReply struct {
	TaskId   int
	TaskType string // Map or Reduce
	Filename string // Map only
}

// WorkerReportTask
type WorkerReportTaskArgs struct {
	WorkerId int
	Taskid   int
	AC       bool
}
type WorkerReportTaskReply struct {
}

// Cook up a unique-ish UNIX-domain socket name
// in /var/tmp, for the coordinator.
// Can't use the current directory since
// Athena AFS doesn't support UNIX-domain sockets.
func coordinatorSock() string {
	s := "/var/tmp/824-mr-"
	s += strconv.Itoa(os.Getuid())
	return s
}
