# Changelog

All notable changes to this project are documented here. The format follows
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/), and versions follow
[Semantic Versioning](https://semver.org/). The major version tracks ibc-go:
`v10.x.y` is released from `master` (ibc-go v10), `v8.x.y` from `release/v8.x`
(ibc-go v8).

## [Unreleased]

First release of the ibc-go v8 line, to be tagged `v8.0.0`.

### Added

- `x/usdcmigration` module and ICS20 IBC middleware for winding down the Noble
  USDC voucher. Requires Cosmos SDK ≥ v0.50.10 and ibc-go ≥ v8.7.0. `go.mod`
  requires only the versions those two need (e.g. `cosmossdk.io/api` v0.7.5,
  `x/tx` v0.13.5, `collections` v0.4.0, gRPC v1.64.1), so adding the module
  does not bump a v0.50 chain's dependencies.
- Receive block: `base_denom` packets from a configured Noble channel get an
  error acknowledgement, so Noble refunds the sender.
- Send block: outgoing transfers of the Noble USDC voucher are rejected on every
  channel and for every sender. The v8 line has no IBC v2 wrapper.
- `MsgReturnToNoble` (gov only): sends the return packet straight to the next
  ICS4 wrapper so Noble releases the given amount of the channel escrow to a
  receiver, with refund handling on error ack or timeout. The module keeps no
  return state; Noble's escrow balance is the only limit. On v8 the packet is
  sent with the transfer module's channel capability, so `NewKeeper` also takes
  its scoped keeper, and error acks wrapped in an ICS29 incentivized ack are
  recognised too.
- `MsgUpdateParams` (gov only), query `params`, and the
  `usdcmigration_return_to_noble`, `usdcmigration_return_refunded` and
  `usdcmigration_params_updated` events.
- Apache-2.0 `LICENSE`.

## Releasing

1. Move the `[Unreleased]` entries under a new `## [vX.Y.Z] - YYYY-MM-DD`
   heading and merge to `master` (or `release/v8.x` for `v8.x.y`).
2. Tag the merged commit and push the tag:

   ```sh
   git checkout release/v8.x && git pull
   git tag -a v8.0.0 -m "v8.0.0"
   git push origin v8.0.0
   ```
