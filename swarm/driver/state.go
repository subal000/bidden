package main

import (
	"encoding/binary"
	"fmt"

	"github.com/gagliardetto/solana-go"
)

type JobStatus uint8

const (
	StatusOpen JobStatus = iota
	StatusBidding
	StatusAwarded
	StatusSettled
)

func (s JobStatus) String() string {
	switch s {
	case StatusOpen:
		return "Open"
	case StatusBidding:
		return "Bidding"
	case StatusAwarded:
		return "Awarded"
	case StatusSettled:
		return "Settled"
	}
	return fmt.Sprintf("Unknown(%d)", uint8(s))
}

// Borsh packs without alignment padding, so these offsets are the field order
// in the Rust struct with the 8 byte Anchor discriminator in front.
type Job struct {
	JobID        uint64
	Requester    solana.PublicKey
	DescHash     [32]byte
	MaxBudgetBps uint16
	BestBidBps   uint16
	BestBidder   solana.PublicKey
	BidCount     uint32
	DeadlineSlot uint64
	Status       JobStatus
	Bump         uint8
}

const jobSize = 8 + 8 + 32 + 32 + 2 + 2 + 32 + 4 + 8 + 1 + 1 // 130

func parseJob(d []byte) (*Job, error) {
	if len(d) < jobSize {
		return nil, fmt.Errorf("job account too short: %d bytes, want %d", len(d), jobSize)
	}
	j := &Job{}
	j.JobID = binary.LittleEndian.Uint64(d[8:16])
	copy(j.Requester[:], d[16:48])
	copy(j.DescHash[:], d[48:80])
	j.MaxBudgetBps = binary.LittleEndian.Uint16(d[80:82])
	j.BestBidBps = binary.LittleEndian.Uint16(d[82:84])
	copy(j.BestBidder[:], d[84:116])
	j.BidCount = binary.LittleEndian.Uint32(d[116:120])
	j.DeadlineSlot = binary.LittleEndian.Uint64(d[120:128])
	j.Status = JobStatus(d[128])
	j.Bump = d[129]
	return j, nil
}

type AgentRegistry struct {
	Authority      solana.PublicKey
	Specialization uint8
	Completed      uint32
	Reputation     uint16
	Earned         uint64
	LastBidBps     uint16
	BidCount       uint32
	Bump           uint8
}

const agentSize = 8 + 32 + 1 + 4 + 2 + 8 + 2 + 4 + 1 // 62

func parseAgent(d []byte) (*AgentRegistry, error) {
	if len(d) < agentSize {
		return nil, fmt.Errorf("agent account too short: %d bytes, want %d", len(d), agentSize)
	}
	a := &AgentRegistry{}
	copy(a.Authority[:], d[8:40])
	a.Specialization = d[40]
	a.Completed = binary.LittleEndian.Uint32(d[41:45])
	a.Reputation = binary.LittleEndian.Uint16(d[45:47])
	a.Earned = binary.LittleEndian.Uint64(d[47:55])
	a.LastBidBps = binary.LittleEndian.Uint16(d[55:57])
	a.BidCount = binary.LittleEndian.Uint32(d[57:61])
	a.Bump = d[61]
	return a, nil
}

type Escrow struct {
	Job    solana.PublicKey
	Amount uint64
	Bump   uint8
}

const escrowSize = 8 + 32 + 8 + 1 // 49

func parseEscrow(d []byte) (*Escrow, error) {
	if len(d) < escrowSize {
		return nil, fmt.Errorf("escrow account too short: %d bytes, want %d", len(d), escrowSize)
	}
	e := &Escrow{}
	copy(e.Job[:], d[8:40])
	e.Amount = binary.LittleEndian.Uint64(d[40:48])
	e.Bump = d[48]
	return e, nil
}
