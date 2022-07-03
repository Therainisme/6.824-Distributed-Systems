package raft

//
// this is an outline of the API that raft must expose to
// the service (or tester). see comments below for
// each of these functions for more details.
//
// rf = Make(...)
//   create a new Raft server.
// rf.Start(command interface{}) (index, term, isleader)
//   start agreement on a new log entry
// rf.GetState() (term, isLeader)
//   ask a Raft for its current term, and whether it thinks it is leader
// ApplyMsg
//   each time a new entry is committed to the log, each Raft peer
//   should send an ApplyMsg to the service (or tester)
//   in the same server.
//

import (
	//	"bytes"
	"context"
	"sync"
	"sync/atomic"

	//	"6.824/labgob"
	"6.824/labrpc"
)

//
// A Go object implementing a single Raft peer.
//
type Raft struct {
	mu        sync.Mutex          // Lock to protect shared access to this peer's state
	peers     []*labrpc.ClientEnd // RPC end points of all peers
	persister *Persister          // Object to hold this peer's persisted state
	me        int                 // this peer's index into peers[]
	dead      int32               // set by Kill()

	// Your data here (2A, 2B, 2C).
	// Look at the paper's Figure 2 for a description of what
	// state a Raft server must maintain.

	// Figure2: Persistent state on all servers:
	// (Updated on stable storage before responding to RPCs)
	currentTerm int        // latest term server has seen
	votedFor    int        // candidateId that received vote in current term
	log         []LogEntry // each entry contains command for state machine, and term

	// Figure2: Volatile state on all servers:
	commitIndex int // index of highest log entry known to be committed
	lastAppiled int // index of highest log entry applied to state machine

	// Figure2: Volatile state on leaders:
	// (Reinitialized after election)
	nextIndex  []int // for each server, index of the next log entry to send to that serve
	matchIndex []int // for each server, index of highest log entry known to be replicated on server

	applyCh              chan ApplyMsg
	cancelPrevStateCtx   context.CancelFunc
	state                int
	receiveAppendEntries chan struct{}
	receiveRequestVote   chan struct{}
}

// return currentTerm and whether this server
// believes it is the leader.
func (rf *Raft) GetState() (int, bool) {
	// Your code here (2A).
	rf.mu.Lock()
	defer rf.mu.Unlock()

	return rf.currentTerm, rf.state == LEADER_STATE
}

//
// the service using Raft (e.g. a k/v server) wants to start
// agreement on the next command to be appended to Raft's log. if this
// server isn't the leader, returns false. otherwise start the
// agreement and return immediately. there is no guarantee that this
// command will ever be committed to the Raft log, since the leader
// may fail or lose an election. even if the Raft instance has been killed,
// this function should return gracefully.
//
// the first return value is the index that the command will appear at
// if it's ever committed. the second return value is the current
// term. the third return value is true if this server believes it is
// the leader.
//
func (rf *Raft) Start(command interface{}) (int, int, bool) {
	// Your code here (2B).

	rf.mu.Lock()
	defer rf.mu.Unlock()
	defer func() {
		go rf.persist()
	}()

	if rf.state != LEADER_STATE {
		return -1, -1, false
	}

	index := rf.getLastLogIndex() + 1
	term := rf.currentTerm
	isLeader := true

	rf.log = append(rf.log, LogEntry{
		Index:   index,
		Term:    term,
		Command: command,
	})

	// rf.uprint("L receive log index: %d term: %d command: %v", index, term, command)
	// rf.uprint("Now L Logs: %v", rf.log)

	return index, term, isLeader
}

//
// the tester doesn't halt goroutines created by Raft after each test,
// but it does call the Kill() method. your code can use killed() to
// check whether Kill() has been called. the use of atomic avoids the
// need for a lock.
//
// the issue is that long-running goroutines use memory and may chew
// up CPU time, perhaps causing later tests to fail and generating
// confusing debug output. any goroutine with a long-running loop
// should call killed() to check whether it should stop.
//
func (rf *Raft) Kill() {
	atomic.StoreInt32(&rf.dead, 1)
	// Your code here, if desired.
}

func (rf *Raft) killed() bool {
	z := atomic.LoadInt32(&rf.dead)
	return z == 1
}

// The ticker go routine starts a new election if this peer hasn't received
// heartsbeats recently.
func (rf *Raft) ticker() {
	for rf.killed() == false {

		// Your code here to check if a leader election should
		// be started and to randomize sleeping time using
		// time.Sleep().

	}
}

//
// the service or tester wants to create a Raft server. the ports
// of all the Raft servers (including this one) are in peers[]. this
// server's port is peers[me]. all the servers' peers[] arrays
// have the same order. persister is a place for this server to
// save its persistent state, and also initially holds the most
// recent saved state, if any. applyCh is a channel on which the
// tester or service expects Raft to send ApplyMsg messages.
// Make() must return quickly, so it should start goroutines
// for any long-running work.
//
func Make(peers []*labrpc.ClientEnd, me int, persister *Persister, applyCh chan ApplyMsg) *Raft {
	rf := &Raft{}
	rf.peers = peers
	rf.persister = persister
	rf.me = me

	// Your initialization code here (2A, 2B, 2C).
	rf.currentTerm = 0
	rf.votedFor = 0
	rf.log = make([]LogEntry, 0)
	rf.log = append(rf.log, LogEntry{Term: 0, Index: 0})

	rf.commitIndex = 0
	rf.lastAppiled = 0

	rf.nextIndex = make([]int, len(rf.peers))
	rf.matchIndex = make([]int, len(rf.peers))

	// some mechanism
	rf.receiveAppendEntries = make(chan struct{}, 100)
	rf.receiveRequestVote = make(chan struct{}, 100)
	rf.applyCh = applyCh

	_, cancel := context.WithCancel(context.Background())
	rf.cancelPrevStateCtx = cancel

	// initialize from state persisted before a crash
	rf.readPersist(persister.ReadRaftState())

	if rf.getLastSnapshotIndex() > 0 {
		rf.lastAppiled = rf.getLastSnapshotIndex()
	}

	// start ticker goroutine to start elections
	rf.gotoState(FOLLOWER_STATE, true)

	return rf
}
