use anchor_lang::prelude::*;
use ephemeral_rollups_sdk::anchor::{commit, delegate, ephemeral};
use ephemeral_rollups_sdk::cpi::DelegateConfig;
use ephemeral_rollups_sdk::ephem::MagicIntentBundleBuilder;

declare_id!("BjMyKMPtFoWk7wXdSh4iz421H8PFWV1LbWcsJPKCzrhb");

pub const JOB_SEED: &[u8] = b"job";
pub const ESCROW_SEED: &[u8] = b"escrow";
pub const AGENT_SEED: &[u8] = b"agent";

/// Bids are basis points of the escrowed budget. 10000 = the full budget.
pub const FULL_BPS: u16 = 10_000;

#[ephemeral]
#[program]
pub mod swarm {
    use super::*;

    // ---------------------------------------------------------------- L1

    pub fn register_agent(ctx: Context<RegisterAgent>, specialization: u8) -> Result<()> {
        let agent = &mut ctx.accounts.agent;
        agent.authority = ctx.accounts.authority.key();
        agent.specialization = specialization;
        agent.completed = 0;
        agent.reputation = 5_000; // neutral starting reputation, in bps
        agent.earned = 0;
        agent.last_bid_bps = 0;
        agent.bid_count = 0;
        agent.bump = ctx.bumps.agent;
        Ok(())
    }

    /// Creates the job and funds the escrow. Escrow is a plain L1 PDA and is
    /// never delegated, so the money never leaves the base layer.
    pub fn post_job(
        ctx: Context<PostJob>,
        job_id: u64,
        desc_hash: [u8; 32],
        budget_lamports: u64,
        deadline_slot: u64,
    ) -> Result<()> {
        require!(budget_lamports > 0, SwarmError::EmptyBudget);

        let job = &mut ctx.accounts.job;
        job.job_id = job_id;
        job.requester = ctx.accounts.requester.key();
        job.desc_hash = desc_hash;
        job.max_budget_bps = FULL_BPS;
        job.best_bid_bps = FULL_BPS;
        job.best_bidder = Pubkey::default();
        job.bid_count = 0;
        job.deadline_slot = deadline_slot;
        job.status = JobStatus::Open;
        job.bump = ctx.bumps.job;

        let escrow = &mut ctx.accounts.escrow;
        escrow.job = job.key();
        escrow.amount = budget_lamports;
        escrow.bump = ctx.bumps.escrow;

        // Move the budget into escrow on top of its rent.
        anchor_lang::system_program::transfer(
            CpiContext::new(
                ctx.accounts.system_program.key(),
                anchor_lang::system_program::Transfer {
                    from: ctx.accounts.requester.to_account_info(),
                    to: escrow.to_account_info(),
                },
            ),
            budget_lamports,
        )?;
        Ok(())
    }

    /// Delegate one AgentRegistry into the ER.
    ///
    /// Delegation is deliberately one account per transaction. Doing all seven
    /// in a single instruction overflows the BPF stack (measured: 4416 bytes
    /// against the 4096 limit) and leaves only 29 bytes of transaction size
    /// headroom. Splitting across transactions is safe. Splitting the validator
    /// identity is not: every call here and in delegate_one_job must pass the
    /// same validator or any ER transaction touching both fails.
    pub fn delegate_one_agent(ctx: Context<DelegateOneAgent>) -> Result<()> {
        ctx.accounts.delegate_agent(
            &ctx.accounts.payer,
            &[AGENT_SEED, ctx.accounts.agent_authority.key.as_ref()],
            DelegateConfig {
                validator: ctx.remaining_accounts.first().map(|a| a.key()),
                ..Default::default()
            },
        )?;
        Ok(())
    }

    /// Delegate the Job into the ER. Same validator identity as every agent.
    pub fn delegate_one_job(ctx: Context<DelegateOneJob>, job_id: u64) -> Result<()> {
        let id = job_id.to_le_bytes();
        ctx.accounts.delegate_job(
            &ctx.accounts.payer,
            &[JOB_SEED, ctx.accounts.requester.key.as_ref(), &id],
            DelegateConfig {
                validator: ctx.remaining_accounts.first().map(|a| a.key()),
                ..Default::default()
            },
        )?;
        Ok(())
    }

    // ---------------------------------------------------------------- ER

    /// The hot path. Called hundreds of times per session inside the ER.
    ///
    /// Never returns an error. A losing race is a landed transaction with an
    /// unchanged best bid, not a failure, so agents need no retry logic and the
    /// demo counter stays honest. No CPIs, no loops, no reallocation.
    pub fn submit_bid(ctx: Context<SubmitBid>, bid_bps: u16) -> Result<()> {
        let job = &mut ctx.accounts.job;
        job.bid_count = job.bid_count.saturating_add(1);
        if job.status == JobStatus::Open {
            job.status = JobStatus::Bidding;
        }
        if job.status == JobStatus::Bidding && bid_bps < job.best_bid_bps {
            job.best_bid_bps = bid_bps;
            job.best_bidder = ctx.accounts.authority.key();
        }
        // The agent's own delegated account, so no cross-agent contention. This
        // is what lets the frontend render every agent's live bid natively
        // instead of only the current leader. Display only: Job.bid_count stays
        // the authoritative counter.
        let agent = &mut ctx.accounts.agent;
        agent.last_bid_bps = bid_bps;
        agent.bid_count = agent.bid_count.saturating_add(1);
        Ok(())
    }

