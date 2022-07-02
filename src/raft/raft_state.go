package raft

import (
	"context"
	"math/rand"
	"time"
)

const (
	LEADER_STATE    = 1
	CANDIDATE_STATE = 2
	FOLLOWER_STATE  = 3
)

func (rf *Raft) gotoState(nextState int, block bool) {
	if block {
		rf.mu.Lock()
		defer rf.mu.Unlock()
	}

	defer func() {
		go rf.persist()
	}()

	// cancel ctx of prevous state
	rf.cancelPrevStateCtx()
	stateCtx, cancel := context.WithCancel(context.Background())
	rf.cancelPrevStateCtx = cancel

	switch nextState {
	case FOLLOWER_STATE:

		rf.state = FOLLOWER_STATE
		rf.votedFor = -1

		// rf.uprint("ungrade to follower")

		go rf.followerTicker(stateCtx)

	case CANDIDATE_STATE:

		rf.state = CANDIDATE_STATE
		rf.votedFor = rf.me
		rf.currentTerm += 1

		rf.uprint("timeout start to elect......")

		go rf.candidateTicker(stateCtx)

	case LEADER_STATE:

		rf.state = LEADER_STATE
		rf.nextIndex = make([]int, len(rf.peers))
		rf.matchIndex = make([]int, len(rf.peers))
		for peer := range rf.peers {
			rf.nextIndex[peer] = rf.getLastLogIndex() + 1
		}

		rf.uprint("successful become leader!")

		go rf.leaderTicker(stateCtx)
	}
}

func (rf *Raft) followerTicker(stateCtx context.Context) {
	// rf.bprint("start new follower ticker")

	for !rf.killed() {
		select {
		case <-stateCtx.Done():
			return

		case <-createTimeout(500, 551):
			rf.gotoState(CANDIDATE_STATE, true)

		case <-rf.receiveAppendEntries:

		case <-rf.receiveRequestVote:
		}
	}
}

func (rf *Raft) candidateTicker(stateCtx context.Context) {
	defer func() {
		go rf.persist()
	}()

	grantCount := 1
	failCount := 0
	voteReplyCh := make(chan VoteReply, len(rf.peers))

	var send = func(server int) {
		rf.mu.Lock()
		args := RequestVoteArgs{
			Term:         rf.currentTerm,
			CandidateId:  rf.me,
			LastLogIndex: rf.getLastLogIndex(),
			LastLogTerm:  rf.getLastLogTerm(),
		}
		rf.mu.Unlock()

		reply := RequestVoteReply{}
		VoteReply := rf.sendRequestVote(server, &args, &reply)

		rf.bprint("get vote %+v", VoteReply)

		rf.mu.Lock()
		defer rf.mu.Unlock()

		if args.Term != rf.currentTerm {
			return
		}

		voteReplyCh <- VoteReply
	}

	for peer := range rf.peers {
		if peer != rf.me {
			go send(peer)
		}
	}

	for !rf.killed() {
		select {
		case <-rf.receiveAppendEntries:
			rf.gotoState(FOLLOWER_STATE, true)
			return

		case <-stateCtx.Done():
			return

		case <-createTimeout(500, 551):
			rf.gotoState(CANDIDATE_STATE, true)
			return

		case reply := <-voteReplyCh:

			rf.mu.Lock()

			if !reply.ok {
				failCount += 1
				if failCount > len(rf.peers)/2 {
					rf.currentTerm -= 1

					rf.gotoState(CANDIDATE_STATE, false)

					rf.mu.Unlock()
					return
				}
			}

			if reply.Term > rf.currentTerm {
				// If RPC request or response contains term T > currentTerm:
				// set currentTerm = T, convert to follower (§5.1)
				rf.currentTerm = reply.Term

				rf.gotoState(FOLLOWER_STATE, false)

				rf.mu.Unlock()
				return
			}

			if reply.ok && reply.VoteGranted {
				grantCount += 1

				if grantCount > len(rf.peers)/2 {

					rf.gotoState(LEADER_STATE, false)

					rf.mu.Unlock()
					return
				}
			}

			rf.mu.Unlock()
		}
	}
}

func (rf *Raft) leaderTicker(stateCtx context.Context) {

	var send = func() {
		rf.mu.Lock()

		for peer := range rf.peers {
			if peer == rf.me {
				continue
			}

			// If last log index ≥ nextIndex for a follower:
			// send AppendEntries RPC with log entries starting at nextIndex

			prevLogIndex := rf.nextIndex[peer] - 1
			prevLogTerm := rf.log[rf.convertIndex(prevLogIndex)].Term
			entries := make([]LogEntry, len(rf.log[rf.convertIndex(rf.nextIndex[peer]):]))
			copy(entries, rf.log[rf.convertIndex(rf.nextIndex[peer]):])

			args := AppendEntriesArgs{
				Term:         rf.currentTerm,
				LeaderId:     rf.me,
				PrevLogIndex: prevLogIndex,
				PrevLogTerm:  prevLogTerm,
				Entries:      entries,
				LeaderCommit: rf.commitIndex,
			}
			go rf.sendAppendEntries(peer, args, &AppendEntriesReply{})
		}

		rf.mu.Unlock()
	}

	go send()

	for !rf.killed() {
		select {
		case <-stateCtx.Done():
			return

		case <-createTimeout(50, 51):
			// Upon election: send initial empty AppendEntries RPCs (heartbeat) to each server;
			// repeat during idle periods to prevent election timeouts (§5.2)

			// rf.bprint("sending")
			go send()
		}
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
