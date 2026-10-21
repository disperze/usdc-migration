# Signal support for x/usdcmigration in a future Agoric software upgrade (Noble USDC wind-down)

This is a signaling (text) proposal. It executes no code and moves no funds. Proposed by the [Disperze validator](https://x.com/Disperze_).

## Background

Circle is ending USDC support on Noble:

- Minting stopped on 2026-10-13.
- CCTP V1 burn limits start shrinking on 2026-10-31.
- USDC on Noble is fully paused on **2027-01-12**.

After the pause Noble rejects every incoming `uusdc` packet, so the Noble USDC voucher on Agoric (`ibc/FE98AAD68F02F03565E9FA39A5E627946699B2B07115889ED812D8BA639576A9`, received over `channel-62`) can no longer be redeemed. About **23,444.18 USDC** of this voucher is on Agoric (supply as of 2026-10-18), and many holders are inactive and will not move it in time.

## Proposal

A YES vote asks the Agoric core contributors to include the [`x/usdcmigration` module](https://github.com/disperze/usdc-migration) in a later software upgrade. That upgrade would:

- Reject new `uusdc` transfers arriving from Noble on `channel-62` (Noble refunds the sender).
- Reject outgoing IBC transfers of the voucher on every channel, so the voucher supply stays equal to the USDC Noble holds in escrow for Agoric.
- Let governance, in a separate proposal, return that escrowed USDC to a Noble multisig in a single IBC packet and freeze the voucher at the same height.

Holders are then repaid off-chain from a snapshot taken at that height. The module never touches user balances, mints nothing and needs no module-account permissions. The same escrow return was already used and tested when [Evmos was shut down](https://github.com/disperze/evmos/blob/release/v20/x/ibc/transfer/keeper/sudo_ibc.go).

## What changes for holders

From the upgrade height the voucher can no longer be sent back to Noble or to any other chain. Holders who want to redeem it themselves must do so before the upgrade. Local transfers and trades on Agoric keep working until the return proposal freezes the voucher.

Fast USDC flows that use this voucher stop at the upgrade; the date will be coordinated with the Fast USDC operators.

## Next steps if this passes

1. A software upgrade proposal that adds the module and sets the Noble channel (`channel-62`).
2. A return proposal (`MsgReturnToNoble` + `MsgSetSendEnabled`) naming the Noble multisig.

Each needs its own vote, and both must be done well before 2027-01-12.

## Repayment

- **Noble multisig signers:** the main validators of the Agoric network; further details will be provided if this proposal is approved.
- **Holders are repaid in:** USDC (Injective).
- **Repayment plan:** [docs/repayment.md](https://github.com/disperze/usdc-migration/blob/master/docs/repayment.md)

## Voting

- **YES:** include `x/usdcmigration` in a future software upgrade.
- **NO:** do not include it; holders must redeem by themselves before 2027-01-12.
- **ABSTAIN:** no preference, counts toward quorum.
- **NO WITH VETO:** the proposal is spam or harmful.
