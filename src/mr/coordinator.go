package mr

import (
	"log"
	"net"
	"net/http"
	"net/rpc"
	"os"
	"sync"
	"time"
)

const (
	tickInterval       = time.Millisecond * 500
	maxTaskRunningTime = time.Second * 10
)

const (
	InitPeriod   = "Init"
	MapPeriod    = "Map"
	ReducePeriod = "Reduce"
	DonePeriod   = "Done"
)

// Task Status
const (
	ready   = 0
	queue   = 1
	running = 2
	err     = 3
	finish  = 4
)

type Coordinator struct {
	files       []string   // filenames of input
	nReduce     int        // Reduce task count
	nMap        int        // Map task count
	taskQueue   chan Task  // TaskQueue，where store Ready Task
	taskInfos   []TaskInfo // Task assigned information
	period      string     // Init or Map or Reduce or Done
	workerCount int        // Worker's id
	sm          sync.Mutex
}

type Task struct {
	Id       int    // TaskInfos index
	Filename string // Map only
	Type     string // Map or Reduce
}

type TaskInfo struct {
	WorkerId  int       // the assigned worker
	StartTime time.Time // the worker start running time
	Status    int       // ready queue running err finish
}

//
// create a Coordinator.
// main/mrcoordinator.go calls this function.
// nReduce is the number of reduce tasks to use.
//
func MakeCoordinator(files []string, nReduce int) *Coordinator {
	// Your code here.
	c := Coordinator{
		files:   files,
		nReduce: nReduce,
		nMap:    len(files),
		period:  InitPeriod,
	}

	if len(files) > nReduce {
		c.taskQueue = make(chan Task, len(files))
		c.taskInfos = make([]TaskInfo, len(files))
	} else {
		c.taskQueue = make(chan Task, nReduce)
		c.taskInfos = make([]TaskInfo, nReduce)
	}

	c.moveToNextPeriod()
	go c.schedule()
	c.server()
	return &c
}

//
// main/mrcoordinator.go calls Done() periodically to find out
// if the entire job has finished.
//
func (c *Coordinator) Done() bool {
	c.sm.Lock()
	var result = c.period == DonePeriod
	c.sm.Unlock()

	if result {
		for {
			select {
			case c.taskQueue <- Task{Id: -1}:
			default:
				return true
			}
		}
	}

	return false
}

// Your code here -- RPC handlers for the worker to call.

func (c *Coordinator) WorkerRegister(args *WorkerRegisterArgs, reply *WorkerRegisterReply) error {
	c.sm.Lock()
	defer c.sm.Unlock()

	reply.WorkerId = c.workerCount
	reply.NMap = c.nMap
	reply.NReduce = c.nReduce

	c.workerCount++
	return nil
}

func (c *Coordinator) WorkerGetTask(args *WorkerGetTaskArgs, reply *WorkerGetTaskReply) error {
	task := <-c.taskQueue

	c.sm.Lock()
	defer c.sm.Unlock()

	if task.Id != -1 {
		c.taskInfos[task.Id].WorkerId = args.WorkerId
		c.taskInfos[task.Id].StartTime = time.Now()
		c.taskInfos[task.Id].Status = running
	}

	reply.TaskId = task.Id
	reply.TaskType = task.Type
	reply.Filename = task.Filename
	return nil
}

func (c *Coordinator) WorkerReportTask(args *WorkerReportTaskArgs, reply *WorkerReportTaskReply) error {
	c.sm.Lock()
	defer c.sm.Unlock()

	if args.AC && c.taskInfos[args.Taskid].WorkerId == args.WorkerId {
		c.taskInfos[args.Taskid].Status = finish
	} else if c.taskInfos[args.Taskid].WorkerId != args.WorkerId {
		// nothing happen task is assgined to other worker
	} else if args.AC == false {
		c.taskInfos[args.Taskid].Status = err
	}

	return nil
}

//
// start a thread that listens for RPCs from worker.go
//
func (c *Coordinator) server() {
	rpc.Register(c)
	rpc.HandleHTTP()
	//l, e := net.Listen("tcp", ":1234")
	sockname := coordinatorSock()
	os.Remove(sockname)
	l, e := net.Listen("unix", sockname)
	if e != nil {
		log.Fatal("listen error:", e)
	}
	go http.Serve(l, nil)
}

func (c *Coordinator) moveToNextPeriod() {
	c.sm.Lock()
	defer c.sm.Unlock()

	switch c.period {
	case InitPeriod:
		// Init -> Map
		c.period = MapPeriod
		// clear task info
		c.taskInfos = make([]TaskInfo, c.nMap)

	case MapPeriod:
		// Map -> Reduce
		c.period = ReducePeriod
		// clear task info
		c.taskInfos = make([]TaskInfo, c.nReduce)

	case ReducePeriod:
		// Reduce -> Done
		c.period = DonePeriod

	default:
		// Done period
		// nothing happened here
	}
}

func (c *Coordinator) schedule() {
	var createReadyTask = func(id int) Task {
		t := Task{
			Id:   id,
			Type: c.period,
		}
		if c.period == MapPeriod {
			t.Filename = c.files[id]
		}
		return t
	}

	var tick = func() {
		var completed = true

		c.sm.Lock()
		for id, taskInfo := range c.taskInfos {
			switch taskInfo.Status {
			case ready:
				c.taskQueue <- createReadyTask(id)
				c.taskInfos[id].Status = queue
				completed = false

			case queue:
				completed = false

			case running:
				if time.Since(c.taskInfos[id].StartTime) > maxTaskRunningTime {
					c.taskQueue <- createReadyTask(id)
					c.taskInfos[id].Status = queue
				}

				completed = false

			case err:
				c.taskQueue <- createReadyTask(id)
				c.taskInfos[id].Status = queue
				completed = false

			case finish:
			default:
				panic("impossible task status")
			}
		}
		c.sm.Unlock()

		if completed {
			c.moveToNextPeriod()
		}
	}

	for {
		tick()
		time.Sleep(tickInterval)
	}
}
