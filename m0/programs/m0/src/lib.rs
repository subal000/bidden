use anchor_lang::prelude::*;
use ephemeral_rollups_sdk::anchor::{commit, delegate, ephemeral};
use ephemeral_rollups_sdk::cpi::DelegateConfig;
use ephemeral_rollups_sdk::ephem::MagicIntentBundleBuilder;

declare_id!("BseL5WXo2AZY5kqqhLbsmnH7FnXFFiKfMiNDAcojjmEJ");

pub const COUNTER_SEED: &[u8] = b"counter";

#[ephemeral]
#[program]
pub mod m0 {
    use super::*;

    pub fn initialize(ctx: Context<Initialize>) -> Result<()> {
        ctx.accounts.counter.count = 0;
        Ok(())
    }

    /// Hot path. Runs in the ER. Kept deliberately minimal.
    pub fn increment(ctx: Context<Increment>) -> Result<()> {
        ctx.accounts.counter.count += 1;
        Ok(())
    }

    /// L1. Lifts the counter into the ER.
    pub fn delegate(ctx: Context<DelegateInput>) -> Result<()> {
        ctx.accounts.delegate_counter(
            &ctx.accounts.payer,
            &[COUNTER_SEED, ctx.accounts.payer.key.as_ref()],
            DelegateConfig {
                // Validator identity is passed as the first remaining account.
                validator: ctx.remaining_accounts.first().map(|a| a.key()),
                ..Default::default()
            },
        )?;
        Ok(())
    }

    /// ER -> L1. Commits final state and releases the account back to L1.
    pub fn undelegate(ctx: Context<Undelegate>) -> Result<()> {
        // Flush the Anchor account to its underlying buffer before the CPI reads it.
        ctx.accounts.counter.exit(&crate::ID)?;
        MagicIntentBundleBuilder::new(
            ctx.accounts.payer.to_account_info(),
            ctx.accounts.magic_context.to_account_info(),
            ctx.accounts.magic_program.to_account_info(),
        )
        .commit_and_undelegate(&[ctx.accounts.counter.to_account_info()])
        .build_and_invoke()?;
        Ok(())
    }
}

#[derive(Accounts)]
pub struct Initialize<'info> {
    #[account(
        init_if_needed,
        payer = payer,
        space = 8 + Counter::INIT_SPACE,
        seeds = [COUNTER_SEED, payer.key().as_ref()],
        bump
    )]
    pub counter: Account<'info, Counter>,
    #[account(mut)]
    pub payer: Signer<'info>,
    pub system_program: Program<'info, System>,
}

#[derive(Accounts)]
pub struct Increment<'info> {
    /// CHECK: seeds are validated against the payer below.
    #[account(mut, seeds = [COUNTER_SEED, payer.key().as_ref()], bump)]
    pub counter: Account<'info, Counter>,
    pub payer: Signer<'info>,
}

/// `#[delegate]` generates `delegate_counter()` from the field name and injects
/// buffer_counter / delegation_record_counter / delegation_metadata_counter.
#[delegate]
#[derive(Accounts)]
pub struct DelegateInput<'info> {
    #[account(mut)]
    pub payer: Signer<'info>,
    /// CHECK: delegated by seeds, validated by the delegation program.
    #[account(mut, del, seeds = [COUNTER_SEED, payer.key().as_ref()], bump)]
    pub counter: AccountInfo<'info>,
}

/// `#[commit]` injects magic_program and magic_context.
#[commit]
#[derive(Accounts)]
pub struct Undelegate<'info> {
    #[account(mut)]
    pub payer: Signer<'info>,
    #[account(mut, seeds = [COUNTER_SEED, payer.key().as_ref()], bump)]
    pub counter: Account<'info, Counter>,
}

#[account]
#[derive(InitSpace)]
pub struct Counter {
    pub count: u64,
}
