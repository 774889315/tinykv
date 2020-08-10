// Copyright 2015 The etcd Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package raft

import (
	"errors"
	pb "github.com/pingcap-incubator/tinykv/proto/pkg/eraftpb"
	"math/rand"
	"sort"
)

// None is a placeholder node ID used when there is no leader.
const None uint64 = 0

// StateType represents the role of a node in a cluster.
type StateType uint64

const (
	StateFollower StateType = iota
	StateCandidate
	StateLeader
)

var stmap = [...]string{
	"StateFollower",
	"StateCandidate",
	"StateLeader",
}

func (st StateType) String() string {
	return stmap[uint64(st)]
}

// ErrProposalDropped is returned when the proposal is ignored by some cases,
// so that the proposer can be notified and fail fast.
var ErrProposalDropped = errors.New("raft proposal dropped")

// Config contains the parameters to start a raft.
type Config struct {
	// ID is the identity of the local raft. ID cannot be 0.
	ID uint64

	// peers contains the IDs of all nodes (including self) in the raft cluster. It
	// should only be set when starting a new raft cluster. Restarting raft from
	// previous configuration will panic if peers is set. peer is private and only
	// used for testing right now.
	peers []uint64

	// ElectionTick is the number of Node.Tick invocations that must pass between
	// elections. That is, if a follower does not receive any message from the
	// leader of current term before ElectionTick has elapsed, it will become
	// candidate and start an election. ElectionTick must be greater than
	// HeartbeatTick. We suggest ElectionTick = 10 * HeartbeatTick to avoid
	// unnecessary leader switching.
	ElectionTick int
	// HeartbeatTick is the number of Node.Tick invocations that must pass between
	// heartbeats. That is, a leader sends heartbeat messages to maintain its
	// leadership every HeartbeatTick ticks.
	HeartbeatTick int

	// Storage is the storage for raft. raft generates entries and states to be
	// stored in storage. raft reads the persisted entries and states out of
	// Storage when it needs. raft reads out the previous state and configuration
	// out of storage when restarting.
	Storage Storage
	// Applied is the last applied index. It should only be set when restarting
	// raft. raft will not return entries to the application smaller or equal to
	// Applied. If Applied is unset when restarting, raft might return previous
	// applied entries. This is a very application dependent configuration.
	Applied uint64
}

func (c *Config) validate() error {
	if c.ID == None {
		return errors.New("cannot use none as id")
	}

	if c.HeartbeatTick <= 0 {
		return errors.New("heartbeat tick must be greater than 0")
	}

	if c.ElectionTick <= c.HeartbeatTick {
		return errors.New("election tick must be greater than heartbeat tick")
	}

	if c.Storage == nil {
		return errors.New("storage cannot be nil")
	}

	return nil
}

// Progress represents a follower’s progress in the view of the leader. Leader maintains
// progresses of all followers, and sends entries to the follower based on its progress.
type Progress struct {
	Match, Next uint64
}

type Raft struct {
	id uint64

	Term uint64
	Vote uint64

	// the log
	RaftLog *RaftLog

	// log replication progress of each peers
	Prs map[uint64]*Progress

	// this peer's role
	State StateType

	// votes records
	votes map[uint64]bool

	// msgs need to send
	msgs []pb.Message

	// the leader id
	Lead uint64

	// heartbeat interval, should send
	heartbeatTimeout int
	// baseline of election interval
	electionTimeout int
	currentElectionTimeout int
	// number of ticks since it reached last heartbeatTimeout.
	// only leader keeps heartbeatElapsed.
	heartbeatElapsed int
	// number of ticks since it reached last electionTimeout
	electionElapsed int

	// leadTransferee is id of the leader transfer target when its value is not zero.
	// Follow the procedure defined in section 3.10 of Raft phd thesis.
	// (https://web.stanford.edu/~ouster/cgi-bin/papers/OngaroPhD.pdf)
	// (Used in 3A leader transfer)
	leadTransferee uint64

	// Only one conf change may be pending (in the log, but not yet
	// applied) at a time. This is enforced via PendingConfIndex, which
	// is set to a value >= the log index of the latest pending
	// configuration change (if any). Config changes are only allowed to
	// be proposed if the leader's applied index is greater than this
	// value.
	// (Used in 3A conf change)
	PendingConfIndex uint64
}

// newRaft return a raft peer with the given config
func newRaft(c *Config) *Raft {
	if err := c.validate(); err != nil {
		panic(err.Error())
	}
	// Your Code Here (2A).
	prs := make(map[uint64]*Progress, len(c.peers))
	vt := make(map[uint64]bool, len(c.peers))
	for i := 0; i < len(c.peers); i++ {
		prs[c.peers[i]] = &Progress{}
	}
	r := &Raft{
		id: c.ID,
		Term: 0,
		RaftLog: newLog(c.Storage),
		State: StateFollower,
		electionTimeout: c.ElectionTick,
		heartbeatTimeout: c.HeartbeatTick,
		Prs: prs,
		votes: vt,
	}
	r.initTimer()
	return r
}

