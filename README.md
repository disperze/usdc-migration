# `x/usdcmigration`

[![test](https://github.com/disperze/usdc-migration/actions/workflows/test.yml/badge.svg)](https://github.com/disperze/usdc-migration/actions/workflows/test.yml)

Cosmos SDK module and IBC middleware that let a chain wind down its **Noble USDC** voucher before Circle pauses USDC on Noble.

Circle announced on 2026-09-10 that it is ending USDC and CCTP V1 support on Noble:
- Minting stopped on 2026-10-13.
- CCTP V1 burn limits start shrinking on 2026-10-31.
- Full pause on **2027-01-12**.

After the pause, Noble rejects every incoming `uusdc` packet, so vouchers can no longer be redeemed. Many holders on IBC-connected chains are inactive and will never move their funds. This module lets **governance** release the USDC held in Noble's escrow for the chain to a Noble multisig in one step. The multisig then swaps or bridges it, for example to Injective USDC; that part is out of scope here. Holders receive the tokens via an on-chain distribution or by claiming it through a contract.

## What it does

| Rule | Behaviour |
|---|---|
| **Receive block** | ICS20 packets of `base_denom` (`uusdc`) arriving on a configured Noble channel get an error acknowledgement, so Noble refunds the sender. Other Noble denoms pass. |
| **Send block** | Outgoing ICS20 transfers of the Noble USDC voucher (`ibc/HASH` of `transfer/<noble-channel>/uusdc`) are rejected on **every** channel, including IBC v2, for **every** sender, gov included. This covers PFM forwards, ibc-hooks and wasm sends, and EVM precompile sends, because they all go through the transfer keeper. |
| **Withdraw funds** | `MsgReturnToNoble` (gov only) builds the ICS20 packet that returns the voucher to Noble and sends it straight to the next ICS4 wrapper, without going through the transfer keeper. Nothing is minted, burned or moved on this chain. Noble sees an ordinary voucher return and releases the matching USDC from the channel escrow to the receiver. |

User balances are **never touched**. After the return their vouchers are no longer backed. Governance freezes them with bank `MsgSetSendEnabled` in the same proposal, so the snapshot height and the freeze height match.

The return packet follows the approach used, and tested, for the Evmos shutdown ([`forceIBCTransfer`](https://github.com/disperze/evmos/blob/release/v20/x/ibc/transfer/keeper/sudo_ibc.go)). Because no funds pass through the gov account, the module needs no `Minter` permission, gov does not have to be removed from the bank blocked addresses, and the bank freeze does not get in the way of a return.

### Return cap and refunds

```
cap = bank supply(voucher) − returned_total(channel)
```

- While both blocks are on, the voucher supply and Noble's escrow for the channel no longer change, except through the gov return, which does not burn the voucher. Normally they are equal, so `cap` is what Noble can still pay out.
- The escrow can be above the supply (for example USDC sent straight to the escrow address) or slightly below it. An explicit `amount` is therefore not limited by `cap`: set it to Noble's escrow balance to release all of it. Noble enforces that balance; a larger amount gets an error ack and is handled like any refund below.
- A return packet is any voucher packet sent by gov on a Noble channel. Gov can no longer send the voucher any other way, so this match is exact.
- If Noble answers with an error ack, or the packet times out, the middleware does **not** pass the packet on to the transfer module, which would mint the voucher back to gov. Instead it only decreases `returned_total`.
- A retry is just another `MsgReturnToNoble`.

## Release lines

| Branch | Go module | Cosmos SDK | ibc-go | Chains |
|---|---|---|---|---|
| `master` | `github.com/disperze/usdc-migration/v10` | ≥ v0.53.4 | v10 (≥ v10.5.0) | Agoric, Xion, Coreum, Neutron, Juno |
| `release/v8.x` | `github.com/disperze/usdc-migration/v8` | ≥ v0.50.10 | v8 (≥ v8.7.0) | Terra2, Haqq, Impacthub (ixo), Archway |

Only the ibc-go-specific files differ between the two lines: the middleware, the ICS4 wrapper and the expected keepers. The v8 line has no IBC v2 wrapper.

Releases are tagged `v10.x.y` on `master` and `v8.x.y` on `release/v8.x`. See [CHANGELOG.md](CHANGELOG.md).

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
- `amount`: `0` means the whole cap. A non-zero amount may exceed the cap, up to Noble's escrow for the channel.
- `timeout_seconds`: `0` means 3600; the maximum is 7 days.
- `memo`

**Queries** (CLI `query usdcmigration …`, REST `/usdcmigration/v1/…`):
- `params`
- `return-cap [channel-id]`: shows the voucher denom, supply, returned total and cap.

**Events:**
- `usdcmigration_return_to_noble`: `height`, `channel_id`, `denom`, `amount`, `receiver`, `sequence`. The `height` is the reference for the holder snapshot.
- `usdcmigration_return_refunded`: emitted on an error ack or timeout.
- `usdcmigration_params_updated`

## Governance runbook

1. **Upgrade.** Add the module (see [docs/integration.md](docs/integration.md)). The upgrade handler sets `noble_channels`.
   - From this height on, nobody can bring new USDC in from Noble or move the voucher out.
   - Local bank sends and DEX trades still work.
2. **Check the cap.** Run `query usdcmigration return-cap <channel>`.
   - Compare it with Noble's escrow balance for the counterparty channel: `GET <noble-rest>/cosmos/bank/v1beta1/balances/<escrow>`. If they differ, use the escrow balance as `amount` instead of `0`.
3. **Proposal**, one proposal executed atomically. Per-chain scaffolds and minimum deposits are in [docs/proposals](docs/proposals/README.md).

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

**If Noble refunds** (error ack or timeout), the cap is restored and an `usdcmigration_return_refunded` event is emitted. Nothing is minted to gov. The retry proposal is just another `MsgReturnToNoble`. The voucher can stay frozen, because the return packet involves no bank send.

**Do the return well before 2027-01-12.** After Circle's pause every attempt is refunded. The Noble multisig also needs time to move the USDC out while CCTP V1 burn limits shrink.

## Integration requirements (all chains)

See [docs/integration.md](docs/integration.md) for per-chain wiring. In short:

1. Add the store key and module. The module needs no module account permissions.
2. Build the keeper **before** the transfer keeper.
   - Pass the keeper (a pointer) as the transfer keeper's ICS4Wrapper.
   - Pass the transfer keeper's previous ICS4Wrapper as this keeper's next wrapper. `MsgReturnToNoble` sends through that wrapper too, so it must be the chain's real send path (PFM, callbacks, or the channel keeper).
3. Wrap the existing ICS20 stack, **outermost**, with `usdcmigration.NewIBCMiddleware`. This middleware also catches the acks and timeouts of return packets.
   - On v10 chains with an IBC v2 transfer route, also wrap it with `usdcmigration/v2.NewIBCMiddleware`.
4. In the upgrade handler, set the params with the chain's Noble channel(s).

Gov may stay a blocked address: the return never sends from, or refunds to, the gov account.

## Development

The SDK v0.53.4 dependency graph (bytedance/sonic v1.14) does not build with Go ≥ 1.25, so use Go 1.24:

```sh
export GOTOOLCHAIN=go1.24.9
go test ./...
bash scripts/protocgen.sh   # needs buf, protoc-gen-gocosmos, protoc-gen-grpc-gateway
```

**Tests:**
- `x/usdcmigration/types`: rule tables, and voucher hashes checked against chain-registry.
- `x/usdcmigration/keeper`: authority, params, the bytes of the return packet, and the send block applying to gov.
- `x/usdcmigration` (end to end, `ibctesting` with three chains):
  - receive block and refund on Noble;
  - send block on v1 and v2, gov included;
  - a return packet draining Noble's escrow, with the voucher supply and all balances unchanged;
  - a return while the voucher is frozen with `SetSendEnabled(false)`;
  - partial returns and the cap;
  - error-ack and timeout refunds with nothing minted, then a retry.
- `testing/simapp`: the ibc-go v10.5.0 simapp with this module wired in.

## License

Apache License 2.0. See [LICENSE](LICENSE).
