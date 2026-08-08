import * as anchor from "@coral-xyz/anchor";
import { Program, web3 } from "@coral-xyz/anchor";
import * as fs from "fs";
import * as os from "os";

const IDL = JSON.parse(fs.readFileSync("target/idl/m0.json", "utf8"));

const L1_RPC = process.env.L1_RPC || "https://api.devnet.solana.com";
const ER_RPC = process.env.ER_RPC || "https://devnet-as.magicblock.app";
const ER_WS = process.env.ER_WS || "wss://devnet-as.magicblock.app";
// Must match the region of ER_RPC. Asia devnet.
const VALIDATOR = new web3.PublicKey(
  process.env.VALIDATOR || "MAS1Dt9qreoRMQ14YQuhg8UTZMMzDdKhmkZMECCzk57",
);
const N = Number(process.env.N || 100);

const COUNTER_SEED = Buffer.from("counter");

function loadWallet(): web3.Keypair {
  const path = process.env.WALLET || `${os.homedir()}/.config/solana/id.json`;
  return web3.Keypair.fromSecretKey(
    Uint8Array.from(JSON.parse(fs.readFileSync(path, "utf8"))),
  );
}

const step = (n: string, msg: string) => console.log(`\n[${n}] ${msg}`);
const ok = (msg: string) => console.log(`    ok  ${msg}`);

async function main() {
  const kp = loadWallet();
  const wallet = new anchor.Wallet(kp);

  const l1 = new anchor.AnchorProvider(
    new web3.Connection(L1_RPC, { commitment: "confirmed" }),
    wallet,
    { commitment: "confirmed" },
  );
  const er = new anchor.AnchorProvider(
    new web3.Connection(ER_RPC, {
      wsEndpoint: ER_WS,
      commitment: "confirmed",
    }),
    wallet,
    { commitment: "confirmed" },
  );

  const program = new Program(IDL, l1);
  const programER = new Program(IDL, er);

  const [counterPDA] = web3.PublicKey.findProgramAddressSync(
    [COUNTER_SEED, kp.publicKey.toBuffer()],
    program.programId,
  );

  console.log("program   ", program.programId.toBase58());
  console.log("wallet    ", kp.publicKey.toBase58());
  console.log("counter   ", counterPDA.toBase58());
  console.log("L1        ", L1_RPC);
  console.log("ER        ", ER_RPC);
  console.log("validator ", VALIDATOR.toBase58());

  const readL1 = async () => {
    const info = await l1.connection.getAccountInfo(counterPDA);
    if (!info) return { count: null, owner: null };
    return {
      count: Number(info.data.readBigUInt64LE(8)),
      owner: info.owner.toBase58(),
    };
  };

  // ---------------------------------------------------------------- 1. init
  step("1/6", "initialize counter on L1");
  {
    const pre = await readL1();
    if (pre.owner && pre.owner !== program.programId.toBase58()) {
      throw new Error(
        `counter is owned by ${pre.owner}, not the program. It is probably still ` +
          `delegated from a previous run. Undelegate it first.`,
      );
    }
    const sig = await program.methods
      .initialize()
      .accounts({ payer: kp.publicKey })
      .rpc();
    ok(`init sig ${sig}`);
    const after = await readL1();
    ok(`count on L1 = ${after.count} (owner ${after.owner})`);
    if (after.count !== 0) throw new Error(`expected 0, got ${after.count}`);
  }

  // ------------------------------------------------------------ 2. delegate
  step("2/6", "delegate counter into the ER");
  {
    const sig = await program.methods
      .delegate()
      .accounts({ payer: kp.publicKey, counter: counterPDA })
      .remainingAccounts([
        { pubkey: VALIDATOR, isSigner: false, isWritable: false },
      ])
      .rpc();
    ok(`delegate sig ${sig}`);

    const owner = (await readL1()).owner;
    ok(`counter now owned by ${owner} on L1 (delegation program)`);

    // Wait for the ER validator to pick the account up.
    const deadline = Date.now() + 30_000;
    for (;;) {
      const info = await er.connection.getAccountInfo(counterPDA);
      if (info && info.owner.equals(program.programId)) {
        ok(`counter live in ER, count = ${Number(info.data.readBigUInt64LE(8))}`);
        break;
      }
      if (Date.now() > deadline) throw new Error("timed out waiting for ER pickup");
      await new Promise((r) => setTimeout(r, 500));
    }
  }

  // ----------------------------------------------------------- 3. increment
  step("3/6", `increment ${N} times against the ER`);
  const t0 = Date.now();
  {
    for (let i = 0; i < N; i++) {
      await programER.methods
        .increment()
        .accounts({ counter: counterPDA, payer: kp.publicKey })
        .rpc({ skipPreflight: true, commitment: "processed" });
      if ((i + 1) % 20 === 0) {
        console.log(`    ${i + 1}/${N}  (${Date.now() - t0}ms)`);
      }
    }
  }
  const elapsed = Date.now() - t0;
  ok(`${N} ER transactions in ${elapsed}ms (${(elapsed / N).toFixed(1)}ms avg)`);

  // -------------------------------------------------------- 4. read in ER
  step("4/6", "read counter in the ER");
  {
    const info = await er.connection.getAccountInfo(counterPDA);
    const count = Number(info!.data.readBigUInt64LE(8));
    ok(`count in ER = ${count}`);
    if (count !== N) throw new Error(`expected ${N} in ER, got ${count}`);

    const l1now = await readL1();
    ok(`count on L1 is still ${l1now.count} (not yet committed)`);
  }

  // ------------------------------------------- 5. commit and undelegate
  step("5/6", "commit + undelegate back to L1");
  {
    const sig = await programER.methods
      .undelegate()
      .accounts({ payer: kp.publicKey, counter: counterPDA })
      .rpc({ skipPreflight: true });
    ok(`undelegate sig (ER) ${sig}`);
  }

  // -------------------------------------------------------- 6. verify L1
  step("6/6", "verify final state on L1");
  {
    const deadline = Date.now() + 60_000;
    for (;;) {
      const { count, owner } = await readL1();
      if (owner === program.programId.toBase58() && count === N) {
        ok(`counter back on L1, owner ${owner}, count = ${count}`);
        break;
      }
      if (Date.now() > deadline) {
        throw new Error(`timed out. last seen owner=${owner} count=${count}`);
      }
      await new Promise((r) => setTimeout(r, 1000));
    }
  }

  console.log(`\nMILESTONE 0 PASSED  (${N} ER txs, final L1 value ${N})`);
  console.log(
    `explorer: https://explorer.solana.com/address/${counterPDA.toBase58()}?cluster=devnet`,
  );
}

main().catch((e) => {
  console.error("\nFAILED:", e.message || e);
  if (e.logs) console.error(e.logs.join("\n"));
  process.exit(1);
});