// sendAppend sends an append RPC with new entries (if any) and the
// current commit index to the given peer. Returns true if a message was sent.
func (r *Raft) sendAppend(to uint64) bool {
	// Your Code Here (2A).
	var ents []*pb.Entry
	for i := r.Prs[to].Match; i < uint64(len(r.RaftLog.entries)); i++ {
		//if r.RaftLog.entries[i].Data == nil {
		//	r.Prs[to].Match++
		//	continue
		//}
		ents = append(ents, &r.RaftLog.entries[i])
	}
	if len(ents) == 0 {
		return false
	}
	logTerm, _ := r.RaftLog.Term(r.Prs[to].Match)
	msg := pb.Message {
		MsgType: pb.MessageType_MsgAppend,
		To: to,
		From: r.id,
		Term: r.Term,
		LogTerm: logTerm,
		Index: r.Prs[to].Match,
		Entries: ents,
		Commit: r.RaftLog.committed,
	}
	r.msgs = append(r.msgs, msg)
	return true
}

// sendHeartbeat sends a heartbeat RPC to the given peer.
func (r *Raft) sendHeartbeat(to uint64) {
	// Your Code Here (2A).
	msg := pb.Message {
		MsgType: pb.MessageType_MsgHeartbeat,
		To: to,
		From: r.id,
		Term: r.Term,
		LogTerm: 0,
		Index: 0,
	}
	if to == r.id {
		r.Step(msg)
	} else {
		r.msgs = append(r.msgs, msg)
	}
}

func (r *Raft) sendRequestVote(to uint64) {
	// Your Code Here (2A).
	index := r.RaftLog.LastIndex()
	logTerm, _ := r.RaftLog.Term(index)
	msg := pb.Message {
		MsgType: pb.MessageType_MsgRequestVote,
		To: to,
		From: r.id,
		Term: r.Term,
		LogTerm: logTerm,
		Index: index,
	}
	r.msgs = append(r.msgs, msg)
}

func (r *Raft) sendRequestVoteResponse(to uint64, reject bool) {
	// Your Code Here (2A).
	msg := pb.Message {
		MsgType: pb.MessageType_MsgRequestVoteResponse,
		To: to,
		From: r.id,
		Term: r.Term,
		Reject: reject,
	}
	r.msgs = append(r.msgs, msg)
}

// tick advances the internal logical clock by a single tick.
func (r *Raft) tick() {
	// Your Code Here (2A).
	switch r.State {
	case StateLeader:
		r.heartbeatElapsed++
		if r.heartbeatElapsed >= r.heartbeatTimeout {
			r.heartbeatElapsed = 0
			r.Step(pb.Message {
				MsgType: pb.MessageType_MsgBeat,
				To: r.id,
				From: r.id,
				Term: r.Term,
				LogTerm: 0,
				Index: 0,
			})
		}
	case StateFollower:
		fallthrough
	case StateCandidate:
		r.electionElapsed++
		if r.electionElapsed >= r.currentElectionTimeout {
			r.Step(pb.Message{From: r.id, To: r.id, MsgType: pb.MessageType_MsgHup})
		}
	}
}

func (r *Raft) initTimer() {
	r.electionElapsed = 0
	r.heartbeatElapsed = 0
	r.currentElectionTimeout = r.electionTimeout + rand.Intn(r.electionTimeout)
}

// becomeFollower transform this peer's state to Follower
func (r *Raft) becomeFollower(term uint64, lead uint64) {
	// Your Code Here (2A).
	r.State = StateFollower
	r.Term = term
	r.Lead = lead
	r.initTimer()
}

// becomeCandidate transform this peer's state to candidate
func (r *Raft) becomeCandidate() {
	// Your Code Here (2A).
	r.State = StateCandidate
	r.Term++
	r.votes = make(map[uint64]bool)
	r.VotedFrom(r.id)
	r.initTimer()
}

// becomeLeader transform this peer's state to leader
func (r *Raft) becomeLeader() {
	// Your Code Here (2A).
	// NOTE: Leader should propose a noop entry on its term
	r.State = StateLeader
	noop := pb.Entry {
		Term: r.Term,
		Index: r.RaftLog.LastIndex() + 1,
	}
	//r.RaftLog.entries = append(r.RaftLog.entries, noop)
	r.Step(pb.Message {
		MsgType: pb.MessageType_MsgPropose,
		From: r.id,
		To: r.id,
		Term: r.Term,
		Entries: []*pb.Entry{&noop},
	})
	//r.RaftLog.committed = r.RaftLog.LastIndex()
	r.initTimer()
}

