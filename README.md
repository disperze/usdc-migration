# usdc-migration: `x/usdcmigration`

Cosmos SDK module and IBC middleware that let a chain wind down its **Noble USDC** voucher before Circle pauses USDC on Noble.

Circle announced on 2026-09-10 that it is ending USDC and CCTP V1 support on Noble:
- Minting stopped on 2026-10-13.
- CCTP V1 burn limits start shrinking on 2026-10-31.
- Full pause on **2027-01-12**.

After the pause, Noble rejects every incoming `uusdc` packet, so vouchers can no longer be redeemed. Many holders on IBC-connected chains are inactive and will never move their funds. This module lets **governance** release the USDC held in Noble's escrow for the chain to a Noble multisig in one step. The multisig then swaps or bridges it, for example to Injective USDC; that part is out of scope here. Holders are repaid off-chain from a snapshot.

## What it does

| Rule | Behaviour |
|---|---|
| **Receive block** | ICS20 packets of `base_denom` (`uusdc`) arriving on a configured Noble channel get an error acknowledgement, so Noble refunds the sender. Other Noble denoms pass. |
| **Send block** | Outgoing ICS20 transfers of the Noble USDC voucher (`ibc/HASH` of `transfer/<noble-channel>/uusdc`) are rejected on **every** channel, including IBC v2, unless the sender is the gov module account. This covers PFM forwards, ibc-hooks and wasm sends, and EVM precompile sends, because they all go through the transfer keeper. |
| **Mint-and-return** | `MsgReturnToNoble` (gov only) mints the voucher that gov does not already hold, then IBC-transfers it back to Noble. The voucher is burned on this chain, and Noble releases the matching USDC from the channel escrow to the receiver. |

User balances are **never touched**. After the return their vouchers are no longer backed. Governance freezes them with bank `MsgSetSendEnabled` in the same proposal, so the snapshot height and the freeze height match.

### Return cap and refunds

```
cap = bank supply(voucher) − gov balance(voucher) − returned_total(channel)
```

- While both blocks are on, Noble's escrow for the channel can only shrink through the gov return. So `cap` is exactly what Noble can still pay out, and a second return cannot over-draw.
- If Noble answers with an error ack, or the packet times out, the transfer module re-mints the voucher to gov. The middleware then decreases `returned_total`.
- A retry reuses the voucher gov already holds and mints nothing new.

## Release lines

| Branch | Go module | Cosmos SDK | ibc-go | Chains |
|---|---|---|---|---|
| `main` | `github.com/giansalex/usdc-migration/v10` | ≥ v0.53.4 | v10 (≥ v10.5.0) | Agoric, Xion, Coreum, Juno |
| `release/v8.x` | `github.com/giansalex/usdc-migration/v8` | ≥ v0.50.10 | v8 (≥ v8.7.0) | Terra2, Haqq, Impacthub (ixo), Archway |

Only the ibc-go-specific files differ between the two lines: the middleware, the ICS4 wrapper and the expected keepers. The v8 line has no IBC v2 wrapper.

## Messages, queries, events

**`MsgUpdateParams`** (gov only). Params:

| field | example (Juno) |
|---|---|
| `noble_channels` | `["channel-224"]` |
| `base_denom` | `"uusdc"` |
| `recv_block_enabled` | `true` |
| `send_block_enabled` | `true` |

**`MsgReturnToNoble`** (gov only). Fields:
- `channel_id`: must be in `noble_channels`.
- `receiver`: a `noble1…` address.
- `amount`: `0` means the whole cap.
- `timeout_seconds`: `0` means 3600; the maximum is 7 days.
- `memo`

**Queries** (CLI `query usdcmigration …`, REST `/usdcmigration/v1/…`):
- `params`
- `return-cap [channel-id]`: shows the voucher denom, supply, gov balance, returned total and cap.

**Events:**
- `usdcmigration_return_to_noble`: `height`, `channel_id`, `denom`, `amount`, `minted`, `receiver`, `sequence`. The `height` is the reference for the holder snapshot.
- `usdcmigration_return_refunded`: emitted on an error ack or timeout.
- `usdcmigration_params_updated`

