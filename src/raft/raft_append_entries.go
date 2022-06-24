package raft

import "time"

type LogEntry struct {
	Index   int
	Term    int
	Command interface{}
}

type AppendEntriesArgs struct {
	Term         int
	LeaderId     int
	PrevLogIndex int
	PrevLogTerm  int
	Entries      []LogEntry
	LeaderCommit int
}

type AppendEntriesReply struct {
	Term    int
	Success bool
}

func (rf *Raft) AppendEntries(args *AppendEntriesArgs, reply *AppendEntriesReply) {
	rf.mu.Lock()
	defer rf.mu.Unlock()

	if args.Term < rf.currentTerm {
		// Reply false if term < currentTerm (§5.1)
		reply.Term = rf.currentTerm
		reply.Success = false
		return
	}

	if args.Term >= rf.currentTerm {
		// If RPC request or response contains term T > currentTerm:
		// set currentTerm = T, convert to follower (§5.1)
		rf.currentTerm = args.Term
		rf.changeState(FOLLOWER_STATE, false)
	}

	rf.lastResetElectionTime = time.Now()

	reply.Term = rf.currentTerm
	reply.Success = true

	if args.PrevLogIndex > rf.getLastLogIndex() {
		// Reply false if log doesn’t contain an entry at prevLogIndex whose term matches prevLogTerm
		reply.Success = false
		return
	}

	if args.PrevLogTerm != rf.log[args.PrevLogIndex].Term {
		// Reply false if log doesn’t contain an entry at prevLogIndex whose term matches prevLogTerm
		reply.Success = false
		return
	}

	rf.log = rf.log[:args.PrevLogIndex+1]
	rf.log = append(rf.log, args.Entries...)

	if rf.commitIndex < args.LeaderCommit {
		// If leaderCommit > commitIndex, set commitIndex = min(leaderCommit, index of last new entry)
		rf.commitIndex = min(args.LeaderCommit, rf.getLastLogIndex())
	}
}

func (rf *Raft) RPCAppendEntries(server int, args *AppendEntriesArgs, reply *AppendEntriesReply) bool {
	ok := rf.peers[server].Call("Raft.AppendEntries", args, reply)
	return ok
}
