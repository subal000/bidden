package main

import (
	"encoding/binary"

	"github.com/gagliardetto/solana-go"
)

var (
	programID  = solana.MustPublicKeyFromBase58("BseL5WXo2AZY5kqqhLbsmnH7FnXFFiKfMiNDAcojjmEJ")
	delegationProgram = solana.MustPublicKeyFromBase58("DELeGGvXpWV2fqJUhqcF5ZSYMS4JTLjteaAMARRSaeSh")
	systemProgram     = solana.MustPublicKeyFromBase58("11111111111111111111111111111111")
	computeBudget     = solana.MustPublicKeyFromBase58("ComputeBudget111111111111111111111111111111")
	magicProgram      = solana.MustPublicKeyFromBase58("Magic11111111111111111111111111111111111111")
	magicContext      = solana.MustPublicKeyFromBase58("MagicContext1111111111111111111111111111111")
)

// Anchor discriminators, taken from target/idl/m0.json.
var (
	discInitialize = []byte{0xaf, 0xaf, 0x6d, 0x1f, 0x0d, 0x98, 0x9b, 0xed}
	discIncrement  = []byte{0x0b, 0x12, 0x68, 0x09, 0x68, 0xae, 0x3b, 0x21}
	discDelegate   = []byte{0x5a, 0x93, 0x4b, 0xb2, 0x55, 0x58, 0x04, 0x89}
	discUndelegate = []byte{0x83, 0x94, 0xb4, 0xc6, 0x5b, 0x68, 0x2a, 0xee}
)

func counterPDA(owner solana.PublicKey) solana.PublicKey {
	pda, _, err := solana.FindProgramAddress([][]byte{[]byte("counter"), owner.Bytes()}, programID)
	if err != nil {
		panic(err)
	}
	return pda
}

// Seeds come from the pda definitions in the IDL, not from memory.
func delegationPDAs(counter solana.PublicKey) (buffer, record, metadata solana.PublicKey) {
	buffer, _, _ = solana.FindProgramAddress([][]byte{[]byte("buffer"), counter.Bytes()}, programID)
	record, _, _ = solana.FindProgramAddress([][]byte{[]byte("delegation"), counter.Bytes()}, delegationProgram)
	metadata, _, _ = solana.FindProgramAddress([][]byte{[]byte("delegation-metadata"), counter.Bytes()}, delegationProgram)
	return
}

// setComputeUnitLimit makes otherwise identical transactions unique.
// Without it, N transactions with the same signer, instruction and cached
// blockhash all hash to one signature and the cluster dedupes them, which
// would make every throughput number meaningless.
func setComputeUnitLimit(units uint32) solana.Instruction {
	data := make([]byte, 5)
	data[0] = 0x02
	binary.LittleEndian.PutUint32(data[1:], units)
	return solana.NewInstruction(computeBudget, solana.AccountMetaSlice{}, data)
}

func ixInitialize(payer solana.PublicKey) solana.Instruction {
	return solana.NewInstruction(programID, solana.AccountMetaSlice{
		solana.NewAccountMeta(counterPDA(payer), true, false),
		solana.NewAccountMeta(payer, true, true),
		solana.NewAccountMeta(systemProgram, false, false),
	}, discInitialize)
}

func ixIncrement(payer solana.PublicKey) solana.Instruction {
	return solana.NewInstruction(programID, solana.AccountMetaSlice{
		solana.NewAccountMeta(counterPDA(payer), true, false),
		solana.NewAccountMeta(payer, true, true),
	}, discIncrement)
}

func ixDelegate(payer, validator solana.PublicKey) solana.Instruction {
	counter := counterPDA(payer)
	buffer, record, metadata := delegationPDAs(counter)
	return solana.NewInstruction(programID, solana.AccountMetaSlice{
		solana.NewAccountMeta(payer, true, true),
		solana.NewAccountMeta(buffer, true, false),
		solana.NewAccountMeta(record, true, false),
		solana.NewAccountMeta(metadata, true, false),
		solana.NewAccountMeta(counter, true, false),
		solana.NewAccountMeta(programID, false, false),
		solana.NewAccountMeta(delegationProgram, false, false),
		solana.NewAccountMeta(systemProgram, false, false),
		// remaining account: the ER validator identity
		solana.NewAccountMeta(validator, false, false),
	}, discDelegate)
}

func ixUndelegate(payer solana.PublicKey) solana.Instruction {
	return solana.NewInstruction(programID, solana.AccountMetaSlice{
		solana.NewAccountMeta(payer, true, true),
		solana.NewAccountMeta(counterPDA(payer), true, false),
		solana.NewAccountMeta(magicProgram, false, false),
		solana.NewAccountMeta(magicContext, true, false),
	}, discUndelegate)
}

func buildSigned(ixs []solana.Instruction, blockhash solana.Hash, payer solana.PrivateKey) (*solana.Transaction, error) {
	tx, err := solana.NewTransaction(ixs, blockhash, solana.TransactionPayer(payer.PublicKey()))
	if err != nil {
		return nil, err
	}
	_, err = tx.Sign(func(k solana.PublicKey) *solana.PrivateKey {
		if k.Equals(payer.PublicKey()) {
			return &payer
		}
		return nil
	})
	return tx, err
}

// incrementTx builds a uniquified increment transaction for the given signer.
func incrementTx(signer solana.PrivateKey, blockhash solana.Hash, uniq uint32) (*solana.Transaction, error) {
	return buildSigned([]solana.Instruction{
		setComputeUnitLimit(uniq),
		ixIncrement(signer.PublicKey()),
	}, blockhash, signer)
}
