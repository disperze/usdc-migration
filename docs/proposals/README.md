# Return proposals

One scaffold per chain in [integration.md](../integration.md), except Juno, which is no longer included. Each one is step 3 of the [governance runbook](../../README.md#governance-runbook): `MsgReturnToNoble` followed by `MsgSetSendEnabled`, executed atomically.

Before any of this, each chain votes on a text proposal approving the integration: see [signaling/](signaling/README.md).

Submit the return proposals only **after** the chain's software upgrade has added `x/usdcmigration` and set `noble_channels`. Before that, the chain does not know `/usdcmigration.v1.MsgReturnToNoble` and rejects the proposal.

## Minimum deposit

Queried on 2026-10-18 from each chain's `cosmos/gov/v1/params/deposit`. A proposal enters the voting period as soon as its total deposit reaches `min_deposit`. Each scaffold's `deposit` field is set to the full `min_deposit`, so the proposal goes straight to voting.

| Chain | Chain ID | `min_deposit` (to reach voting) | Initial deposit at submit | Deposit period | Voting period | Quorum |
|---|---|---|---|---|---|---|
| Agoric | `agoric-3` | **5,000 BLD** (`5000000000ubld`) | none | 2 days | 3 days | 33.4% |
| Xion | `xion-mainnet-1` | **1,000 XION** (`1000000000uxion`) | none | 2 days | 4 days | 33.4% |
| Coreum | `coreum-mainnet-1` | **50,000 CORE** (`50000000000ucore`) | 50% = 25,000 CORE | 14 days | 5 days | 40% |
| Terra2 | `phoenix-1` | **5,000 LUNA** (`5000000000uluna`) | none | 7 days | 7 days | 30% |
| Haqq | `haqq_11235-1` | **6,000 ISLM** (`6000000000000000000000aISLM`) | none | 2 days | 3 days | 33.4% |
| Impacthub (ixo) | `ixo-5` | **10,000 IXO** (`10000000000uixo`) | 10% = 1,000 IXO | 2 days | 5 days | 33.4% |
| Archway | `archway-1` | **5,000 ARCH** (`5000000000000000000000aarch`) | none | 14 days | 7 days | 33.4% |

"Initial deposit at submit" is `min_initial_deposit_ratio × min_deposit`: the smallest deposit the chain accepts when the proposal is submitted. Every chain's threshold is 50%. The deposit is refunded unless the proposal is vetoed (or doesn't reach quorum, on chains that burn deposits for that).

Expedited proposals (1-day voting) are not used in the scaffolds. On Agoric and Archway `expedited_min_deposit` is set in `stake`, a denom that doesn't exist on those chains, so expedited proposals can't be submitted there at all.

## Values already filled in (checked on-chain 2026-10-26)

| Chain | Gov module address (`authority`) | Source channel | Noble channel | Noble escrow address | Noble escrow balance (USDC) |
|---|---|---|---|---|---|
| Agoric | `agoric10d07y265gmmuvt4z0w9aw880jnsr700jgl36x9` | `channel-62` | `channel-21` | `noble16knm868dfqlvxhpdsv5w6n2an6rgsn3shzldlm` | 87,016.97 |
| Xion | `xion10d07y265gmmuvt4z0w9aw880jnsr700jctf8qc` | `channel-2` | `channel-113` | `noble10zfpsus4h27eqgrjar7yj2mwer3q84zkwmsgml` | 67,319.81 |
| Coreum | `core10d07y265gmmuvt4z0w9aw880jnsr700jfvt73g` | `channel-19` | `channel-49` | `noble1u929fykjps0qny9enq0rgdsavau7amszl7dj9y` | 55,000.34 |
| Terra2 | `terra10d07y265gmmuvt4z0w9aw880jnsr700juxf95n` | `channel-253` | `channel-30` | `noble12vssh8spx53qzg532xpj22sa7e85ts4x76e5rm` | 311,071.38 |
| Haqq | `haqq10d07y265gmmuvt4z0w9aw880jnsr700jc9xkg9` | `channel-4` | `channel-32` | `noble1y376tnlmyu23jaxdjj3xdpsf2crxfgpaly3ynx` | 68,769.51 |
| Impacthub (ixo) | `ixo10d07y265gmmuvt4z0w9aw880jnsr700j9hdhjq` | `channel-26` | `channel-15` | `noble186mpcj53q3hgfnzdu9wkteczs236vdclxm264k` | 54,302.02 |
| Archway | `archway10d07y265gmmuvt4z0w9aw880jnsr700j0f0puy` | `channel-29` | `channel-12` | `noble1004rr9e45qpm2ylvxymphm8sryvxq3a2kcyjgw` | 33,621.45 |
| **Total** | | | | | **677,101.47** |

