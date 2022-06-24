package raft

import (
	"math/rand"
	"time"
)

const (
	FOLLOWER_STATE  = 1
	CANDIDATE_STATE = 2
	LEADER_STATE    = 3

	ELECTION_TIMEOUT_MAX = 300
	ELECTION_TIMEOUT_MIN = 150

	HEARTBEAT_TIMEOUT = 50
)

func (rf *Raft) changeState(nextState int, resetTime bool) {
	switch nextState {
	case FOLLOWER_STATE:
		rf.state = FOLLOWER_STATE
		rf.votedFor = -1
		rf.voteCount = 0

		if resetTime {
			rf.lastResetElectionTime = time.Now()
		}

	case CANDIDATE_STATE:
		rf.state = CANDIDATE_STATE
		rf.votedFor = rf.me
		rf.voteCount = 1
		rf.currentTerm += 1
		rf.startSendRequestVote()
		rf.lastResetElectionTime = time.Now()

	case LEADER_STATE:
		rf.state = LEADER_STATE
		rf.votedFor = -1
		rf.voteCount = 0
		rf.nextIndex = make([]int, len(rf.peers))
		for i := 0; i < len(rf.peers); i++ {
			rf.nextIndex[i] = rf.getLastLogIndex() + 1
		}
		rf.matchIndex = make([]int, len(rf.peers))
		rf.lastResetElectionTime = time.Now()
	}
}

// create a timeout [low, cap] ms
func createTimeout(low, cap int) <-chan struct{} {
	ch := make(chan struct{})
	go func() {
		time.Sleep(time.Millisecond * time.Duration(low+rand.Intn(cap-low+1)))
		ch <- struct{}{}
	}()

	return ch
}

func (rf *Raft) getLastLogIndex() int {
	return rf.log[len(rf.log)-1].Index
}

func (rf *Raft) getLastLogTerm() int {
	return rf.log[len(rf.log)-1].Term
}

func min(a int, b int) int {
	if a > b {
		return a
	} else {
		return b
	}
}
