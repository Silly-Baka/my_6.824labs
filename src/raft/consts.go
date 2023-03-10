package raft

import "time"

const (
	Follower  = 0x1
	Candidate = 0x2
	Leader    = 0x3

	ElectionTimeout  = 400 * time.Millisecond
	HeartBeatTimeout = 100 * time.Millisecond

	HasNotVote = -1
)