    /// Closes bidding. Runs in the ER once the deadline slot has passed.
    /// Touches the Job and the winner's registry, which is the transaction that
    /// requires both to share a validator identity.
    pub fn award_job(ctx: Context<AwardJob>) -> Result<()> {
        let job = &mut ctx.accounts.job;
        require!(job.status == JobStatus::Bidding, SwarmError::BadState);
        require!(job.best_bidder != Pubkey::default(), SwarmError::NoBids);
        require_keys_eq!(
            ctx.accounts.winner_agent.authority,
            job.best_bidder,
            SwarmError::WrongWinner
        );

        job.status = JobStatus::Awarded;
        let agent = &mut ctx.accounts.winner_agent;
        agent.completed = agent.completed.saturating_add(1);
        Ok(())
    }

    /// Schedules the commit and undelegation. This only registers the intent.
    /// The L1 write is executed later by the validator, so the client must poll
    /// L1 until the Job owner flips back to this program.
    ///
    /// Pass every delegated AgentRegistry as a remaining account.
    pub fn commit_and_undelegate<'info>(
        ctx: Context<'info, CommitAndUndelegate<'info>>,
    ) -> Result<()> {
        let job = &ctx.accounts.job;
        require!(job.status == JobStatus::Awarded, SwarmError::BadState);
        job.exit(&crate::ID)?;

        let mut accounts = Vec::with_capacity(1 + ctx.remaining_accounts.len());
        accounts.push(ctx.accounts.job.to_account_info());
        for a in ctx.remaining_accounts.iter() {
            accounts.push(a.clone());
        }

        MagicIntentBundleBuilder::new(
            ctx.accounts.payer.to_account_info(),
            ctx.accounts.magic_context.to_account_info(),
            ctx.accounts.magic_program.to_account_info(),
        )
        .commit_and_undelegate(&accounts)
        .build_and_invoke()?;
        Ok(())
    }

    // ---------------------------------------------------------------- L1

    /// Pays the winner from escrow. Runs on L1 only after undelegation has
    /// landed and the Job is owned by this program again.
    pub fn settle(ctx: Context<Settle>, job_id: u64) -> Result<()> {
        let _ = job_id; // consumed by the seeds constraint
        let job = &mut ctx.accounts.job;
        require!(job.status == JobStatus::Awarded, SwarmError::BadState);
        require_keys_eq!(
            ctx.accounts.winner.key(),
            job.best_bidder,
            SwarmError::WrongWinner
        );

        let budget = ctx.accounts.escrow.amount;
        let payout = (budget as u128)
            .checked_mul(job.best_bid_bps as u128)
            .ok_or(SwarmError::MathOverflow)?
            .checked_div(FULL_BPS as u128)
            .ok_or(SwarmError::MathOverflow)? as u64;
        require!(payout <= budget, SwarmError::MathOverflow);

        // Escrow is owned by this program, so lamports move by direct mutation.
        // The `close = requester` constraint returns the rent and the unspent
        // remainder to the requester after this runs.
        let escrow_ai = ctx.accounts.escrow.to_account_info();
        let winner_ai = ctx.accounts.winner.to_account_info();
        **escrow_ai.try_borrow_mut_lamports()? = escrow_ai
            .lamports()
            .checked_sub(payout)
            .ok_or(SwarmError::EscrowUnderflow)?;
        **winner_ai.try_borrow_mut_lamports()? = winner_ai
            .lamports()
            .checked_add(payout)
            .ok_or(SwarmError::MathOverflow)?;

        let agent = &mut ctx.accounts.winner_agent;
        agent.earned = agent.earned.saturating_add(payout);

        job.status = JobStatus::Settled;
        Ok(())
    }
}

// ====================================================================== state

#[derive(AnchorSerialize, AnchorDeserialize, Clone, Copy, PartialEq, Eq, Debug, InitSpace)]
pub enum JobStatus {
    Open,
    Bidding,
    Awarded,
    Settled,
}

#[account]
#[derive(InitSpace)]
pub struct Job {
    /// Distinguishes runs by the same requester. Part of the PDA seeds, so
    /// every take gets a fresh Job account and old ones stay on the explorer.
    pub job_id: u64,
    pub requester: Pubkey,
    pub desc_hash: [u8; 32],
    pub max_budget_bps: u16,
    pub best_bid_bps: u16,
    pub best_bidder: Pubkey,
    pub bid_count: u32,
    pub deadline_slot: u64,
    pub status: JobStatus,
    pub bump: u8,
}

