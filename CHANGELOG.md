# Changelog

All notable changes to this project are documented here. The format follows
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/), and versions follow
[Semantic Versioning](https://semver.org/). The major version tracks ibc-go:
`v10.x.y` is released from `master` (ibc-go v10), `v8.x.y` from `release/v8.x`
(ibc-go v8).

## [Unreleased]

First release, to be tagged `v10.0.0`.

### Added

- `x/usdcmigration` module and ICS20 IBC middleware for winding down the Noble
  USDC voucher. Requires Cosmos SDK ≥ v0.53.4 and ibc-go ≥ v10.5.0.
- Receive block: `base_denom` packets from a configured Noble channel get an
  error acknowledgement, so Noble refunds the sender.
- Send block: outgoing transfers of the Noble USDC voucher are rejected on every
  channel and for every sender, on IBC v1 and IBC v2 (`x/usdcmigration/v2`).
- `MsgReturnToNoble` (gov only): sends the return packet straight to the next
  ICS4 wrapper so Noble releases the channel escrow to a receiver, with a return
  cap (`supply − returned_total`) and refund handling on error ack or timeout.
- `MsgUpdateParams` (gov only), queries `params` and `return-cap`, and the
  `usdcmigration_return_to_noble`, `usdcmigration_return_refunded` and
  `usdcmigration_params_updated` events.
- Apache-2.0 `LICENSE`.

## Releasing

1. Move the `[Unreleased]` entries under a new `## [vX.Y.Z] - YYYY-MM-DD`
   heading and merge to `master` (or `release/v8.x` for `v8.x.y`).
2. Tag the merged commit and push the tag:

   ```sh
   git checkout master && git pull
   git tag -a v10.0.0 -m "v10.0.0"
   git push origin v10.0.0
   ```
