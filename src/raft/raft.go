package raft

//
// this is an outline of the API that raft must expose to
// the service (or tester). see comments below for
// each of these functions for more details.
//
// rf = Make(...)
//   create a new Raft server.
// rf.Start(command interface{}) (index, term, isleader)
//   start agreement on a new log entry
// rf.GetState() (term, isLeader)
//   ask a Raft for its current term, and whether it thinks it is leader
// ApplyMsg
//   each time a new entry is committed to the log, each Raft peer
//   should send an ApplyMsg to the service (or tester)
//   in the same server.
//

import (
	//	"bytes"
	"math/rand"
	"sync"
	"sync/atomic"
	"time"

	//	"6.5840/labgob"
	"6.5840/labrpc"
)

// as each Raft peer becomes aware that successive log entries are
// committed, the peer should send an ApplyMsg to the service (or
// tester) on the same server, via the applyCh passed to Make(). set
// CommandValid to true to indicate that the ApplyMsg contains a newly
// committed log entry.
//
// in part 2D you'll want to send other kinds of messages (e.g.,
// snapshots) on the applyCh, but set CommandValid to false for these
// other uses.
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

// todo 尚未完善的Entry定义
type LogEntry struct {
	term int    // the term when receive the command
	data []byte // command
}

// A Go object implementing a single Raft peer.
type Raft struct {
	mu        sync.Mutex          // Lock to protect shared access to this peer's state
	peers     []*labrpc.ClientEnd // RPC end points of all peers
	persister *Persister          // Object to hold this peer's persisted state
	me        int                 // this peer's index into peers[]
	dead      int32               // set by Kill()
	// Your data here (2A, 2B, 2C).
	// Look at the paper's Figure 2 for a description of what
	// state a Raft server must maintain.
	currentTerm int // latest term this raft has seen
	votedFor    int // index of target peer

	// todo 待完善 当前用不到
	log         []LogEntry // log
	state       int        // follower、candidate、leader
	heartBeatCh chan interface{}
	voteCh      chan bool // check if got a vote request
	hasLeader   bool
}

// return currentTerm and whether this server
// believes it is the leader.
func (rf *Raft) GetState() (int, bool) {

	var term int
	var isleader bool
	// Your code here (2A).

	rf.mu.Lock()
	defer rf.mu.Unlock()

	term = rf.currentTerm
	isleader = rf.state == Leader

	return term, isleader
}

// save Raft's persistent state to stable storage,
// where it can later be retrieved after a crash and restart.
// see paper's Figure 2 for a description of what should be persistent.
// before you've implemented snapshots, you should pass nil as the
// second argument to persister.Save().
// after you've implemented snapshots, pass the current snapshot
// (or nil if there's not yet a snapshot).
func (rf *Raft) persist() {
	// Your code here (2C).
	// Example:
	// w := new(bytes.Buffer)
	// e := labgob.NewEncoder(w)
	// e.Encode(rf.xxx)
	// e.Encode(rf.yyy)
	// raftstate := w.Bytes()
	// rf.persister.Save(raftstate, nil)
}

// restore previously persisted state.
func (rf *Raft) readPersist(data []byte) {
	if data == nil || len(data) < 1 { // bootstrap without any state?
		return
	}
	// Your code here (2C).
	// Example:
	// r := bytes.NewBuffer(data)
	// d := labgob.NewDecoder(r)
	// var xxx
	// var yyy
	// if d.Decode(&xxx) != nil ||
	//    d.Decode(&yyy) != nil {
	//   error...
	// } else {
	//   rf.xxx = xxx
	//   rf.yyy = yyy
	// }
}

// the service says it has created a snapshot that has
// all info up to and including index. this means the
// service no longer needs the log through (and including)
// that index. Raft should now trim its log as much as possible.
func (rf *Raft) Snapshot(index int, snapshot []byte) {
	// Your code here (2D).

}

// example RequestVote RPC arguments structure.
// field names must start with capital letters!
type RequestVoteArgs struct {
	// Your data here (2A, 2B).
	Term         int
	CandidateId  int
	LastLogTerm  int
	LastLogIndex int
}

// example RequestVote RPC reply structure.
// field names must start with capital letters!
type RequestVoteReply struct {
	// Your data here (2A).
	Term        int  // = peer's term if > currentTerm
	VoteGranted bool // true if got vote
}

