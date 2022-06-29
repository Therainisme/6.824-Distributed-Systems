package raft

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
		reply.Success = false
		reply.Term = rf.currentTerm

		return
	}

	if args.Term > rf.currentTerm {
		// If RPC request or response contains term T > currentTerm:
		// set currentTerm = T, convert to follower (§5.1)
		rf.currentTerm = args.Term

		go rf.gotoState(FOLLOWER_STATE)
	}

	reply.Success = true
	reply.Term = rf.currentTerm

	rf.receiveAppendEntries <- struct{}{}

	rf.uprint("receive AppendEntries from [Peer %d]", args.LeaderId)
}

func (rf *Raft) sendAppendEntries(server int, args *AppendEntriesArgs, reply *AppendEntriesReply) EntriesReply {
	ok := rf.peers[server].Call("Raft.AppendEntries", args, reply)

	return EntriesReply{
		AppendEntriesReply: *reply,
		ok:                 ok,
	}
}

type EntriesReply struct {
	AppendEntriesReply
	ok bool
}