func (r *Raft) VotedFrom(from uint64) {
	r.votes[from] = true
	if len(r.votes) > len(r.Prs) / 2 {
		r.becomeLeader()
	}
}

// Step the entrance of handle message, see `MessageType`
// on `eraftpb.proto` for what msgs should be handled
func (r *Raft) Step(m pb.Message) error {
	// Your Code Here (2A).
	if r.Term < m.Term {
		r.becomeFollower(m.Term, m.From)
		r.Vote = None
	}

	switch m.MsgType {
	case pb.MessageType_MsgHup:
		if r.State == StateLeader {
			return nil
		}
		r.becomeCandidate()
		for i, _ := range r.Prs {
			if i != r.id {
				r.sendRequestVote(i)
			}
		}
	case pb.MessageType_MsgBeat:
		if r.State != StateLeader {
			return nil
		}
		for i, _ := range r.Prs {
			r.sendHeartbeat(i)
		}
	case pb.MessageType_MsgPropose:
		if r.State != StateLeader {
			return nil
		}
		for _, e := range m.Entries {
			e.Index = r.RaftLog.LastIndex() + 1
			e.Term = r.Term
			r.RaftLog.entries = append(r.RaftLog.entries, *e)
		}
		for i, _ := range r.Prs {
			if i == r.id {
				continue
			}
			r.sendAppend(i)
		}
		if len(r.Prs) == 1 {
			r.RaftLog.committed = r.RaftLog.LastIndex()
		}
	case pb.MessageType_MsgAppend:
		if r.Term <= m.Term {
			r.becomeFollower(m.Term, m.From)
		}
		if logTerm, _ := r.RaftLog.Term(m.Index); logTerm != m.LogTerm {
			r.msgs = append(r.msgs, pb.Message {
				MsgType: pb.MessageType_MsgAppendResponse,
				To: m.From,
				From: m.To,
				Term: r.Term,
				Reject: true,
			})
			return nil
		}
		for i := range m.Entries {
			if logTerm, _ := r.RaftLog.Term(m.Entries[i].Index); logTerm != 0 {
				if logTerm < m.Entries[i].Term {
					r.RaftLog.entries = r.RaftLog.entries[:m.Entries[i].Index - 1]
					r.RaftLog.stabled = m.Entries[i].Index - 1
				} else {
					if logTerm == m.Entries[i].Term {
						continue
					}
				}
			}
			r.RaftLog.entries = append(r.RaftLog.entries, *m.Entries[i])
		}
		r.RaftLog.committed = m.Commit
		r.msgs = append(r.msgs, pb.Message {
			MsgType: pb.MessageType_MsgAppendResponse,
			To: m.From,
			From: m.To,
			Term: r.Term,
			Reject: false,
		})
	case pb.MessageType_MsgAppendResponse:
		if r.State != StateLeader {
			return nil
		}
		resort := r.Prs[m.From].Match <= r.RaftLog.committed && m.Index > r.RaftLog.committed
		r.Prs[m.From].Match = m.Index
		if resort {
			match := make(uint64Slice, len(r.Prs))
			j := 0
			for i := range r.Prs {
				match[j] = r.Prs[i].Match
				j++
			}
			sort.Sort(match)
			r.RaftLog.committed = match[(len(r.Prs) + 1) / 2]
		}
	case pb.MessageType_MsgRequestVote:
		index := r.RaftLog.LastIndex()
		logTerm, _ := r.RaftLog.Term(index)
		if logTerm > m.LogTerm || logTerm == m.LogTerm && index > m.Index {
			r.sendRequestVoteResponse(m.From, true)
			return nil
		}
		if r.State == StateLeader {
			r.State = StateFollower
		}
		switch r.Vote {
		case None:
			r.Vote = m.From
			fallthrough
		case m.From:
			r.sendRequestVoteResponse(m.From, false)
		default:
			r.sendRequestVoteResponse(m.From, true)
		}
	case pb.MessageType_MsgRequestVoteResponse:
		if !m.Reject && !r.votes[m.From] {
			r.VotedFrom(m.From)
		}
	case pb.MessageType_MsgHeartbeat:
		if r.Term <= m.Term {
			r.becomeFollower(m.Term, m.From)
		}
	}
	return nil
}

// handleAppendEntries handle AppendEntries RPC request
func (r *Raft) handleAppendEntries(m pb.Message) {
	// Your Code Here (2A).
}

// handleHeartbeat handle Heartbeat RPC request
func (r *Raft) handleHeartbeat(m pb.Message) {
	// Your Code Here (2A).
}

// handleSnapshot handle Snapshot RPC request
func (r *Raft) handleSnapshot(m pb.Message) {
	// Your Code Here (2C).
}

// addNode add a new node to raft group
func (r *Raft) addNode(id uint64) {
	// Your Code Here (3A).
}

// removeNode remove a node from raft group
func (r *Raft) removeNode(id uint64) {
	// Your Code Here (3A).
}
