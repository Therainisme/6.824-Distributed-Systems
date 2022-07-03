package raft

//
// A service wants to switch to snapshot.  Only do so if Raft hasn't
// have more recent info since it communicate the snapshot on applyCh.
//
func (rf *Raft) CondInstallSnapshot(lastIncludedTerm int, lastIncludedIndex int, snapshot []byte) bool {
	// Your code here (2D).

	return true
}

// the service says it has created a snapshot that has
// all info up to and including index. this means the
// service no longer needs the log through (and including)
// that index. Raft should now trim its log as much as possible.
func (rf *Raft) Snapshot(index int, snapshot []byte) {
	// Your code here (2D).
	rf.uprint("create snapshot to index: %d", index)

	// ! why can not lock here?
	// rf.mu.Lock()
	// defer rf.mu.Unlock()

	// Each server takes snapshots independently, covering just the committed entries in its log.

	if index <= rf.getLastSnapshotIndex() || index > rf.getLastLogIndex() {
		return
	}

	rf.trimLog(index, rf.log[rf.convertIndex(index)].Term)

	rf.uprint("remained logs: %v", rf.log)

	// Most of the work consists of the state machine writing its current state to the snapshot.
	rf.persister.SaveStateAndSnapshot(rf.getPersistData(), snapshot)
}

type InstallSnapshotArgs struct {
	Term              int
	LeaderId          int
	LastIncludedIndex int
	LastIncludedTerm  int
	Data              []byte
}

type InstallSnapshotReply struct {
	Term int
}

func (rf *Raft) InstallSnapshot(args *InstallSnapshotArgs, reply *InstallSnapshotReply) {
	// Usually the snapshot will contain new information not already in the recipient’s log.

	rf.mu.Lock()
	defer rf.mu.Unlock()
	defer func() {
		go rf.persist()
	}()

	rf.uprint("receive snapshot to %d from server %d", args.LastIncludedIndex, args.LeaderId)

	if args.Term < rf.currentTerm {
		// Reply immediately if term < currentTerm
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

	reply.Term = rf.currentTerm

	// If existing log entry has same index and term as snapshot’s last included entry,
	// retain log entries following it and reply

	// the follower is too far behind
	//
	// but if args.LastIncludedIndex < rf.commitIndex,
	// means that the follower contains rf.commitIndex, reject snapshot from leader
	if args.LastIncludedIndex > rf.commitIndex {
		rf.trimLog(args.LastIncludedIndex, args.LastIncludedTerm)
		rf.lastAppiled = args.LastIncludedIndex
		rf.commitIndex = args.LastIncludedIndex
		rf.persister.SaveStateAndSnapshot(rf.getPersistData(), args.Data)

		rf.applyCh <- ApplyMsg{
			SnapshotValid: true,
			Snapshot:      args.Data,
			SnapshotIndex: rf.getLastSnapshotIndex(),
			SnapshotTerm:  rf.getLastSnapshotTerm(),
		}
	}
}

func (rf *Raft) trimLog(lastIncludedIndex, lastIncludedTerm int) {
	remainedLog := make([]LogEntry, 0)
	remainedLog = append(remainedLog, LogEntry{Index: lastIncludedIndex, Term: lastIncludedTerm})

	for i := rf.convertIndex(lastIncludedIndex + 1); i < len(rf.log); i++ {
		remainedLog = append(remainedLog, rf.log[i])
	}

	rf.log = remainedLog
}

func (rf *Raft) sendInstallSnapshot(server int, args *InstallSnapshotArgs, reply *InstallSnapshotReply) {

	ok := rf.peers[server].Call("Raft.InstallSnapshot", args, reply)

	rf.mu.Lock()
	defer rf.mu.Unlock()
	defer func() {
		go rf.persist()
	}()

	if !ok || rf.state != LEADER_STATE || args.Term != rf.currentTerm {
		return
	}

	if reply.Term > rf.currentTerm {
		rf.currentTerm = reply.Term
		rf.gotoState(FOLLOWER_STATE, false)
		return
	}

	rf.nextIndex[server] = args.LastIncludedIndex + 1
	rf.matchIndex[server] = args.LastIncludedIndex
}

func (rf *Raft) getLastSnapshotIndex() int {
	return rf.log[0].Index
}

func (rf *Raft) getLastSnapshotTerm() int {
	return rf.log[0].Term
}

// snaphost last index: 3
// raft log: [3, 4, 5]
// raft idx: [0, 1, 2]
// input idx = 5
// return = 2
func (rf *Raft) convertIndex(idx int) int {
	return idx - rf.getLastSnapshotIndex()
}
