package raft

//
// as each Raft peer becomes aware that successive log entries are
// committed, the peer should send an ApplyMsg to the service (or
// tester) on the same server, via the applyCh passed to Make(). set
// CommandValid to true to indicate that the ApplyMsg contains a newly
// committed log entry.
//
// in part 2D you'll want to send other kinds of messages (e.g.,
// snapshots) on the applyCh, but set CommandValid to false for these
// other uses.
//
type ApplyMsg struct {
	CommandValid bool
	Command      interface{}
	CommandIndex int

	// For 2D:
	SnapshotValid bool
	Snapshot      []byte
	SnapshotTerm  int
	SnapshotIndex int
}

func (rf *Raft) apply() {
	// If commitIndex > lastApplied:
	// increment lastApplied, apply log[lastApplied] to state machine (§5.3)

	for i := rf.lastAppiled + 1; i <= rf.commitIndex; i++ {
		msg := ApplyMsg{}
		msg.CommandIndex = i
		msg.CommandValid = true
		msg.Command = rf.log[i].Command
		rf.applyCh <- msg

		rf.uprint("apply %v", i)
	}

	rf.lastAppiled = rf.commitIndex
}
