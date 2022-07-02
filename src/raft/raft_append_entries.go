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
	defer func() {
		go rf.persist()
	}()

	// rf.uprint("Logs %+v", rf.log)
	rf.uprint("receive from server %d", args.LeaderId)

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

		rf.gotoState(FOLLOWER_STATE, false)
	}

	rf.receiveAppendEntries <- struct{}{}

	// rf.uprint("receive AppendEntries from [Peer %d]", args.LeaderId)

	if args.PrevLogIndex > rf.getLastLogIndex() {
		reply.Success = false
		reply.Term = rf.currentTerm
		return
	}

	// If an existing entry conflicts with a new one (same index but different terms),
	// delete the existing entry and all that follow it (§5.3)
	if args.PrevLogTerm != rf.log[rf.convertIndex(args.PrevLogIndex)].Term {
		reply.Success = false
		reply.Term = rf.currentTerm
		rf.log = rf.log[:rf.convertIndex(args.PrevLogIndex+1)]

		return
	}

	if len(args.Entries) > 0 && rf.getLastLogIndex() >= args.Entries[len(args.Entries)-1].Index {
		for _, entries := range args.Entries {
			if rf.log[rf.convertIndex(entries.Index)].Term != entries.Term {
				rf.log = rf.log[:rf.convertIndex(entries.Index)]
				reply.Success = false
				reply.Term = rf.currentTerm
				return
			}
		}

		reply.Success = true
		reply.Term = rf.currentTerm
	} else {
		// Success
		reply.Success = true
		reply.Term = rf.currentTerm

		rf.log = rf.log[:rf.convertIndex(args.PrevLogIndex+1)]
		rf.log = append(rf.log, args.Entries...)
	}

	// If leaderCommit > commitIndex, set commitIndex = min(leaderCommit, index of last new entry)
	if args.LeaderCommit > rf.commitIndex {
		rf.commitIndex = min(args.LeaderCommit, rf.getLastLogIndex())
		rf.apply()
	}
}

func (rf *Raft) sendAppendEntries(server int, args AppendEntriesArgs, reply *AppendEntriesReply) {
	ok := rf.peers[server].Call("Raft.AppendEntries", &args, reply)

	rf.mu.Lock()
	defer rf.mu.Unlock()
	defer func() {
		go rf.persist()
	}()

	if !ok || args.Term != rf.currentTerm {
		rf.uprint("send error to server %d", server)
		return
	}

	if reply.Term > rf.currentTerm {
		// If RPC request or response contains term T > currentTerm:
		// set currentTerm = T, convert to follower (§5.1)
		rf.currentTerm = reply.Term
		rf.gotoState(FOLLOWER_STATE, false)
		return
	}

	if reply.Success {
		if len(args.Entries) > 0 {
			// If successful: update nextIndex and matchIndex for follower (§5.3)
			rf.nextIndex[server] = args.Entries[len(args.Entries)-1].Index + 1
			rf.matchIndex[server] = args.PrevLogIndex + len(args.Entries)
		}
	} else {
		// If AppendEntries fails because of log inconsistency: decrement nextIndex and retry (§5.3)
		if rf.nextIndex[server] > 1 {
			rf.nextIndex[server] = rf.getPrevTermLogIndex(rf.nextIndex[server]) + 1
			rf.uprint("log len %d, nextIndex %d\n", len(rf.log), rf.nextIndex[server])
		} else {
			rf.nextIndex[server] = 1
		}
	}

	// If there exists an N such that N > commitIndex, a majority of matchIndex[i] ≥ N, and log[N].term == currentTerm:
	// set commitIndex = N (§5.3, §5.4).
	for n := rf.getLastLogIndex(); n > rf.commitIndex && rf.log[rf.convertIndex(n)].Term == rf.currentTerm; n-- {
		count := 1
		for peer := range rf.peers {
			if peer != rf.me && rf.matchIndex[peer] >= n {
				count++
			}
		}
		if count > len(rf.peers)/2 {
			rf.commitIndex = n
			rf.apply()
			break
		}
	}
}
