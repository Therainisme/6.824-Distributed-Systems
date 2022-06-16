package mr

import (
	"encoding/json"
	"fmt"
	"hash/fnv"
	"io/ioutil"
	"log"
	"net/rpc"
	"os"
)

//
// Map functions return a slice of KeyValue.
//
type KeyValue struct {
	Key   string
	Value string
}

//
// use ihash(key) % NReduce to choose the reduce
// task number for each KeyValue emitted by Map.
//
func ihash(key string) int {
	h := fnv.New32a()
	h.Write([]byte(key))
	return int(h.Sum32() & 0x7fffffff)
}

//
// main/mrworker.go calls this function.
//
func Worker(mapf func(string, string) []KeyValue, reducef func(string, []string) string) {
	// Your worker implementation here.
	w := worker{}
	w.mapFunc = mapf
	w.reduceFunc = reducef

	w.register()
	w.run()
}

//
// send an RPC request to the coordinator, wait for the response.
// usually returns true.
// returns false if something goes wrong.
//
func call(rpcname string, args interface{}, reply interface{}) bool {
	// c, err := rpc.DialHTTP("tcp", "127.0.0.1"+":1234")
	sockname := coordinatorSock()
	c, err := rpc.DialHTTP("unix", sockname)
	if err != nil {
		log.Fatal("dialing:", err)
	}
	defer c.Close()

	err = c.Call(rpcname, args, reply)
	if err == nil {
		return true
	}

	fmt.Println(err)
	return false
}

type worker struct {
	id      int
	nMap    int
	nReduce int

	mapFunc    func(string, string) []KeyValue
	reduceFunc func(string, []string) string
}

func (w *worker) register() {
	// Register
	args := WorkerRegisterArgs{}
	reply := WorkerRegisterReply{}
	if ok := call("Coordinator.WorkerRegister", &args, &reply); !ok {
		print("Call Coordinator.WorkerRegister err")
		os.Exit(1)
	}

	// store some information
	w.id = reply.WorkerId
	w.nMap = reply.NMap
	w.nReduce = reply.NReduce
}

func (w *worker) run() {
	var getTask = func() Task {
		args := WorkerGetTaskArgs{WorkerId: w.id}
		reply := WorkerGetTaskReply{}
		if ok := call("Coordinator.WorkerGetTask", &args, &reply); !ok {
			print("Call Coordinator.WorkerGetTask err")
			os.Exit(1)
		}

		return Task{
			Id:       reply.TaskId,
			Type:     reply.TaskType,
			Filename: reply.Filename,
		}
	}

	var reportTask = func(t Task, AC bool) {
		args := WorkerReportTaskArgs{WorkerId: w.id, Taskid: t.Id, AC: AC}
		reply := WorkerReportTaskReply{}
		if ok := call("Coordinator.WorkerReportTask", &args, &reply); !ok {
			print("Call Coordinator.WorkerReportTask err")
			os.Exit(1)
		}
	}

	var handleMap = func(t Task) {
		content, err := ioutil.ReadFile(t.Filename)
		if err != nil {
			print("cannot read %v", t.Filename)
			reportTask(t, false)
			return
		}

		kva := w.mapFunc(t.Filename, string(content))
		intermediateData := make([][]KeyValue, w.nReduce)

		for _, kv := range kva {
			idx := ihash(kv.Key) % w.nReduce
			intermediateData[idx] = append(intermediateData[idx], kv)
		}

		for idx, kvs := range intermediateData {
			file, err := ioutil.TempFile("", "")
			if err != nil {
				print("cannot create temp file")
				reportTask(t, false)
				return
			}

			enc := json.NewEncoder(file)
			for _, kv := range kvs {
				if err := enc.Encode(&kv); err != nil {
					print("write intermediate file error")
					reportTask(t, false)
					return
				}
			}

			// mr-<task's id>-<idx>
			if err := os.Rename(file.Name(), fmt.Sprintf("mr-%d-%d", t.Id, idx)); err != nil {
				print("cannot rename intermediate file")
				reportTask(t, false)
				return
			}
		}

		reportTask(t, true)
	}

	var handleReduce = func(t Task) {
		// read from mr-<0~nMap>-<task's id>
		outputFile, err := ioutil.TempFile("", "")
		if err != nil {
			print("cannot create temp file")
			reportTask(t, false)
			return
		}

		mapOutput := make(map[string][]string, 0)

		// read intermediate file
		for i := 0; i < w.nMap; i++ {
			// read mr-<i>-<task's id>
			filename := fmt.Sprintf("mr-%d-%d", i, t.Id)
			intermediateFile, err := os.Open(filename)
			if err != nil {
				print("cannot open intermediate file")
				reportTask(t, false)
				return
			}

			dec := json.NewDecoder(intermediateFile)
			for {
				var kv KeyValue
				if err := dec.Decode(&kv); err != nil {
					break
				}
				if _, ok := mapOutput[kv.Key]; !ok {
					mapOutput[kv.Key] = make([]string, 0)
				}
				mapOutput[kv.Key] = append(mapOutput[kv.Key], kv.Value)
			}
		}

		// call reduceFunc
		for key, valueList := range mapOutput {
			_, err := fmt.Fprintf(outputFile, "%v %v\n", key, w.reduceFunc(key, valueList))
			if err != nil {
				print("write output file error")
				reportTask(t, false)
				return
			}
		}

		// mr-out-<task's id>
		if err := os.Rename(outputFile.Name(), fmt.Sprintf("mr-out-%d", t.Id)); err != nil {
			print("write output file error")
			reportTask(t, false)
			return
		}

		reportTask(t, true)
	}

	for {
		t := getTask()
		if t.Id == -1 {
			print("worker receive task which id is -1, exit")
			return
		}

		print("worker [%d] is running task: [%+v]", w.id, t)

		switch t.Type {
		case MapPeriod:
			handleMap(t)

		case ReducePeriod:
			handleReduce(t)

		default:
			print("Receive impossible task type")
			os.Exit(1)
		}
	}
}
