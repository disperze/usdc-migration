# Signal support for one Neutron software upgrade that adds x/usdcmigration and returns the Noble USDC escrow

This is a signaling (text) proposal. It executes no code and moves no funds. Proposed by the [Disperze validator](https://x.com/Disperze_).

## Background

Circle is ending USDC support on Noble:

- Minting stopped on 2026-10-13.
- CCTP V1 burn limits start shrinking on 2026-10-31.
- USDC on Noble is fully paused on **2027-01-12**.

After the pause Noble rejects every incoming `uusdc` packet, so the Noble USDC voucher on Neutron (`ibc/B559A80D62249C8AA07A380E2A2BEA6E5CA9A6F079C912C3A9E9B494105E4F81`, received over `channel-30`) can no longer be redeemed on Noble. About **2,057,515.89 USDC** of this voucher is on Neutron (supply as of 2026-10-26), and many holders are inactive and will not move it in time.

## Proposal

A YES vote asks the Neutron core contributors to ship **one software upgrade** that adds the [`x/usdcmigration` module](https://github.com/disperze/usdc-migration). At the upgrade height, its upgrade handler:

1. Rejects new `uusdc` transfers arriving from Noble on `channel-30` (Noble refunds the sender).
2. Rejects outgoing IBC transfers of the voucher on every channel.
3. Removes the voucher's IBC rate-limit path. Its quota (35% of the supply a day) would otherwise reject the return.
4. Returns Noble's USDC escrow for Neutron to a Noble multisig in a single IBC packet (`MsgReturnToNoble`). The amount is the voucher supply at that height, and the multisig address is fixed in the upgrade code.
5. Stores and instantiates the redemption contract described below.

Neutron governance can only execute software upgrades, so every step is in the upgrade itself. There is no follow-up return proposal, and nothing can be changed afterwards without another upgrade. The module never touches user balances and mints nothing. The same escrow return was already used and tested when [Evmos was shut down](https://github.com/disperze/evmos/blob/release/v20/x/ibc/transfer/keeper/sudo_ibc.go).

If the return packet times out or fails on Noble, nothing is lost: Noble's escrow is untouched, and the return is sent again to the same multisig.

## What changes for holders

Nothing changes until the upgrade height. Holders who only recently regained access to their funds can still withdraw them and send the voucher back to Noble themselves, for example with Skip:Go. The upgrade will be scheduled as late as safely possible before 2027-01-12, and its height announced well in advance.

From the upgrade height the voucher can no longer leave Neutron over IBC. It stays in holders' accounts and can still be moved on Neutron, so it can be redeemed in the contract below.

The native DEX is already in withdraw-only mode. Its liquidity providers withdraw the voucher as usual, then redeem it.

## Repayment: redemption contract

Holders are repaid in **USDC (Injective)** through a redemption contract on Neutron:

- A holder sends the Noble USDC voucher to the contract and receives the same amount of USDC (Injective) back. Each redemption is paid in full or rejected.
- There is no snapshot and no deadline: whoever holds the voucher can redeem it, at any time. Inactive holders lose nothing; their share waits in the contract.
- Vouchers in DEX pools and other contracts need no special handling: their users withdraw the voucher and redeem it.
- The contract has no owner, no pause and no withdraw function. Its wasm admin is the gov module, so only a software upgrade can change it.
- The Noble multisig converts the returned USDC to USDC (Injective) and sends it to the contract. Anyone can check the contract's balance against the voucher supply.

Other details:

- **Noble multisig signers:** the main validators of the Neutron network; further details will be provided if this proposal is approved.
- **Repayment plan:** [docs/repayment.md](https://github.com/disperze/usdc-migration/blob/master/docs/repayment.md) (option 1, redemption contract)

## Next steps if this passes

1. Publish the Neutron upgrade plan for review by Neutron's security maintainers: the upgrade handler, the Noble multisig and its signers, and the redemption contract.
2. Submit one software upgrade proposal with that plan, well before 2027-01-12.

## Voting

- **YES:** ship one software upgrade that adds `x/usdcmigration`, returns the Noble USDC escrow and deploys the redemption contract.
- **NO:** do not ship it; holders must redeem by themselves before 2027-01-12.
- **ABSTAIN:** no preference, counts toward quorum.
- **NO WITH VETO:** the proposal is spam or harmful.
