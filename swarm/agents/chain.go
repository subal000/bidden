package main

import (
	"encoding/binary"
	"errors"
	"fmt"
	"sync/atomic"

	"github.com/gagliardetto/solana-go"
)

var errNoKey = errors.New("no keypair for agent")

var (
	programID     = solana.MustPublicKeyFromBase58("BjMyKMPtFoWk7wXdSh4iz421H8PFWV1LbWcsJPKCzrhb")
	computeBudget = solana.MustPublicKeyFromBase58("ComputeBudget111111111111111111111111111111")
	discSubmitBid = []byte{0x13, 0xa4, 0xed, 0xfe, 0x40, 0x8b, 0xed, 0x5d}
)

// uniq makes otherwise identical bids into distinct transactions.
//
// An agent that cannot improve the price holds, resubmitting the same value.
// Same signer, same instruction, same bid, same cached blockhash hashes to one
// signature, and the cluster dedupes it as already processed. The bids silently
// stop landing, and the closer the auction gets to the floor the worse it gets:
// measured 10.0 then 6.4 then 3.3 bids/s across three consecutive runs before
// this was added. A varying compute unit limit costs nothing and makes every
// transaction unique.
var uniq atomic.Uint32

func setComputeUnitLimit(units uint32) solana.Instruction {
	data := make([]byte, 5)
	data[0] = 0x02
	binary.LittleEndian.PutUint32(data[1:], units)
	return solana.NewInstruction(computeBudget, solana.AccountMetaSlice{}, data)
}

func jobPDA(requester solana.PublicKey, jobID uint64) solana.PublicKey {
	id := make([]byte, 8)
	binary.LittleEndian.PutUint64(id, jobID)
	p, _, _ := solana.FindProgramAddress([][]byte{[]byte("job"), requester.Bytes(), id}, programID)
	return p
}

func agentPDA(authority solana.PublicKey) solana.PublicKey {
	p, _, _ := solana.FindProgramAddress([][]byte{[]byte("agent"), authority.Bytes()}, programID)
	return p
}

func buildSubmitBid(job solana.PublicKey, signer solana.PrivateKey, bidBps uint16, bh solana.Hash) (*solana.Transaction, error) {
	data := binary.LittleEndian.AppendUint16(append([]byte{}, discSubmitBid...), bidBps)
	ix := solana.NewInstruction(programID, solana.AccountMetaSlice{
		solana.NewAccountMeta(job, true, false),
		solana.NewAccountMeta(agentPDA(signer.PublicKey()), true, false),
		solana.NewAccountMeta(signer.PublicKey(), false, true),
	}, data)
	cu := setComputeUnitLimit(60_000 + uniq.Add(1)%100_000)
	tx, err := solana.NewTransaction([]solana.Instruction{cu, ix}, bh, solana.TransactionPayer(signer.PublicKey()))
	if err != nil {
		return nil, err
	}
	_, err = tx.Sign(func(k solana.PublicKey) *solana.PrivateKey {
		if k.Equals(signer.PublicKey()) {
			return &signer
		}
		return nil
	})
	return tx, err
}

var statusNames = []string{"Open", "Bidding", "Awarded", "Settled"}

// parseJobState decodes the on chain Job payload. Offsets follow the Borsh
// field order with the 8 byte Anchor discriminator in front.
func parseJobState(ai *AccountInfo) (JobState, error) {
	if ai == nil {
		return JobState{}, errors.New("job account not found")
	}
	d := ai.Data
	if len(d) < 130 {
		return JobState{}, fmt.Errorf("job account too short: %d bytes", len(d))
	}
	status := "Unknown"
	if int(d[128]) < len(statusNames) {
		status = statusNames[d[128]]
	}
	var bidder solana.PublicKey
	copy(bidder[:], d[84:116])
	best := binary.LittleEndian.Uint16(d[82:84])
	// bid_count is read straight from the payload, never inferred from a count
	// of websocket events. The payload stays authoritative under any load.
	return JobState{
		BidCount:   binary.LittleEndian.Uint32(d[116:120]),
		BestBidBps: best,
		BestBidder: bidder.String(),
		Status:     status,
	}, nil
}
