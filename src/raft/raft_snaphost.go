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

}

func (rf *Raft) getLastSnapshotIndex() int {
	return rf.log[0].Index
}

func (rf *Raft) getLastSnaphostTerm() int {
	return rf.log[0].Term
}

// snaphost index: 3
// raft log: [3, 4, 5]
// raft idx: [0, 1, 2]
// input idx = 5
// return = 2
func (rf *Raft) convertIndex(idx int) int {
	return idx - rf.getLastSnapshotIndex()
}
