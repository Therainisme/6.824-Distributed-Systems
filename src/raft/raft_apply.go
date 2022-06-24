package raft

func (rf *Raft) applyLog() {
	applyMessages := make([]ApplyMsg, 0)

	rf.mu.Lock()

	// If commitIndex > lastApplied:
	// increment lastApplied, apply log[lastApplied] to state machine (§5.3)
	for i := rf.lastApplied + 1; i <= rf.commitIndex; i++ {
		applyMessages = append(applyMessages, ApplyMsg{
			CommandIndex: i,
			CommandValid: true,
			Command:      rf.log[i].Command,
		})
	}

	rf.mu.Unlock()

	// ! Use of channel must be outside the mutex

	rf.lastApplied = rf.commitIndex
	for _, msg := range applyMessages {
		rf.applyCh <- msg
	}
}
