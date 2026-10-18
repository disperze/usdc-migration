# Return proposals

One scaffold per chain in [integration.md](../integration.md). Each one is step 3 of the [governance runbook](../../README.md#governance-runbook): `MsgReturnToNoble` followed by `MsgSetSendEnabled`, executed atomically.

Submit them only **after** the chain's software upgrade has added `x/usdcmigration` and set `noble_channels`. Before that, the chain does not know `/usdcmigration.v1.MsgReturnToNoble` and rejects the proposal.

## Minimum deposit

Queried on 2026-10-18 from each chain's `cosmos/gov/v1/params/deposit`. A proposal enters the voting period as soon as its total deposit reaches `min_deposit`. Each scaffold's `deposit` field is set to the full `min_deposit`, so the proposal goes straight to voting.

| Chain | Chain ID | `min_deposit` (to reach voting) | Initial deposit at submit | Deposit period | Voting period | Quorum |
|---|---|---|---|---|---|---|
| Juno | `juno-1` | **5,000 JUNO** (`5000000000ujuno`) | 20% = 1,000 JUNO | 10 days | 5 days | 33.4% |
| Agoric | `agoric-3` | **5,000 BLD** (`5000000000ubld`) | none | 2 days | 3 days | 33.4% |
| Xion | `xion-mainnet-1` | **1,000 XION** (`1000000000uxion`) | none | 2 days | 4 days | 33.4% |
| Coreum | `coreum-mainnet-1` | **50,000 CORE** (`50000000000ucore`) | 50% = 25,000 CORE | 14 days | 5 days | 40% |
| Terra2 | `phoenix-1` | **5,000 LUNA** (`5000000000uluna`) | none | 7 days | 7 days | 30% |
| Haqq | `haqq_11235-1` | **6,000 ISLM** (`6000000000000000000000aISLM`) | none | 2 days | 3 days | 33.4% |
| Impacthub (ixo) | `ixo-5` | **10,000 IXO** (`10000000000uixo`) | 10% = 1,000 IXO | 2 days | 5 days | 33.4% |
| Archway | `archway-1` | **5,000 ARCH** (`5000000000000000000000aarch`) | none | 14 days | 7 days | 33.4% |

"Initial deposit at submit" is `min_initial_deposit_ratio × min_deposit`: the smallest deposit the chain accepts when the proposal is submitted. Every chain's threshold is 50%. The deposit is refunded unless the proposal is vetoed (or doesn't reach quorum, on chains that burn deposits for that).

Expedited proposals (1-day voting) are not used in the scaffolds. On Agoric and Archway `expedited_min_deposit` is set in `stake`, a denom that doesn't exist on those chains, so expedited proposals can't be submitted there at all.

## Values already filled in (checked on-chain 2026-10-18)

| Chain | Gov module address (`authority`) | Noble channel | Voucher supply = USDC to return |
|---|---|---|---|
| Juno | `juno10d07y265gmmuvt4z0w9aw880jnsr700jvss730` | `channel-224` | 13,581.84 |
| Agoric | `agoric10d07y265gmmuvt4z0w9aw880jnsr700jgl36x9` | `channel-62` | 23,444.18 |
| Xion | `xion10d07y265gmmuvt4z0w9aw880jnsr700jctf8qc` | `channel-2` | 65,690.84 |
| Coreum | `core10d07y265gmmuvt4z0w9aw880jnsr700jfvt73g` | `channel-19` | 55,102.61 |
| Terra2 | `terra10d07y265gmmuvt4z0w9aw880jnsr700juxf95n` | `channel-253` | 386,565.78 |
| Haqq | `haqq10d07y265gmmuvt4z0w9aw880jnsr700jc9xkg9` | `channel-4` | 68,769.51 |
| Impacthub (ixo) | `ixo10d07y265gmmuvt4z0w9aw880jnsr700j9hdhjq` | `channel-26` | 66,809.26 |
| Archway | `archway10d07y265gmmuvt4z0w9aw880jnsr700j0f0puy` | `channel-29` | 33,628.67 |
| **Total** | | | **713,592.70** |

Every voucher denom in the scaffolds matches `query ibc-transfer denom-hash transfer/<channel>/uusdc` on that chain. That includes the three that are computed rather than taken from the chain registry (Coreum, Haqq, ixo).

## Before submitting

Replace the placeholders in each file:

- `noble1REPLACE_WITH_MULTISIG`: the Noble multisig. It appears in `receiver` and in `summary`.
- `ipfs://REPLACE_WITH_METADATA_CID`: the proposal metadata (255 characters max), or `""`.
- `REPLACE_WITH_REPAYMENT_PLAN_URL` in `summary`: where holders can read how they are repaid from the snapshot.

Then:

1. Check the cap: `<binary> query usdcmigration return-cap <channel>`. It should equal Noble's escrow balance for the counterparty channel.
2. Submit:

   ```sh
   <binary> tx gov submit-proposal docs/proposals/<chain>.json \
     --from <key> --chain-id <chain-id> --gas auto --gas-adjustment 1.5 --fees <fees>
   ```

3. After it passes, relay the return packet and verify it (runbook step 4). Then take the snapshot at the `height` from the `usdcmigration_return_to_noble` event.

Notes:

- **Keep the message order.** The return comes before the freeze. On `main` the return packet involves no bank send, so the order doesn't matter there. On the v8 line, the transfer keeper checks `SendEnabled` until the return packet is ported, so a freeze placed first would make the return fail.
- **`timeout_seconds` is `86400`** (1 day), which gives relayers time. The maximum is 7 days. If the packet times out, the cap is restored, and the retry is a new proposal with only `MsgReturnToNoble`. The voucher stays frozen.
- **Leave `memo` empty** (see [integration.md](../integration.md)).
- **Coreum** mainnet runs `v7.0.0` today, but integration.md analysed tx-chain v8. Re-check the wiring against the release the upgrade starts from.
- **Agoric**: coordinate the date with the Fast USDC operators. The freeze stops Fast USDC flows that use this voucher.
- **Timing:** Coreum and Archway allow up to a 14-day deposit period, and Terra2 and Archway have 7-day voting. Counting the upgrade first, submit early enough that the return is relayed well before 2027-01-12.
