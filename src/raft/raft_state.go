package raft

import (
	"math/rand"
	"time"
)

const (
	FOLLOWER_STATE  = 1
	CANDIDATE_STATE = 2
	LEADER_STATE    = 3

	ELECTION_TIMEOUT_MAX = 100
	ELECTION_TIMEOUT_MIN = 50

	HEARTBEAT_TIMEOUT = 27
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

func getRand(server int64) int {
	rand.Seed(time.Now().Unix() + server)
	return rand.Intn(ELECTION_TIMEOUT_MAX-ELECTION_TIMEOUT_MIN) + ELECTION_TIMEOUT_MIN
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
