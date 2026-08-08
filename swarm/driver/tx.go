package main

import (
	"encoding/binary"

	"github.com/gagliardetto/solana-go"
)

var (
	programID         = solana.MustPublicKeyFromBase58("BjMyKMPtFoWk7wXdSh4iz421H8PFWV1LbWcsJPKCzrhb")
	delegationProgram = solana.MustPublicKeyFromBase58("DELeGGvXpWV2fqJUhqcF5ZSYMS4JTLjteaAMARRSaeSh")
	systemProgram     = solana.MustPublicKeyFromBase58("11111111111111111111111111111111")
	magicProgram      = solana.MustPublicKeyFromBase58("Magic11111111111111111111111111111111111111")
	magicContext      = solana.MustPublicKeyFromBase58("MagicContext1111111111111111111111111111111")
)

// Anchor discriminators, read from target/idl/swarm.json.
var (
	discRegisterAgent  = []byte{0x87, 0x9d, 0x42, 0xc3, 0x02, 0x71, 0xaf, 0x1e}
	discPostJob        = []byte{0x22, 0xd0, 0x3a, 0xf8, 0x81, 0xea, 0xb3, 0xd3}
	discDelegateJob    = []byte{0x51, 0x69, 0x17, 0x55, 0xc2, 0xd1, 0xca, 0xbf}
	discDelegateAgent  = []byte{0x49, 0xfa, 0x79, 0xad, 0x92, 0x7a, 0xfe, 0x03}
	discSubmitBid      = []byte{0x13, 0xa4, 0xed, 0xfe, 0x40, 0x8b, 0xed, 0x5d}
	discAwardJob       = []byte{0x75, 0xbd, 0xa9, 0x09, 0x92, 0x41, 0x20, 0xda}
	discCommitUndelegate = []byte{0x09, 0x6c, 0x84, 0x57, 0xb8, 0x4c, 0x62, 0x54}
	discSettle         = []byte{0xaf, 0x2a, 0xb9, 0x57, 0x90, 0x83, 0x66, 0xd4}
)

// -------------------------------------------------------------------- PDAs

// jobPDA includes job_id, so each recording take gets a fresh Job account and
// every previous settled Job stays on the explorer.
func jobPDA(requester solana.PublicKey, jobID uint64) solana.PublicKey {
	id := make([]byte, 8)
	binary.LittleEndian.PutUint64(id, jobID)
	p, _, _ := solana.FindProgramAddress([][]byte{[]byte("job"), requester.Bytes(), id}, programID)
	return p
}

func escrowPDA(job solana.PublicKey) solana.PublicKey {
	p, _, _ := solana.FindProgramAddress([][]byte{[]byte("escrow"), job.Bytes()}, programID)
	return p
}

func agentPDA(authority solana.PublicKey) solana.PublicKey {
	p, _, _ := solana.FindProgramAddress([][]byte{[]byte("agent"), authority.Bytes()}, programID)
	return p
}

// Seeds taken from the pda definitions in the IDL, not from memory.
func delegationPDAs(target solana.PublicKey) (buffer, record, metadata solana.PublicKey) {
	buffer, _, _ = solana.FindProgramAddress([][]byte{[]byte("buffer"), target.Bytes()}, programID)
	record, _, _ = solana.FindProgramAddress([][]byte{[]byte("delegation"), target.Bytes()}, delegationProgram)
	metadata, _, _ = solana.FindProgramAddress([][]byte{[]byte("delegation-metadata"), target.Bytes()}, delegationProgram)
	return
}

// ------------------------------------------------------------ instructions

func ixRegisterAgent(authority solana.PublicKey, specialization uint8) solana.Instruction {
	data := append(append([]byte{}, discRegisterAgent...), specialization)
	return solana.NewInstruction(programID, solana.AccountMetaSlice{
		solana.NewAccountMeta(agentPDA(authority), true, false),
		solana.NewAccountMeta(authority, true, true),
		solana.NewAccountMeta(systemProgram, false, false),
	}, data)
}

func ixPostJob(requester solana.PublicKey, jobID uint64, descHash [32]byte, budget, deadlineSlot uint64) solana.Instruction {
	job := jobPDA(requester, jobID)
	data := append([]byte{}, discPostJob...)
	data = binary.LittleEndian.AppendUint64(data, jobID)
	data = append(data, descHash[:]...)
	data = binary.LittleEndian.AppendUint64(data, budget)
	data = binary.LittleEndian.AppendUint64(data, deadlineSlot)
	return solana.NewInstruction(programID, solana.AccountMetaSlice{
		solana.NewAccountMeta(job, true, false),
		solana.NewAccountMeta(escrowPDA(job), true, false),
		solana.NewAccountMeta(requester, true, true),
		solana.NewAccountMeta(systemProgram, false, false),
	}, data)
}

