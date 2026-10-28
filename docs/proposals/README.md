# Return proposals

One scaffold per chain in [integration.md](../integration.md), except Juno, which is no longer included. Each one is step 3 of the [governance runbook](../../README.md#governance-runbook): `MsgReturnToNoble` followed by `MsgSetSendEnabled`, executed atomically.

Before any of this, each chain votes on a text proposal approving the integration: see [signaling/](signaling/README.md).

Submit the return proposals only **after** the chain's software upgrade has added `x/usdcmigration` and set `noble_channels`. Before that, the chain does not know `/usdcmigration.v1.MsgReturnToNoble` and rejects the proposal.

## Minimum deposit

Queried on 2026-10-18 (Neutron on 2026-10-26) from each chain's `cosmos/gov/v1/params/deposit`. A proposal enters the voting period as soon as its total deposit reaches `min_deposit`. Each scaffold's `deposit` field is set to the full `min_deposit`, so the proposal goes straight to voting.

| Chain | Chain ID | `min_deposit` (to reach voting) | Initial deposit at submit | Deposit period | Voting period | Quorum |
|---|---|---|---|---|---|---|
| Agoric | `agoric-3` | **5,000 BLD** (`5000000000ubld`) | none | 2 days | 3 days | 33.4% |
| Xion | `xion-mainnet-1` | **1,000 XION** (`1000000000uxion`) | none | 2 days | 4 days | 33.4% |
| Coreum | `coreum-mainnet-1` | **50,000 CORE** (`50000000000ucore`) | 50% = 25,000 CORE | 14 days | 5 days | 40% |
| Neutron | `neutron-1` | **300,000 NTRN** (`300000000000untrn`) | 50% = 150,000 NTRN | 3 days | 7 days | 30% |
| Terra2 | `phoenix-1` | **5,000 LUNA** (`5000000000uluna`) | none | 7 days | 7 days | 30% |
| Haqq | `haqq_11235-1` | **6,000 ISLM** (`6000000000000000000000aISLM`) | none | 2 days | 3 days | 33.4% |
| Impacthub (ixo) | `ixo-5` | **10,000 IXO** (`10000000000uixo`) | 10% = 1,000 IXO | 2 days | 5 days | 33.4% |
| Archway | `archway-1` | **5,000 ARCH** (`5000000000000000000000aarch`) | none | 14 days | 7 days | 33.4% |

"Initial deposit at submit" is `min_initial_deposit_ratio × min_deposit`: the smallest deposit the chain accepts when the proposal is submitted. The vote threshold (`threshold` in the gov params) is 50% on every chain. The deposit is refunded unless the proposal is vetoed (or doesn't reach quorum, on chains that burn deposits for that).

Expedited proposals (1-day voting) are not used in the scaffolds. On Agoric and Archway `expedited_min_deposit` is set in `stake`, a denom that doesn't exist on those chains, so expedited proposals can't be submitted there at all.

## Values already filled in (checked on-chain 2026-10-26)

| Chain | Gov module address (`authority`) | Source channel | Noble channel | Noble escrow address | Noble escrow balance (USDC) |
|---|---|---|---|---|---|
| Agoric | `agoric10d07y265gmmuvt4z0w9aw880jnsr700jgl36x9` | `channel-62` | `channel-21` | `noble16knm868dfqlvxhpdsv5w6n2an6rgsn3shzldlm` | 87,016.97 |
| Xion | `xion10d07y265gmmuvt4z0w9aw880jnsr700jctf8qc` | `channel-2` | `channel-113` | `noble10zfpsus4h27eqgrjar7yj2mwer3q84zkwmsgml` | 67,319.81 |
| Coreum | `core10d07y265gmmuvt4z0w9aw880jnsr700jfvt73g` | `channel-19` | `channel-49` | `noble1u929fykjps0qny9enq0rgdsavau7amszl7dj9y` | 55,000.34 |
| Neutron | `neutron10d07y265gmmuvt4z0w9aw880jnsr700j7a68v5` | `channel-30` | `channel-18` | `noble1gw2v3j8jxpmnq73rmu3vjz5egle5eep4xvvjtu` | 2,058,476.26 |
| Terra2 | `terra10d07y265gmmuvt4z0w9aw880jnsr700juxf95n` | `channel-253` | `channel-30` | `noble12vssh8spx53qzg532xpj22sa7e85ts4x76e5rm` | 311,071.38 |
| Haqq | `haqq10d07y265gmmuvt4z0w9aw880jnsr700jc9xkg9` | `channel-4` | `channel-32` | `noble1y376tnlmyu23jaxdjj3xdpsf2crxfgpaly3ynx` | 68,769.51 |
| Impacthub (ixo) | `ixo10d07y265gmmuvt4z0w9aw880jnsr700j9hdhjq` | `channel-26` | `channel-15` | `noble186mpcj53q3hgfnzdu9wkteczs236vdclxm264k` | 54,302.02 |
| Archway | `archway10d07y265gmmuvt4z0w9aw880jnsr700j0f0puy` | `channel-29` | `channel-12` | `noble1004rr9e45qpm2ylvxymphm8sryvxq3a2kcyjgw` | 33,621.45 |
| **Total** | | | | | **2,735,577.73** |

The source channel is the chain's end of its transfer channel to Noble, and the value set in `noble_channels`. The Noble channel is the counterparty, taken from the [chain registry `_IBC` files](https://github.com/cosmos/chain-registry/tree/master/_IBC). The escrow address is `sha256("ics20-1" + 0x00 + "transfer/<noble channel>")[:20]` in bech32 `noble`, and matches Noble's `ibc/apps/transfer/v1/channels/<noble channel>/ports/transfer/escrow_address`. Balances are its `uusdc` balance at Noble height 62,477,983 (Neutron's at 62,493,049).

Every scaffold sets `amount` (required) to the escrow balance in `uusdc` at that height. The voucher supply on the source chain is not a substitute: it matches the escrow on Xion, Coreum, Haqq and ixo, but not elsewhere:

| Chain | Voucher supply | Noble escrow | `amount` |
|---|---|---|---|
| Agoric | 23,444.18 | 87,016.97 | `87016970394` |
| Neutron | 2,057,515.89 | 2,058,476.26 | `2058476260015` |
| Terra2 | 310,071.86 | 311,071.38 | `311071376277` |
| Archway | 33,623.14 | 33,621.45 | `33621448005` |

On Agoric, Neutron and Terra2 the escrow is above the supply; on Archway it is below. Noble refunds any amount above the escrow balance (error ack), so a stale, too-high amount releases nothing. The escrow only stops moving at the upgrade height, so set every `amount` again from Noble after the upgrade, once pending packets are relayed (step 1 of [Before submitting](#before-submitting)).

Every voucher denom in the scaffolds matches `query ibc-transfer denom-hash transfer/<channel>/uusdc` on that chain. That includes the three that are computed rather than taken from the chain registry (Coreum, Haqq, ixo).

## Before submitting

Replace the placeholders in each file:

- `noble1REPLACE_WITH_MULTISIG`: the Noble multisig. It appears in `receiver` and in `summary`.
- `ipfs://REPLACE_WITH_METADATA_CID`: the proposal metadata (255 characters max), or `""`.
- `REPLACE_WITH_REPAYMENT_PLAN_URL` in `summary`: where holders can read how they are repaid from the snapshot.

Then:

1. **Relay every pending packet on the channel, in both directions, after the upgrade.** Packets sent before the upgrade height still move funds when they land: a Noble→chain `uusdc` packet now gets an error ack and Noble refunds the sender from escrow, a chain→Noble voucher packet releases USDC from escrow, and an expired packet refunds its sender once it is timed out. Until they are all received, acknowledged or timed out, the escrow balance and the voucher supply can still change, and the `amount` would be stale. Both ends hold old commitments today (on 2026-10-26, 2,443 on Noble `channel-21` and 717 on Agoric `channel-62`), so do not expect zero; what matters is that nothing is left to relay.

   ```sh
   hermes query packet pending --chain <noble-chain-id> --port transfer --channel <noble channel>
   hermes clear packets --chain <noble-chain-id> --port transfer --channel <noble channel>
   ```

   Run the query again until it reports no unreceived packets or acks on either side. Then read the final escrow balance.
2. Set `amount` to Noble's final escrow balance for the counterparty channel: `GET <noble-rest>/cosmos/bank/v1beta1/balances/<escrow address>/by_denom?denom=uusdc`.
3. Submit:

   ```sh
   <binary> tx gov submit-proposal docs/proposals/<chain>.json \
     --from <key> --chain-id <chain-id> --gas auto --gas-adjustment 1.5 --fees <fees>
   ```

4. After it passes, relay the return packet and verify it (runbook step 4). Then take the snapshot at the `height` from the `usdcmigration_return_to_noble` event.

Notes:

- **Keep the message order.** On Neutron the rate-limit removal must come before the return, otherwise the rate limit rejects the return packet. The return comes before the freeze. On both release lines the return packet involves no bank send, so `SendEnabled` does not apply to it and a freeze placed first would not block it.
- **`timeout_seconds` is `86400`** (1 day), which gives relayers time. The maximum is 7 days. If the packet times out, Noble's escrow is untouched, and the retry is a new proposal with only `MsgReturnToNoble`. The voucher stays frozen.
- **Leave `memo` empty** (see [integration.md](../integration.md)).
- **Coreum** mainnet runs `v7.0.0` today, but integration.md analysed tx-chain v8. Re-check the wiring against the release the upgrade starts from.
- **Agoric**: coordinate the date with the Fast USDC operators. The freeze stops Fast USDC flows that use this voucher.
  - **Why the escrow is above the voucher supply.** When the voucher moves into an Agoric smart contract (ERTP purse), `x/vbank` burns it from the bank account, and mints it again when it moves back out (`GrabCoins` / `SendCoins` in agoric-sdk [`x/vbank/keeper/keeper.go`](https://github.com/Agoric/agoric-sdk/blob/master/golang/cosmos/x/vbank/keeper/keeper.go)). USDC held in contracts therefore drops out of the bank supply, while Noble's escrow still holds it. In vstorage, `published.agoricNames.vbankAsset` maps `ibc/FE98AAD68F02F03565E9FA39A5E627946699B2B07115889ED812D8BA639576A9` to the ERTP `USDC` brand.
  - **Where it is** (2026-10-27): of the 63,572.79 USDC gap, 62,829.67 is in the Fast USDC liquidity pool (`published.fastUsdc.poolMetrics`, `shareWorth` numerator) and 286.89 in the Inter Protocol PSM (`published.psm.IST.USDC.metrics`, `anchorPoolBalance`). About 456 USDC is untraced, probably in smart-wallet purses or other contracts.
  - **Not in a bank snapshot.** Fast USDC liquidity providers hold pool shares, not the voucher, and the PSM pool backs minted IST. A snapshot of bank balances only covers the voucher still in accounts (23,444.18), so read these contract balances at the same height. Both keep moving until the upgrade.
- **Neutron**: the scaffold has three messages. The first is a `MsgExecuteContract` from the gov module to the IBC rate-limit contract (`neutron15aqgplxcavqhurr0g5wwtdw6025dknkqwkfh0n46gp2qjl6236cs2yd3nl`), which removes the voucher's path on channel `any`. Its send quotas (35% of the supply per day, 70% per week) would otherwise reject the return packet, and the whole proposal would fail.
  - On 2026-10-26 the gov module held every role on the contract (`get_roles`), with no timelock and an empty message queue, so `remove_path` runs in the same execution.
  - Before submitting, check the quotas again: `neutrond query wasm contract-state smart <contract> '{"get_quotas":{"channel_id":"any","denom":"<voucher>"}}'`. If a path was added for `channel-30` too, add a second `remove_path` for it.
- **Timing:** Coreum and Archway allow up to a 14-day deposit period, and Terra2 and Archway have 7-day voting. Counting the upgrade first, submit early enough that the return is relayed well before 2027-01-12.