// example RequestVote RPC handler.
func (rf *Raft) RequestVote(args *RequestVoteArgs, reply *RequestVoteReply) {
	// Your code here (2A, 2B).

	rf.mu.Lock()
	//defer rf.mu.Unlock()
	// reject if it has voted or it is Candidate（vote for itself)
	if rf.votedFor != -1 || rf.state == Candidate {
		DPrintf("[%v] reject vote to [%v]", rf.me, args.CandidateId)
		rf.mu.Unlock()
		reply.VoteGranted = false
		return
	}
	term := rf.currentTerm

	var lastLogTerm int
	var lastLogIndex int
	if len(rf.log) > 0 {
		lastLogTerm = rf.log[len(rf.log)-1].term
		lastLogIndex = len(rf.log) - 1
	}
	rf.mu.Unlock()

	if args.Term < term {
		reply.VoteGranted = false

	} else if args.Term == term {
		// check lastLogTerm and lastLogIndex
		if args.LastLogTerm > lastLogTerm {
			reply.VoteGranted = true
			rf.votedFor = args.CandidateId
		} else if args.LastLogTerm == lastLogTerm && args.LastLogIndex >= lastLogIndex {
			reply.VoteGranted = true
			rf.votedFor = args.CandidateId
		}
	} else {
		rf.votedFor = args.CandidateId
		reply.VoteGranted = true
	}
	reply.Term = term

	// todo: send message that has got requestVote to the backup
	rf.heartBeatCh <- AppendEntriesArgs{
		Term: term,
	}
	//rf.voteCh <- true
}

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
func (rf *Raft) sendRequestVote(server int, args *RequestVoteArgs, reply *RequestVoteReply) bool {
	ok := rf.peers[server].Call("Raft.RequestVote", args, reply)
	return ok
}

// the service using Raft (e.g. a k/v server) wants to start
// agreement on the next command to be appended to Raft's log. if this
// server isn't the leader, returns false. otherwise start the
// agreement and return immediately. there is no guarantee that this
// command will ever be committed to the Raft log, since the leader
// may fail or lose an election. even if the Raft instance has been killed,
// this function should return gracefully.
//
// the first return value is the index that the command will appear at
// if it's ever committed. the second return value is the current
// term. the third return value is true if this server believes it is
// the leader.
func (rf *Raft) Start(command interface{}) (int, int, bool) {
	index := -1
	term := -1
	isLeader := true

	// Your code here (2B).

	return index, term, isLeader
}

// the tester doesn't halt goroutines created by Raft after each test,
// but it does call the Kill() method. your code can use killed() to
// check whether Kill() has been called. the use of atomic avoids the
// need for a lock.
//
// the issue is that long-running goroutines use memory and may chew
// up CPU time, perhaps causing later tests to fail and generating
// confusing debug output. any goroutine with a long-running loop
// should call killed() to check whether it should stop.
func (rf *Raft) Kill() {
	atomic.StoreInt32(&rf.dead, 1)
	// Your code here, if desired.
}

func (rf *Raft) killed() bool {
	z := atomic.LoadInt32(&rf.dead)
	return z == 1
}

