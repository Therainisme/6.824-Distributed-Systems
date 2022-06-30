package raft

import (
	"fmt"
	"log"
)

// Debugging
const Debug = false

func DPrintf(format string, a ...interface{}) (n int, err error) {
	if Debug {
		log.Printf(format, a...)
	}
	return
}

func (rf *Raft) bprint(format string, a ...interface{}) {
	if Debug {
		rf.mu.Lock()
		defer rf.mu.Unlock()
		info := fmt.Sprintf("[Peer:%d Term: %d, Type: %s, Commit: %d, Kill: %v] ", rf.me, rf.currentTerm, rf.getStateStr(), rf.commitIndex, rf.killed())
		formatInfo := info + format
		log.Printf(formatInfo, a...)
	}
}

func (rf *Raft) uprint(format string, a ...interface{}) {
	if Debug {
		info := fmt.Sprintf("[Peer:%d Term: %d, Type: %s, Commit: %d, Kill: %v] ", rf.me, rf.currentTerm, rf.getStateStr(), rf.commitIndex, rf.killed())
		formatInfo := info + format
		log.Printf(formatInfo, a...)
	}
}

func (rf *Raft) getStateStr() string {
	switch rf.state {
	case FOLLOWER_STATE:
		return "FOL"
	case CANDIDATE_STATE:
		return "CAN"
	default:
		return "DER"
	}
}

func min(x, y int) int {
	if x < y {
		return x
	}
	return y
}
