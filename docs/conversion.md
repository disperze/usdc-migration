# Converting the returned USDC to USDC (Injective)

After the return, the Noble multisig holds the chain's USDC on Noble. Before repayment it converts that USDC to **USDC (Injective)** and sends it to the chain. This page covers that step, with Neutron as the worked example. It has two paths:

- **[Path A: Osmosis](#path-a-osmosis-skipgo)** (recommended): one transaction on Noble, 1:1 swap through an Osmosis transmuter pool, delivered to Neutron over IBC. No CCTP, so Circle's 2026-12-01 CCTP cutoff does not apply. It still has to finish before 2027-01-12.
- **[Path B: CCTP V1 through Ethereum](#path-b-cctp-v1-through-ethereum)** (fallback): burn on Noble with CCTP V1 to Ethereum, then CCTP V2 from Ethereum to Injective, then IBC to Neutron. It needs signers on Ethereum and Injective, and must be done **before 2026-12-01**.

Values were checked on-chain on 2026-11-06. Check them again before converting.

## Before converting

1. **The return has settled.** Noble acknowledged the return packet, and the multisig's balance holds the returned amount:

   ```sh
   curl -s "<noble-rest>/cosmos/bank/v1beta1/balances/noble1<multisig>/by_denom?denom=uusdc"
   ```

2. **Keep some USDC for gas.** Noble charges transaction fees in USDC, so do not convert the last few USDC.
3. **Send a small test first** (for example 100 USDC) along the same path, all the way to the chain, before moving the rest.

## Timing (Circle)

From [Circle's announcement](https://www.circle.com/blog/circle-is-discontinuing-support-for-usdc-and-cctp-v1-on-noble) (2026-09-10) and the [CCTP V1 deprecation notice](https://www.circle.com/blog/cctp-version-updates):

| Date | What happens |
|---|---|
| 2026-10-13 | New USDC minting on Noble stopped. |
| 2026-10-31 | CCTP V1 burn limits started shrinking toward zero. |
| 2026-12-01 | CCTP V1 deprecation completes. CCTP exits from Noble may be limited to destinations that still support CCTP V1. |
| 2027-01-12 | USDC on Noble and its CCTP routes are fully paused. Circle snapshots remaining balances. |
| 2027-01-13 | Circle's manual redemption process opens for addresses in the snapshot. |

## Path A: Osmosis (Skip:Go)

### Route

| Hop | From → to | Channel | Denom in → out |
|---|---|---|---|
| 1 | Noble → Osmosis | Noble `channel-1` | `uusdc` → `ibc/498A0751C798A0D9A389AA3691123DADA57DAA4FE165D5C75894505B876BA6E4` (USDC.n on Osmosis) |
| 2 | Swap on Osmosis | pool `3497` | USDC.n → `ibc/794C7D7F3B857713878A3A1927251FA6AC1EEE520424C1F6FAFE9BA26D476138` (USDC.inj on Osmosis) |
| 3 | Osmosis → Injective | Osmosis `channel-122` | → `erc20:0xa00C59fF5a080D2b954d0c75e46E22a0c371235a` (USDC Injective) |
| 4 | Injective → Neutron | Injective `channel-177` | → `ibc/02BCFF6F786C7CBCDFE1AAFD8CFAC70D2AA9D57CF63AC0915048B666C650D8C5` (USDC Injective on Neutron) |

It is all **one `MsgTransfer` signed on Noble**, from Skip's API:
- Its memo calls Skip's entry-point contract on Osmosis (`osmo10a3k4hvk37cc4hnxctw4p95fhscd2z6h2rmx0aukc6rm8u9qqx9smfsh7u`), which swaps in pool 3497.
- The contract then sends the result to Injective, with a packet-forward memo that forwards it to Neutron.
- Nobody signs on Osmosis or Injective.

**Pool 3497** (`osmo147h5x9pcj7lm0cttlaefx6sqq5vdfnmwfcqxkmjd7exqm9gc7grqhr75m0`) is an Osmosis CosmWasm pool:
- It runs `crates.io:transmuter` 3.2.0 (a 1:1 swap, no fee), is active, and has no limiters.
- Skip quoted 2,058,476.26 USDC through it at 1:1 with 0% price impact.
- On 2026-11-06 it held 2,089,464.82 USDC.inj, 333,716.70 USDC.n and 270.18 of a third USDC.

**Liquidity is the main risk.** The pool's USDC.inj covers Neutron's escrow (about 2.03M) with only about 55k to spare, and other chains migrating from Noble use the same pool.
- Ask the pool admin (`osmo1rxjakgd8yhks2j7hc7pt6a22z3zd64grexpyf7`), Injective or Skip to add USDC.inj before the conversion.
- Check the balance right before each transfer:

  ```sh
  curl -s "<osmosis-rest>/cosmwasm/wasm/v1/contract/osmo147h5x9pcj7lm0cttlaefx6sqq5vdfnmwfcqxkmjd7exqm9gc7grqhr75m0/smart/$(printf '{"get_total_pool_liquidity":{}}' | base64 -w0)"
  ```

  If the pool cannot fill a transfer, the swap fails and the USDC.n lands at the recovery address on Osmosis. Nothing is lost, but the conversion has to wait for liquidity or use path B.

### Wallets

| Chain | Wallet | Signs? | Needs |
|---|---|---|---|
| Noble | **Noble multisig** (the return receiver) | yes, one tx per chunk | A plain multisig account of `secp256k1` keys (`keys add --multisig`), not a contract. USDC for fees. |
| Osmosis | the same multisig, `osmo` prefix | only if a hop fails | **Recovery address**: if the swap or the next hop fails, the funds land here, and the multisig signs on Osmosis to move them. OSMO for fees in that case. |
| Injective | an `inj` address of the signers | no | **Pass-through** for the packet forward to Neutron. If that forward fails after its retries, the funds go back to the Osmosis recovery address. |
| Neutron | **Neutron multisig** (same signers, `neutron` prefix) | no (receives) | Final receiver. It then creates and funds the redemption pool ([repayment.md](repayment.md#neutron)), with NTRN for fees. |

The Noble, Osmosis and Neutron addresses of one multisig are the same 20 bytes with a different bech32 prefix, because all three chains derive the address from the same `secp256k1` multisig public key. Injective derives addresses from `eth_secp256k1` keys, so the `inj` form of the multisig may not be usable there. On this path it only passes funds through. Still, confirm that a failed forward refunds to Osmosis (step 3 of [Steps](#steps)) before relying on it.

### Steps

1. **Quote** the route:

   ```sh
   curl -s -X POST https://api.skip.build/v2/fungible/route -H 'content-type: application/json' -d '{
     "amount_in": "<amount uusdc>",
     "source_asset_denom": "uusdc", "source_asset_chain_id": "noble-1",
     "dest_asset_denom": "ibc/02BCFF6F786C7CBCDFE1AAFD8CFAC70D2AA9D57CF63AC0915048B666C650D8C5", "dest_asset_chain_id": "neutron-1",
     "allow_multi_tx": true, "smart_relay": false
   }'
   ```

   Check `chain_ids` (`noble-1`, `osmosis-1`, `injective-1`, `neutron-1`), `txs_required: 1`, `amount_out` equal to `amount_in` and the swap on pool `3497`.
2. **Build the message** with `POST https://api.skip.build/v2/fungible/msgs`. Send the route's `operations` and an `address_list` in that chain order: Noble multisig, Osmosis multisig (recovery), Injective address, Neutron multisig. Use `slippage_tolerance_percent: "0.1"` and `timeout_seconds: "86400"`. The result is a single `/ibc.applications.transfer.v1.MsgTransfer` on `noble-1`.
3. **Read the memo before signing.** It must name pool `3497` and `min_asset` close to the amount. It must have `recover_address` set to the Osmosis multisig, and a `forward` to `channel-177` with `receiver` = the Neutron multisig.
4. **Sign and broadcast** as a multisig. Put the message in an unsigned tx, have each signer sign it, combine the signatures, and broadcast:

   ```sh
   nobled tx sign unsigned.json --multisig noble1<multisig> --from <signer> --chain-id noble-1 --output-document <signer>.json
   nobled tx multisign unsigned.json <multisig-key> <signer1>.json <signer2>.json ... > signed.json
   nobled tx broadcast signed.json
   ```

   The transfer and its memo carry timeouts, set by `timeout_seconds` in step 2. Skip's default is only 5 minutes, too short to collect multisig signatures, so ask for 24 hours. The pool swaps at a fixed 1:1, so a longer window does not change the minimum amount out. If the timeout passes before broadcast, quote and build again.
5. **Send in chunks**, for example four transfers of about 500k, quoting again each time. Then a failure affects one chunk, and the pool's liquidity is checked before each.
6. **Check** on Neutron that the multisig received the USDC (Injective). Then continue with the redemption pool.

## Path B: CCTP V1 through Ethereum

### Route

Noble's CCTP V1 module cannot burn to Injective directly. Its destinations are domains 0 to 10 (`remote_token_messengers`), and Injective is domain 29, a CCTP V2 chain. So the route goes through Ethereum (domain 0):

| Step | Chain | Action |
|---|---|---|
| 1 | Noble | `MsgDepositForBurn` (CCTP V1): burn `uusdc` to domain 0, `mint_recipient` = the Ethereum Safe. |
| 2 | Circle | V1 attestation: `GET https://iris-api.circle.com/attestations/<keccak256 of the message>`. The message is in the Noble tx's `MessageSent` event. |
| 3 | Ethereum | `receiveMessage(message, attestation)` on the V1 MessageTransmitter. Anyone can submit it; it mints USDC to the Safe. |
| 4 | Ethereum | From the Safe: `approve` the V2 TokenMessenger, then `depositForBurn(amount, 29, mintRecipient, USDC, 0x0, maxFee, 2000)`. Injective supports standard transfers only (no Fast Transfer), so use finality threshold `2000`. Take `maxFee` from Circle's fee API. |
| 5 | Circle | V2 attestation: `GET https://iris-api.circle.com/v2/messages/0?transactionHash=<tx>`. |
| 6 | Injective | `receiveMessage` on the V2 MessageTransmitter. Anyone can submit it; it mints `erc20:0xa00C59fF5a080D2b954d0c75e46E22a0c371235a` to `mintRecipient`. |
| 7 | Injective | IBC transfer of that denom to the Neutron multisig over `channel-177`, signed by the Injective account. |

Take the Ethereum and Injective contract addresses from [Circle's CCTP docs](https://developers.circle.com/cctp/cctp-supported-blockchains). On Noble, Ethereum's V1 TokenMessenger is `0xBd3fa81B58Ba92a82136038B25aDec7066af3155`.

### Limits

- **Per-message burn limit on Noble:** 500,000 USDC on 2026-11-06 (`GET <noble-rest>/circle/cctp/v1/per_message_burn_limits`). It keeps shrinking toward zero, so the full amount needs several `MsgDepositForBurn` messages. They can all go in one Noble tx, since the limit is per message.
- **Deadline:** CCTP V1 deprecation completes on 2026-12-01. Steps 1 to 3 must be finished before then. After that, the V1 mint on Ethereum may no longer be available. With the Neutron upgrade on 2026-11-23/24, this leaves about one week.
- Check `burning_and_minting_paused` on Noble, and the V2 per-message limit on Ethereum, before starting.

### Wallets

| Chain | Wallet | Signs? | Needs |
|---|---|---|---|
| Noble | **Noble multisig** | yes | Signs the `MsgDepositForBurn` messages. USDC for fees. |
| Ethereum | **Safe** with the same signers (their Ethereum keys) | yes | `mint_recipient` of step 1. It signs `approve` and `depositForBurn`. ETH for gas, including `receiveMessage` if nobody else submits it. |
| Injective | **Injective account** of the signers | yes | `mintRecipient` of step 4, so it must be able to hold the ERC-20 USDC and sign the IBC transfer to Neutron. INJ for gas. Confirm which multisig scheme Injective supports for this. Otherwise the last hop depends on a single key, and the Neutron team prefers not to sign on Injective at all. |
| Neutron | **Neutron multisig** | no (receives) | Final receiver, as in path A. |

## If neither path is available

The USDC stays in the Noble multisig. Circle's manual redemption, from 2027-01-13, covers addresses in the pause-day snapshot that are holder-controlled and pass Circle's compliance checks. A plain multisig account on Noble qualifies as holder-controlled. This route is slower and its payout terms are set by Circle, so treat it as a last resort, not a plan.

## Comparison

| | Path A: Osmosis | Path B: CCTP V1 via Ethereum |
|---|---|---|
| Signatures | Noble only (Osmosis only on failure) | Noble, Ethereum, Injective |
| Deadline | 2027-01-12 (Noble pause) | 2026-12-01 (CCTP V1) |
| Limit | Pool 3497's USDC.inj liquidity | Noble per-message burn limit (shrinking) |
| Rate | 1:1, no swap fee | 1:1, plus Ethereum gas and CCTP fees |
| Time | Minutes per chunk, once signed | Two attestations, plus Ethereum finality for the V2 burn (about 15–20 min) |
| Lands on Neutron | Directly | Only after the Injective account sends it |
