package raft

func (rf *Raft) leaderTicker() {
	for !rf.killed() {
		<-createTimeout(HEARTBEAT_TIMEOUT, HEARTBEAT_TIMEOUT)

		rf.mu.Lock()
		isLeader := rf.state == LEADER_STATE
		rf.mu.Unlock()

		if isLeader {
			rf.sendAppendEntries()
		}
	}
}

func (rf *Raft) sendAppendEntries() {
	var send = func(server int) {
		rf.mu.Lock()

		if rf.state != LEADER_STATE {
			rf.mu.Unlock()
			return
		}

		args := AppendEntriesArgs{}
		args.Term = rf.currentTerm
		args.LeaderId = rf.me
		args.PrevLogIndex = rf.nextIndex[server] - 1
		args.PrevLogTerm = rf.log[args.PrevLogIndex].Term
		if rf.nextIndex[server] <= rf.getLastLogIndex() {
			// If last log index ≥ nextIndex for a follower:
			// send AppendEntries RPC with log entries starting at nextIndex
			args.Entries = rf.log[rf.nextIndex[server]:]
		}
		args.LeaderCommit = rf.commitIndex

		reply := AppendEntriesReply{}
		rf.mu.Unlock()

		ok := rf.RPCAppendEntries(server, &args, &reply)
		if !ok {
			return
		}

		rf.mu.Lock()
		defer rf.mu.Unlock()

		if rf.state != LEADER_STATE {
			return
		}

		if reply.Term > rf.currentTerm {
			// If RPC request or response contains term T > currentTerm:
			// set currentTerm = T, convert to follower (§5.1)
			rf.currentTerm = reply.Term
			rf.changeState(FOLLOWER_STATE, true)
			return
		}

		if reply.Success {
			// If successful: update nextIndex and matchIndex for follower (§5.3)
			if len(args.Entries) > 0 {
				rf.nextIndex[server] = args.Entries[len(args.Entries)-1].Index + 1
				rf.matchIndex[server] = rf.nextIndex[server] - 1
			}
		} else {
			// If AppendEntries fails because of log inconsistency: decrement nextIndex and retry (§5.3)
			if rf.nextIndex[server] > 1 {
				rf.nextIndex[server] -= 1
			}
		}

	}

	for peer := range rf.peers {
		if peer != rf.me {
			go send(peer)
		}
	}

	rf.mu.Lock()
	defer rf.mu.Unlock()

	// If there exists an N such that
	// N > commitIndex, a majority of matchIndex[i] ≥ N, and log[N].term == currentTerm:
	// set commitIndex = N (§5.3, §5.4).
	for n := rf.getLastLogIndex(); n > rf.commitIndex && rf.log[n].Term == rf.currentTerm; n-- {
		count := 1
		for peer := range rf.peers {
			if peer != rf.me && rf.matchIndex[peer] >= n {
				count++
			}
		}
		if count > len(rf.peers)/2 {
			rf.commitIndex = n
			go rf.applyLog()
			break
		}
	}
}