// ixDelegateJob lifts the Job alone. One delegated account per transaction:
// doing all seven at once overflows the BPF stack.
func ixDelegateJob(payer, requester, validator solana.PublicKey, jobID uint64) solana.Instruction {
	job := jobPDA(requester, jobID)
	buffer, record, metadata := delegationPDAs(job)
	return solana.NewInstruction(programID, solana.AccountMetaSlice{
		solana.NewAccountMeta(payer, true, true),
		solana.NewAccountMeta(requester, false, false),
		solana.NewAccountMeta(buffer, true, false),
		solana.NewAccountMeta(record, true, false),
		solana.NewAccountMeta(metadata, true, false),
		solana.NewAccountMeta(job, true, false),
		solana.NewAccountMeta(programID, false, false),
		solana.NewAccountMeta(delegationProgram, false, false),
		solana.NewAccountMeta(systemProgram, false, false),
		// remaining account: validator identity, identical for every delegation
		solana.NewAccountMeta(validator, false, false),
	}, binary.LittleEndian.AppendUint64(append([]byte{}, discDelegateJob...), jobID))
}

func ixDelegateAgent(payer, agentAuthority, validator solana.PublicKey) solana.Instruction {
	agent := agentPDA(agentAuthority)
	buffer, record, metadata := delegationPDAs(agent)
	return solana.NewInstruction(programID, solana.AccountMetaSlice{
		solana.NewAccountMeta(payer, true, true),
		solana.NewAccountMeta(agentAuthority, false, false),
		solana.NewAccountMeta(buffer, true, false),
		solana.NewAccountMeta(record, true, false),
		solana.NewAccountMeta(metadata, true, false),
		solana.NewAccountMeta(agent, true, false),
		solana.NewAccountMeta(programID, false, false),
		solana.NewAccountMeta(delegationProgram, false, false),
		solana.NewAccountMeta(systemProgram, false, false),
		solana.NewAccountMeta(validator, false, false),
	}, discDelegateAgent)
}

// ixSubmitBid is the hot path. Three accounts, two bytes of argument.
// The agent registry is written so the frontend can read every agent's live
// bid, not just the current leader's.
func ixSubmitBid(job, agentAuthority solana.PublicKey, bidBps uint16) solana.Instruction {
	data := binary.LittleEndian.AppendUint16(append([]byte{}, discSubmitBid...), bidBps)
	return solana.NewInstruction(programID, solana.AccountMetaSlice{
		solana.NewAccountMeta(job, true, false),
		solana.NewAccountMeta(agentPDA(agentAuthority), true, false),
		solana.NewAccountMeta(agentAuthority, false, true),
	}, data)
}

func ixAwardJob(job, winnerAgent, payer solana.PublicKey) solana.Instruction {
	return solana.NewInstruction(programID, solana.AccountMetaSlice{
		solana.NewAccountMeta(job, true, false),
		solana.NewAccountMeta(winnerAgent, true, false),
		solana.NewAccountMeta(payer, false, true),
	}, discAwardJob)
}

// ixCommitAndUndelegate schedules the intent for the Job plus every delegated
// registry, passed as remaining accounts.
func ixCommitAndUndelegate(payer, job solana.PublicKey, registries []solana.PublicKey) solana.Instruction {
	metas := solana.AccountMetaSlice{
		solana.NewAccountMeta(payer, true, true),
		solana.NewAccountMeta(job, true, false),
		solana.NewAccountMeta(magicProgram, false, false),
		solana.NewAccountMeta(magicContext, true, false),
	}
	for _, r := range registries {
		metas = append(metas, solana.NewAccountMeta(r, true, false))
	}
	return solana.NewInstruction(programID, metas, discCommitUndelegate)
}

func ixSettle(requester, winner solana.PublicKey, jobID uint64) solana.Instruction {
	job := jobPDA(requester, jobID)
	return solana.NewInstruction(programID, solana.AccountMetaSlice{
		solana.NewAccountMeta(job, true, false),
		solana.NewAccountMeta(escrowPDA(job), true, false),
		solana.NewAccountMeta(winner, true, false),
		solana.NewAccountMeta(agentPDA(winner), true, false),
		solana.NewAccountMeta(requester, true, false),
		solana.NewAccountMeta(systemProgram, false, false),
	}, binary.LittleEndian.AppendUint64(append([]byte{}, discSettle...), jobID))
}

// ------------------------------------------------------------ transactions

func buildSigned(ixs []solana.Instruction, blockhash solana.Hash, payer solana.PrivateKey, extra ...solana.PrivateKey) (*solana.Transaction, error) {
	tx, err := solana.NewTransaction(ixs, blockhash, solana.TransactionPayer(payer.PublicKey()))
	if err != nil {
		return nil, err
	}
	signers := append([]solana.PrivateKey{payer}, extra...)
	_, err = tx.Sign(func(k solana.PublicKey) *solana.PrivateKey {
		for i := range signers {
			if k.Equals(signers[i].PublicKey()) {
				return &signers[i]
			}
		}
		return nil
	})
	return tx, err
}

func systemTransfer(from, to solana.PublicKey, lamports uint64) solana.Instruction {
	data := make([]byte, 12)
	binary.LittleEndian.PutUint32(data[0:], 2)
	binary.LittleEndian.PutUint64(data[4:], lamports)
	return solana.NewInstruction(systemProgram, solana.AccountMetaSlice{
		solana.NewAccountMeta(from, true, true),
		solana.NewAccountMeta(to, true, false),
	}, data)
}
