package raft

import "time"

//
// example RequestVote RPC arguments structure.
// field names must start with capital letters!
//
type RequestVoteArgs struct {
	// Your data here (2A, 2B).
	Term         int
	CandidateId  int
	LastLogIndex int
	LastLogTerm  int
}

//
// example RequestVote RPC reply structure.
// field names must start with capital letters!
//
type RequestVoteReply struct {
	// Your data here (2A).
	Term        int
	VoteGranted bool
}

//
// example RequestVote RPC handler.
//
func (rf *Raft) RequestVote(args *RequestVoteArgs, reply *RequestVoteReply) {
	// Your code here (2A, 2B).
	rf.mu.Lock()
	defer rf.mu.Unlock()

	if args.Term < rf.currentTerm {
		// Reply false if term < currentTerm (§5.1)
		reply.Term = rf.currentTerm
		reply.VoteGranted = false
		return
	}

	if args.Term > rf.currentTerm {
		// If RPC request or response contains term T > currentTerm:
		// set currentTerm = T, convert to follower (§5.1)
		rf.currentTerm = args.Term
		rf.changeState(FOLLOWER_STATE, false)
	}

	if !rf.isUpToDate(args.LastLogTerm, args.LastLogIndex) {
		reply.Term = rf.currentTerm
		reply.VoteGranted = false
		return
	}

	if rf.votedFor != -1 && rf.votedFor != args.CandidateId && args.Term == rf.currentTerm {
		reply.Term = rf.currentTerm
		reply.VoteGranted = false
		return
	}

	// vote for the candidate
	rf.votedFor = args.CandidateId
	rf.lastResetElectionTime = time.Now()
	rf.currentTerm = args.Term

	reply.Term = rf.currentTerm
	reply.VoteGranted = true
}

func (rf *Raft) isUpToDate(candidateTerm int, candidateIndex int) bool {
	term, index := rf.getLastLogTerm(), rf.getLastLogIndex()
	return candidateTerm > term || (candidateTerm == term && candidateIndex >= index)
}

//
// example code to send a RequestVote RPC to a server.
// server is the index of the target server in rf.peers[].
// expects RPC arguments in args.
// fills in *reply with RPC reply, so caller should
// pass &reply.
// the types of the args and reply passed to Call() must be
// the same as the types of the arguments declared in the
// handler function (including whether they are pointers).
//
// The labrpc package simulates a lossy network, in which servers
// may be unreachable, and in which requests and replies may be lost.
// Call() sends a request and waits for a reply. If a reply arrives
// within a timeout interval, Call() returns true; otherwise
// Call() returns false. Thus Call() may not return for a while.
// A false return can be caused by a dead server, a live server that
// can't be reached, a lost request, or a lost reply.
//
// Call() is guaranteed to return (perhaps after a delay) *except* if the
// handler function on the server side does not return.  Thus there
// is no need to implement your own timeouts around Call().
//
// look at the comments in ../labrpc/labrpc.go for more details.
//
// if you're having trouble getting RPC to work, check that you've
// capitalized all field names in structs passed over RPC, and
// that the caller passes the address of the reply struct with &, not
// the struct itself.
//
func (rf *Raft) sendRequestVote(server int, args *RequestVoteArgs, reply *RequestVoteReply) bool {
	ok := rf.peers[server].Call("Raft.RequestVote", args, reply)
	return ok
}

func (rf *Raft) startSendRequestVote() {
	var handle = func(server int) {
		rf.mu.Lock()
		args := RequestVoteArgs{
			Term:         rf.currentTerm,
			CandidateId:  rf.me,
			LastLogIndex: rf.getLastLogIndex(),
			LastLogTerm:  rf.getLastLogTerm(),
		}
		reply := RequestVoteReply{}
		rf.mu.Unlock()

		result := rf.sendRequestVote(server, &args, &reply)
		if !result {
			return
		}

		rf.mu.Lock()
		defer rf.mu.Unlock()

		if rf.state != CANDIDATE_STATE || args.Term < rf.currentTerm {
			// invaild state
			return
		}

		if reply.VoteGranted && rf.currentTerm == args.Term {
			rf.voteCount += 1
			if rf.voteCount >= len(rf.peers)/2+1 {
				// immediately become leader and send heartbeat
				// prevent others from being the leader
				rf.changeState(LEADER_STATE, true)
				return
			}
		}

		if reply.Term > args.Term {
			// If RPC request or response contains term T > currentTerm:
			// set currentTerm = T, convert to follower (§5.1)
			if rf.currentTerm < reply.Term {
				rf.currentTerm = reply.Term
			}
			rf.changeState(FOLLOWER_STATE, false)
			return
		}
	}

	for peer := range rf.peers {
		if peer != rf.me {
			go handle(peer)
		}
	}
}

func (rf *Raft) candidateTicker() {
	for !rf.killed() {
		nowTime := time.Now()
		<-createTimeout(ELECTION_TIMEOUT_MIN, ELECTION_TIMEOUT_MAX)

		rf.mu.Lock()
		if rf.lastResetElectionTime.Before(nowTime) && rf.state != LEADER_STATE {
			rf.changeState(CANDIDATE_STATE, true)
		}
		rf.mu.Unlock()
	}
}