func (rf *Raft) ticker() {
	for rf.killed() == false {

		// Your code here (2A)
		// Check if a leader election should be started.
		cond := sync.Cond{L: &rf.mu}

		done := false

		switch rf.state {
		case Follower:
			// this goroutine do election timeout
			go func() {
				for {
					// election timeout, 200ms-400ms
					eto := 200 + (rand.Int63() % 201)

					timer := time.AfterFunc(time.Duration(eto)*time.Millisecond, func() {
						// means no heartbeat or vote request
						rf.heartBeatCh <- false
					})
					// check heartbeat, false if timeout
					hasHeartbeat := <-rf.heartBeatCh
					DPrintf("[%v] get heartbeat [%v]", rf.me, hasHeartbeat)

					if hasHeartbeat != false {
						// reset election timeout
						hb, _ := hasHeartbeat.(AppendEntriesArgs)
						DPrintf("[%v] get heartbeat from [%v]", rf.me, hb.LeaderId)
						timer.Stop()

						// clear voted for
						rf.mu.Lock()
						rf.votedFor = -1
						rf.mu.Unlock()

					} else {
						// become candidate
						DPrintf("[%v] election timeout, become candidate", rf.me)

						rf.mu.Lock()
						defer rf.mu.Unlock()
						rf.state = Candidate
						rf.currentTerm++

						done = true
						cond.Broadcast()
						return
					}
				}
			}()
		case Leader:
			// todo :  if get new leader's heartbeat, how to handle it
			go func() {
				// leader do heartbeat forever
				DPrintf("[%v] i am the leader!!!!", rf.me)
				for {
					// store information to avoid deadlock
					rf.mu.Lock()
					term := rf.currentTerm
					leaderId := rf.me
					rf.mu.Unlock()

					args := AppendEntriesArgs{
						Term:     term,
						LeaderId: leaderId,
					}
					// send heartbeat to each peer
					// todo : change to rpc
					for idx, _ := range rf.peers {
						if idx == rf.me {
							continue
						}
						// send heartbeat
						go func() {
							reply := AppendEntriesReply{}
							DPrintf("[%v] leader send heartbeat to [%v]", rf.me, idx)
							ok := rf.sendAppendEntries(idx, &args, &reply)

							if !ok {
								DPrintf("[%v] append entries to server [%v] error", rf.me, idx)
							}
						}()
					}
					// wait for 100ms
					time.Sleep(HeartBeatTimeout)
				}
			}()
		case Candidate:
			rf.mu.Lock()

			term := rf.currentTerm
			var lastLogTerm int
			var lastLogIndex int

			l := len(rf.log)
			if l > 0 {
				lastLogTerm = rf.log[l-1].term
				lastLogIndex = l - 1
			}

			// vote for self
			rf.votedFor = rf.me

			rf.mu.Unlock()

			args := RequestVoteArgs{
				Term:         term,
				CandidateId:  rf.me,
				LastLogTerm:  lastLogTerm,
				LastLogIndex: lastLogIndex,
			}

			voteCount := atomic.Int32{}
			// add itself
			voteCount.Store(1)

			candidateCond := sync.Cond{L: &rf.mu}

			for idx, p := range rf.peers {
				if idx == rf.me {
					continue
				}
				go func(server int, peer *labrpc.ClientEnd) {
					DPrintf("[%v] requesting vote from [%v]", rf.me, server)
					//ok := rf.peers[server].Call("Raft.RequestVote", args, reply)
					reply := RequestVoteReply{}
					ok := peer.Call("Raft.RequestVote", &args, &reply)
					if !ok {
						DPrintf("[%v] failed to request vote from [%v]", rf.me, server)
					} else if reply.VoteGranted {
						DPrintf("[%v] success get vote from [%v]", rf.me, server)
						ct := int(voteCount.Add(1))

						// has been get majority votes
						if ct >= len(rf.peers)/2+1 {
							// become leader
							rf.mu.Lock()
							defer rf.mu.Lock()

							// check leader
							if !rf.hasLeader {
								DPrintf("[%v] become leader", rf.me)
								rf.state = Leader
								rf.hasLeader = true
								candidateCond.Broadcast()
							}
						}
					}
				}(idx, p)
			}
			// this goroutine check heartbeat
			go func() {
				data := <-rf.heartBeatCh
				hb, ok := data.(AppendEntriesArgs)
				if !ok {
					DPrintf("不知道什么错误")
				}

				DPrintf("[%v] candidate get heartbeat from [%v]", rf.me, hb.LeaderId)

				rf.mu.Lock()
				defer rf.mu.Lock()

				// return follower if accept the new leader
				if hb.Term >= rf.currentTerm {
					rf.state = Follower
					rf.hasLeader = true
				}
				candidateCond.Broadcast()
			}()
			//// this goroutine check requestVote
			//go func() {
			//
			//}()
			// this goroutine check timeout
			go func() {
				eto := 200 + (rand.Int63() % 201)
				time.Sleep(time.Duration(eto) * time.Millisecond)

				rf.mu.Lock()
				defer rf.mu.Unlock()
				// check leader
				if !rf.hasLeader {
					DPrintf("[%v] election timeout", rf.me)

					candidateCond.Broadcast()
				}
			}()

			// waiting for the result of election
			// 1)win  2) other leader  3) timeout
			rf.mu.Lock()
			candidateCond.Wait()
			rf.mu.Unlock()

			done = true
			cond.Broadcast()
		}

		// waiting for state change
		rf.mu.Lock()
		for !done {
			cond.Wait()
		}
		rf.mu.Unlock()

		// pause for a random amount of time between 50 and 350
		// milliseconds.
		//ms := 50 + (rand.Int63() % 300)
		//time.Sleep(time.Duration(ms) * time.Millisecond)
	}
}

// the service or tester wants to create a Raft server. the ports
// of all the Raft servers (including this one) are in peers[]. this
// server's port is peers[me]. all the servers' peers[] arrays
// have the same order. persister is a place for this server to
// save its persistent state, and also initially holds the most
// recent saved state, if any. applyCh is a channel on which the
// tester or service expects Raft to send ApplyMsg messages.
// Make() must return quickly, so it should start goroutines
// for any long-running work.
func Make(peers []*labrpc.ClientEnd, me int,
	persister *Persister, applyCh chan ApplyMsg) *Raft {
	rf := &Raft{}
	rf.peers = peers
	rf.persister = persister
	rf.me = me

	// Your initialization code here (2A, 2B, 2C).
	rf.state = Follower
	rf.votedFor = -1
	rf.heartBeatCh = make(chan interface{})

	// initialize from state persisted before a crash
	rf.readPersist(persister.ReadRaftState())

	// start ticker goroutine to start elections
	go rf.ticker()

	return rf
}

func (rf *Raft) AppendEntries(args *AppendEntriesArgs, reply *AppendEntriesReply) error {

	rf.mu.Lock()
	defer rf.mu.Unlock()
	rf.heartBeatCh <- args

	if args.Term < rf.currentTerm {
		reply.Term = rf.currentTerm
		return nil
	}

	l := len(rf.log)
	if args.PrevLogIndex < l && rf.log[args.PrevLogIndex].term != args.PrevLogTerm {
		reply.Success = false
		// todo : modify in Lab2B: log replication
		return nil
	}

	rf.currentTerm = args.Term

	return nil
}

func (rf *Raft) sendAppendEntries(server int, args *AppendEntriesArgs, apply *AppendEntriesReply) bool {
	ok := rf.peers[server].Call("Raft.AppendEntries", args, apply)

	return ok
}