#[account]
#[derive(InitSpace)]
pub struct AgentRegistry {
    pub authority: Pubkey,
    pub specialization: u8,
    pub completed: u32,
    pub reputation: u16,
    pub earned: u64,
    /// Written by submit_bid so the frontend can read every agent's live bid
    /// without parsing transaction logs. Display only.
    pub last_bid_bps: u16,
    /// This agent's own bid tally. Display only: if it ever disagrees with
    /// Job.bid_count, the job total is authoritative.
    pub bid_count: u32,
    pub bump: u8,
}

/// Never delegated. Lives on L1 for the entire lifecycle.
#[account]
#[derive(InitSpace)]
pub struct Escrow {
    pub job: Pubkey,
    pub amount: u64,
    pub bump: u8,
}

#[error_code]
pub enum SwarmError {
    #[msg("job is not in the required state for this instruction")]
    BadState,
    #[msg("no bids were submitted")]
    NoBids,
    #[msg("winner does not match the recorded best bidder")]
    WrongWinner,
    #[msg("budget must be greater than zero")]
    EmptyBudget,
    #[msg("arithmetic overflow")]
    MathOverflow,
    #[msg("escrow has insufficient lamports")]
    EscrowUnderflow,
}

// =================================================================== contexts

#[derive(Accounts)]
pub struct RegisterAgent<'info> {
    #[account(
        init,
        payer = authority,
        space = 8 + AgentRegistry::INIT_SPACE,
        seeds = [AGENT_SEED, authority.key().as_ref()],
        bump
    )]
    pub agent: Account<'info, AgentRegistry>,
    #[account(mut)]
    pub authority: Signer<'info>,
    pub system_program: Program<'info, System>,
}

#[derive(Accounts)]
#[instruction(job_id: u64)]
pub struct PostJob<'info> {
    #[account(
        init,
        payer = requester,
        space = 8 + Job::INIT_SPACE,
        seeds = [JOB_SEED, requester.key().as_ref(), &job_id.to_le_bytes()],
        bump
    )]
    pub job: Account<'info, Job>,
    #[account(
        init,
        payer = requester,
        space = 8 + Escrow::INIT_SPACE,
        seeds = [ESCROW_SEED, job.key().as_ref()],
        bump
    )]
    pub escrow: Account<'info, Escrow>,
    #[account(mut)]
    pub requester: Signer<'info>,
    pub system_program: Program<'info, System>,
}

#[delegate]
#[derive(Accounts)]
pub struct DelegateOneJob<'info> {
    #[account(mut)]
    pub payer: Signer<'info>,
    /// CHECK: seed authority for the job PDA
    pub requester: UncheckedAccount<'info>,
    /// CHECK: delegated by seeds
    #[account(mut, del)]
    pub job: UncheckedAccount<'info>,
}

#[delegate]
#[derive(Accounts)]
pub struct DelegateOneAgent<'info> {
    #[account(mut)]
    pub payer: Signer<'info>,
    /// CHECK: seed authority for the agent PDA
    pub agent_authority: UncheckedAccount<'info>,
    /// CHECK: delegated by seeds
    #[account(mut, del)]
    pub agent: UncheckedAccount<'info>,
}

/// Three accounts. Deliberately the smallest context in the program.
/// `has_one` costs one pubkey comparison, where a seeds constraint would pay
/// for a PDA derivation on every one of several hundred calls.
#[derive(Accounts)]
pub struct SubmitBid<'info> {
    #[account(mut)]
    pub job: Account<'info, Job>,
    #[account(mut, has_one = authority)]
    pub agent: Account<'info, AgentRegistry>,
    pub authority: Signer<'info>,
}

#[derive(Accounts)]
pub struct AwardJob<'info> {
    #[account(mut)]
    pub job: Account<'info, Job>,
    #[account(mut)]
    pub winner_agent: Account<'info, AgentRegistry>,
    pub payer: Signer<'info>,
}

#[commit]
#[derive(Accounts)]
pub struct CommitAndUndelegate<'info> {
    #[account(mut)]
    pub payer: Signer<'info>,
    #[account(mut)]
    pub job: Account<'info, Job>,
    // every delegated AgentRegistry is passed as a remaining account
}

#[derive(Accounts)]
#[instruction(job_id: u64)]
pub struct Settle<'info> {
    #[account(
        mut,
        seeds = [JOB_SEED, requester.key().as_ref(), &job_id.to_le_bytes()],
        bump = job.bump,
        has_one = requester
    )]
    pub job: Account<'info, Job>,
    #[account(
        mut,
        seeds = [ESCROW_SEED, job.key().as_ref()],
        bump = escrow.bump,
        close = requester
    )]
    pub escrow: Account<'info, Escrow>,
    /// CHECK: verified against job.best_bidder in the handler
    #[account(mut)]
    pub winner: UncheckedAccount<'info>,
    #[account(
        mut,
        seeds = [AGENT_SEED, winner.key().as_ref()],
        bump = winner_agent.bump
    )]
    pub winner_agent: Account<'info, AgentRegistry>,
    /// CHECK: receives the escrow remainder and rent
    #[account(mut)]
    pub requester: UncheckedAccount<'info>,
    pub system_program: Program<'info, System>,
}
