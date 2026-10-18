# Signaling proposals

These are text proposals, one per chain in [integration.md](../../integration.md). Each one asks the chain's governance to approve including `x/usdcmigration` in a later software upgrade. They have no messages (`"messages": []`), so they execute nothing and move no funds. All 8 chains run gov v1 (SDK ≥ v0.50), where a proposal without messages is the standard text proposal.

This is the first of three votes on each chain:

1. **Signaling** (this folder): does the chain want the module?
2. **Software upgrade**: adds the module and sets `noble_channels`. Its wiring is in [integration.md](../../integration.md).
3. **Return**: `MsgReturnToNoble` + `MsgSetSendEnabled`. The scaffolds are in [../](../README.md).

Each file's `summary` is the full proposal text that voters see in explorers and wallets. It is written per chain: chain name, Noble channel, voucher denom, the voucher supply as of 2026-10-18, and chain-specific notes for Agoric (Fast USDC), Coreum (DEX orders) and Haqq (EVM precompile).

## Deposit

Each `deposit` is set to the chain's full `min_deposit`, so the proposal goes straight to voting. The amounts and periods are in the table in [../README.md](../README.md#minimum-deposit).

| Chain | `deposit` |
|---|---|
| Juno | 5,000 JUNO |
| Agoric | 5,000 BLD |
| Xion | 1,000 XION |
| Coreum | 50,000 CORE |
| Terra2 | 5,000 LUNA |
| Haqq | 6,000 ISLM |
| Impacthub (ixo) | 10,000 IXO |
| Archway | 5,000 ARCH |

## Before submitting

1. **Post on the chain's forum first** and leave it open for discussion for a few days. Most of these communities expect that before an on-chain vote.
2. **Replace the placeholders** in `summary` and `metadata`:
   - `REPLACE_WITH_SIGNERS`: who controls the Noble multisig.
   - `REPLACE_WITH_ASSET`: what holders are repaid in, e.g. Injective USDC.
   - `REPLACE_WITH_REPAYMENT_PLAN_URL`: how the snapshot and repayment work.
   - `REPLACE_WITH_FORUM_URL`: the discussion thread.
   - `ipfs://REPLACE_WITH_METADATA_CID`: proposal metadata (255 characters max), or `""`.
3. **Check with the core team.** The upgrade is theirs to ship, so agree on the target release and date before the vote.
4. **Submit:**

   ```sh
   <binary> tx gov submit-proposal docs/proposals/signaling/<chain>.json \
     --from <key> --chain-id <chain-id> --gas auto --gas-adjustment 1.5 --fees <fees>
   ```

## Timing

Leave room for all three votes before **2027-01-12**. The slowest case, Archway, has a 7-day voting period for each of the three votes. Add upgrade coordination, then relaying and checking the return, and there is little slack if signaling starts later than early November.

The text tells holders that **the upgrade itself ends self-redemption**: from that height the voucher can't leave the chain. Announce the upgrade height early, so holders who want to send their USDC back to Noble themselves can do it before then.