## Governance runbook

1. **Upgrade.** Add the module (see [docs/integration.md](docs/integration.md)). The upgrade handler sets `noble_channels`.
   - From this height on, nobody can bring new USDC in from Noble or move the voucher out.
   - Local bank sends and DEX trades still work.
2. **Check the cap.** Run `query usdcmigration return-cap <channel>`.
   - `cap` should equal Noble's escrow balance for the counterparty channel: `GET <noble-rest>/cosmos/bank/v1beta1/balances/<escrow>`.
3. **Proposal**, one proposal executed atomically:

   ```json
   {
     "messages": [
       {
         "@type": "/usdcmigration.v1.MsgReturnToNoble",
         "authority": "<gov module address>",
         "channel_id": "channel-224",
         "receiver": "noble1<multisig>",
         "amount": "0",
         "timeout_seconds": "86400",
         "memo": ""
       },
       {
         "@type": "/cosmos.bank.v1beta1.MsgSetSendEnabled",
         "authority": "<gov module address>",
         "send_enabled": [{ "denom": "ibc/<voucher hash>", "enabled": false }],
         "use_default_for": []
       }
     ]
   }
   ```

4. **Relay and verify.**
   - Relay the packet. Noble's escrow for the channel should be `0`, and the multisig should hold the USDC.
   - On this chain, `returned_total` should equal the amount and `cap` should be `0`.
5. **Snapshot.** Take the holder snapshot of `ibc/<voucher hash>` at the `height` emitted by `usdcmigration_return_to_noble`, e.g. `query bank denom-owners … --height`.

**If Noble refunds** (error ack or timeout), gov holds the voucher again and the cap is restored. Because the voucher is frozen, the retry proposal must re-enable sending around the return:

```
[ MsgSetSendEnabled(true), MsgReturnToNoble, MsgSetSendEnabled(false) ]
```

**Do the return well before 2027-01-12.** After Circle's pause every attempt is refunded. The Noble multisig also needs time to move the USDC out while CCTP V1 burn limits shrink.

## Integration requirements (all chains)

See [docs/integration.md](docs/integration.md) for per-chain wiring. In short:

1. Add the store key and module, and grant `maccPerms[usdcmigration] = {Minter}`.
2. Build the keeper **before** the transfer keeper.
   - Pass the keeper (a pointer) as the transfer keeper's ICS4Wrapper.
   - Pass the transfer keeper's previous ICS4Wrapper as this keeper's next wrapper.
   - Call `SetTransferKeeper` once the transfer keeper is built.
3. Wrap the existing ICS20 stack, **outermost**, with `usdcmigration.NewIBCMiddleware`.
   - On v10 chains with an IBC v2 transfer route, also wrap it with `usdcmigration/v2.NewIBCMiddleware`.
4. **Gov must not be a blocked address.** ibc-go refuses to send from, or refund to, blocked addresses. `MsgReturnToNoble` fails with `ErrAuthorityBlocked` until gov is removed from the bank blocked list. Today gov is blocked on Terra2, Agoric, Haqq, ixo and Coreum.
5. In the upgrade handler, set the params with the chain's Noble channel(s).

## Development

The SDK v0.53.4 dependency graph (bytedance/sonic v1.14) does not build with Go ≥ 1.25, so use Go 1.24:

```sh
export GOTOOLCHAIN=go1.24.9
go test ./...
bash scripts/protocgen.sh   # needs buf, protoc-gen-gocosmos, protoc-gen-grpc-gateway
```

**Tests:**
- `x/usdcmigration/types`: rule tables, and voucher hashes checked against chain-registry.
- `x/usdcmigration/keeper`: authority, params and the blocked-gov guard.
- `x/usdcmigration` (end to end, `ibctesting` with three chains):
  - receive block and refund on Noble;
  - send block on v1 and v2;
  - mint-and-return draining Noble's escrow;
  - partial returns and the cap;
  - error-ack and timeout refunds, and a retry without re-minting.
- `testing/simapp`: the ibc-go v10.5.0 simapp with this module wired in.