The source channel is the chain's end of its transfer channel to Noble, and the value set in `noble_channels`. The Noble channel is the counterparty, taken from the [chain registry `_IBC` files](https://github.com/cosmos/chain-registry/tree/master/_IBC). The escrow address is `sha256("ics20-1" + 0x00 + "transfer/<noble channel>")[:20]` in bech32 `noble`, and matches Noble's `ibc/apps/transfer/v1/channels/<noble channel>/ports/transfer/escrow_address`. Balances are its `uusdc` balance at Noble height 62,477,983.

The return cap (`amount: "0"`) is the voucher supply on the source chain, not the escrow balance. They are equal on Xion, Coreum, Haqq and ixo, so those scaffolds use `"0"`. Elsewhere the scaffold sets `amount` to the escrow balance in `uusdc`, which the module accepts even above the cap:

| Chain | Voucher supply | Noble escrow | `amount` |
|---|---|---|---|
| Agoric | 23,444.18 | 87,016.97 | `87016970394` |
| Terra2 | 310,071.86 | 311,071.38 | `311071376277` |
| Archway | 33,623.14 | 33,621.45 | `33621448005` |

On Agoric and Terra2 this releases the escrow above the supply too. On Archway the supply is above the escrow, so `"0"` would ask Noble for more than it holds. Noble refunds any amount above the escrow balance (error ack). The escrow only stops moving at the upgrade height, so set these amounts again from Noble after the upgrade, once pending packets are relayed (step 1 of [Before submitting](#before-submitting)).

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
2. Check the cap: `<binary> query usdcmigration return-cap <channel>`, and Noble's escrow balance for the counterparty channel. If they differ, set `amount` to the escrow balance (Agoric, Terra2 and Archway already do); if they are equal, `"0"` returns it all.
3. Submit:

   ```sh
   <binary> tx gov submit-proposal docs/proposals/<chain>.json \
     --from <key> --chain-id <chain-id> --gas auto --gas-adjustment 1.5 --fees <fees>
   ```

4. After it passes, relay the return packet and verify it (runbook step 4). Then take the snapshot at the `height` from the `usdcmigration_return_to_noble` event.

Notes:

- **Keep the message order.** The return comes before the freeze. On `main` the return packet involves no bank send, so the order doesn't matter there. On the v8 line, the transfer keeper checks `SendEnabled` until the return packet is ported, so a freeze placed first would make the return fail.
- **`timeout_seconds` is `86400`** (1 day), which gives relayers time. The maximum is 7 days. If the packet times out, the cap is restored, and the retry is a new proposal with only `MsgReturnToNoble`. The voucher stays frozen.
- **Leave `memo` empty** (see [integration.md](../integration.md)).
- **Coreum** mainnet runs `v7.0.0` today, but integration.md analysed tx-chain v8. Re-check the wiring against the release the upgrade starts from.
- **Agoric**: coordinate the date with the Fast USDC operators. The freeze stops Fast USDC flows that use this voucher.
- **Timing:** Coreum and Archway allow up to a 14-day deposit period, and Terra2 and Archway have 7-day voting. Counting the upgrade first, submit early enough that the return is relayed well before 2027-01-12.
