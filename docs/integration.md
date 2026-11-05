# Integrating `x/usdcmigration`

Line references point to the release analysed for each chain. Re-check them against the release you upgrade from.

| Chain | Line | Import | Noble channel | Voucher denom |
|---|---|---|---|---|
| Juno | v10 | `…/usdc-migration/v10` | `channel-224` | `ibc/4A482FA914A4B9B05801ED81C33713899F322B24F76A06F4B8FE872485EA22FF` |
| Agoric | v10 | `…/usdc-migration/v10` | `channel-62` | `ibc/FE98AAD68F02F03565E9FA39A5E627946699B2B07115889ED812D8BA639576A9` |
| Xion | v10 | `…/usdc-migration/v10` | `channel-2` | `ibc/F082B65C88E4B6D5EF1DB243CDA1D331D002759E938A0F5CD3FFDC5D53B3E349` |
| Coreum | v10 | `…/usdc-migration/v10` | `channel-19` | `ibc/E1E3674A0E4E1EF9C69646F9AF8D9497173821826074622D831BAB73CCB99A2D` * |
| Neutron | v10 | `…/usdc-migration/v10` | `channel-30` | `ibc/B559A80D62249C8AA07A380E2A2BEA6E5CA9A6F079C912C3A9E9B494105E4F81` |
| Terra2 | v8 | `…/usdc-migration/v8` | `channel-253` | `ibc/2C962DAB9F57FE0921435426AE75196009FAA1981BF86991203C8411F8980FDB` |
| Haqq | v8 | `…/usdc-migration/v8` | `channel-4` | `ibc/BFF0D3805B50D93E2FA5C0B2DDF7E0B30A631076CD80BC12A48C0E95404B4A41` * |
| Impacthub (ixo) | v8 | `…/usdc-migration/v8` | `channel-26` | `ibc/6BBE9BD4246F8E04948D5A4EEE7164B2630263B9EBB5E7DC5F0A46C62A2FF97B` * |
| Archway | v8 | `…/usdc-migration/v8` | `channel-29` | `ibc/43897B9739BD63E3A08A88191999C632E052724AB96BD4C74AE31375C991F48D` |

\* Computed as `sha256("transfer/<channel>/uusdc")`, because the denom is missing from that chain's chain-registry assetlist. Confirm it on-chain before the upgrade: `query ibc-transfer denom-hash transfer/<channel>/uusdc`.

## Generic wiring

```go
import (
	"github.com/disperze/usdc-migration/v10/x/usdcmigration"
	usdcmigrationkeeper "github.com/disperze/usdc-migration/v10/x/usdcmigration/keeper"
	usdcmigrationtypes "github.com/disperze/usdc-migration/v10/x/usdcmigration/types"
	usdcmigrationv2 "github.com/disperze/usdc-migration/v10/x/usdcmigration/v2" // v10 + IBC v2 only
)

// app struct
UsdcMigrationKeeper *usdcmigrationkeeper.Keeper

// store keys
usdcmigrationtypes.StoreKey,

// 1. BEFORE the transfer keeper: give it the ICS4 wrapper the transfer keeper used until now.
//    MsgReturnToNoble sends its packet through this wrapper as well.
app.UsdcMigrationKeeper = usdcmigrationkeeper.NewKeeper(
	appCodec, runtime.NewKVStoreService(keys[usdcmigrationtypes.StoreKey]),
	<previous transfer ICS4Wrapper>, // e.g. app.PacketForwardKeeper or app.IBCKeeper.ChannelKeeper
	scopedTransferKeeper,            // v8 only: the transfer module's scoped keeper
	authtypes.NewModuleAddress(govtypes.ModuleName).String(),
)

// 2. The transfer keeper sends through it. Every later copy of the transfer
//    keeper (PFM, wasm, ...) then shares this pointer.
app.TransferKeeper = ibctransferkeeper.NewKeeper(..., app.UsdcMigrationKeeper /* ics4Wrapper */, ...)

// 3. Outermost on the ICS20 stack (it also handles acks/timeouts of return packets):
transferStack = usdcmigration.NewIBCMiddleware(app.UsdcMigrationKeeper, transferStack)

// 4. (v10, only if an IBC v2 transfer route exists) outermost on the v2 route:
ibcv2TransferStack = usdcmigrationv2.NewIBCMiddleware(app.UsdcMigrationKeeper, ibcv2TransferStack)

// 5. Module manager + InitGenesis/ExportGenesis order (after transfer):
usdcmigration.NewAppModule(app.UsdcMigrationKeeper),
```

**Rules that matter:**

- **Never call `TransferKeeper.WithICS4Wrapper(x)` after this.** Use `app.UsdcMigrationKeeper.WithICS4Wrapper(x)` instead.
  - The transfer keeper is a value type. A copy taken before a late `WithICS4Wrapper`, such as PFM's from its `SetTransferKeeper`, would keep its old wrapper and skip the send block.
  - With our keeper fixed in the constructor, every copy points at the same `*Keeper`. Re-pointing its inner wrapper later then reaches all of them.
  - `MsgReturnToNoble` sends through that inner wrapper, so a late `WithICS4Wrapper(cbStack)` must be kept. Otherwise returns would skip `cbStack`.
- **No module account permissions and no blocked-address changes.** The return builds the ICS20 packet itself, so nothing is minted, burned or sent from the gov account. Gov may stay in the bank keeper's blocked addresses.
- **v8: pass the transfer module's scoped keeper** (`app.CapabilityKeeper.ScopeToModule(ibctransfertypes.ModuleName)`, often `app.ScopedTransferKeeper`). The return packet is sent with the transfer module's channel capability. Without it, `MsgReturnToNoble` fails with `ErrChannelCapabilityNotFound`.
- **v8, ICS29 fees:** when ibc-fee sits under this middleware (Archway, ixo), the error ack of a return packet is read through the incentivized ack and never reaches ibc-fee. If someone paid relayer fees for that packet, they stay in fee escrow. Noble channels are normally not fee-enabled.
- **Do not change params while a return is in flight.** <a id="do-not-change-params-while-a-return-is-in-flight"></a>
  - After `MsgReturnToNoble` executes, do not submit `MsgUpdateParams` until the return packet has settled: Noble acknowledged it, or it was refunded (`usdcmigration_return_refunded` event after an error ack or timeout).
  - This applies in particular to `noble_channels`, `base_denom` and `send_block_enabled`. Acks and timeouts are recognised as return packets by comparing them with the **current** params (sender is gov, channel is a Noble channel, denom is the voucher). If a param changed in between, an error ack or timeout would no longer be recognised and would reach the transfer module, which refunds gov by minting the voucher back, unbacked.
  - Keep `send_block_enabled = true` for the whole migration. It is what guarantees that gov cannot send the voucher any other way, so every gov voucher packet on a Noble channel is a return.
  - In practice a return settles within minutes and a gov proposal takes a full voting period, so this only needs care if a param-change proposal is already in voting when the return executes.
- **Leave the return memo empty.** The packet still goes through the ICS4 wrappers below this keeper, so a callbacks key in the memo would trigger a callbacks middleware there.

**Upgrade handler:**

```go
storeUpgrades := storetypes.StoreUpgrades{Added: []string{usdcmigrationtypes.StoreKey}}

// in the handler, after RunMigrations:
params := usdcmigrationtypes.DefaultParams()
params.NobleChannels = []string{"channel-224"} // this chain's Noble channel
if err := app.UsdcMigrationKeeper.SetParams(ctx, params); err != nil { // validates the params
	return nil, err
}
```

`RunMigrations` calls `InitGenesis` with the default (inert) genesis, so set the params **after** it.

---

## Juno (v31, ibc-go v10)

**Files:** `app/keepers/keepers.go`.

- **Keeper construction**
  - Create `UsdcMigrationKeeper` with `appKeepers.PacketForwardKeeper` as its ICS4 wrapper, before `TransferKeeper` (line 372).
  - `TransferKeeper` (line 372): pass `appKeepers.UsdcMigrationKeeper` instead of `appKeepers.PacketForwardKeeper`.
- **Transfer stack** (lines 608-615): after the PFM wrap, add `transferStack = usdcmigration.NewIBCMiddleware(appKeepers.UsdcMigrationKeeper, transferStack)`.
- **IBC v2:** none.

## Agoric (u23, ibc-go v10)

**File:** `golang/cosmos/app/app.go`.

- **Keeper construction**
  - Create `UsdcMigrationKeeper` with `app.PacketForwardKeeper` as its ICS4 wrapper.
  - `TransferKeeper` (line 647): pass `app.UsdcMigrationKeeper` instead of `app.PacketForwardKeeper`.
- **Transfer stack**
  - Wrap after vtransfer, before `ibcRouter.AddRoute` (line 695): `ics20TransferStack = usdcmigration.NewIBCMiddleware(app.UsdcMigrationKeeper, ics20TransferStack)`.
  - Do this on the inner `ibcRouter`, not on `vibcDynamicRouter`.
- **Holders:**
  - Vouchers held by SwingSet (vbank) accounts and the Fast USDC contracts are ordinary bank balances.
  - The bank freeze also stops Fast USDC flows that use this voucher, so coordinate with the Fast USDC operators.

## Xion (v31, ibc-go v10)

**File:** `app/app.go`.

- **Keeper construction**
  - Create `UsdcMigrationKeeper` with `app.IBCKeeper.ChannelKeeper` as its ICS4 wrapper, before `TransferKeeper` (line 665).
  - `TransferKeeper` constructor: pass `app.UsdcMigrationKeeper` as the ICS4 wrapper argument (the first `app.IBCKeeper.ChannelKeeper`).
- **Late `WithICS4Wrapper`:** line 819, `app.TransferKeeper.WithICS4Wrapper(cbStack)`, becomes
  `app.UsdcMigrationKeeper.WithICS4Wrapper(cbStack)`.
  - This also fixes an existing gap: PFM's copy of the transfer keeper (set at line 677) currently skips `cbStack`.
- **Transfer stack:** after the PFM wrap (lines 820-825), add `transferStack = usdcmigration.NewIBCMiddleware(app.UsdcMigrationKeeper, transferStack)`.
- **IBC v2** (lines 836-846): add `ibcv2TransferStack = usdcmigrationv2.NewIBCMiddleware(app.UsdcMigrationKeeper, ibcv2TransferStack)` before `AddRoute`.

## Coreum (tx-chain v8, ibc-go v10)

**File:** `app/app.go`.

- **Keeper construction**
  - Create `UsdcMigrationKeeper` with `app.PacketForwardKeeper` as its ICS4 wrapper.
  - `NewTransferKeeperWrapper` (line 629): pass `app.UsdcMigrationKeeper` instead of `app.PacketForwardKeeper`.
- **Late `WithICS4Wrapper`:** line 825, `app.TransferKeeper.WithICS4Wrapper(cbStack)`, becomes `app.UsdcMigrationKeeper.WithICS4Wrapper(cbStack)`.
- **Transfer stack:** after `wibctransfer.NewPurposeMiddleware` (line 834), add `ibcTransferStack = usdcmigration.NewIBCMiddleware(app.UsdcMigrationKeeper, ibcTransferStack)`.
- **IBC v2** (lines 845-853): wrap `transferStackV2` with `usdcmigrationv2.NewIBCMiddleware` before `AddRoute`.
- **DEX-locked balances:** voucher locked in DEX orders is not spendable, but this module never moves user balances, so nothing changes. The bank freeze will make DEX orders on the voucher fail to settle.

## Neutron (v11.4.0, ibc-go v10)

Repository: [solva-solutions/neutron](https://github.com/solva-solutions/neutron). **File:** `app/app.go`, in `WireICS20PreWasmKeeper`.

- **Keeper construction**
  - Create `UsdcMigrationKeeper` with `app.RateLimitingICS4Wrapper` as its ICS4 wrapper, after the rate-limit wrapper (line 1666) and before `TransferKeeper` (line 1669).
  - `wrapkeeper.NewKeeper` (line 1669): pass `app.UsdcMigrationKeeper` instead of `app.RateLimitingICS4Wrapper` (line 1673).
  - The copies of the transfer keeper (PFM's `SetTransferKeeper` at line 1683, the wasm keeper's `portSource`, the wasm IBC handler) then share the pointer. There is no late `WithICS4Wrapper`.
- **Transfer stack**
  - After the ibc-hooks wrap (line 1704), wrap it: `app.TransferStack = usdcmigration.NewIBCMiddleware(app.UsdcMigrationKeeper, &hooksTransferModule)`.
  - The `TransferStack` field (line 398) is `*ibchooks.IBCMiddleware`; change it to `ibcporttypes.IBCModule`. Its only other use is `AddRoute` (line 859).
- **IBC v2:** none. `ChannelKeeperV2` is only given to the wasm keeper.
- **Wasm keeper ICS4 wrapper** (line 808): leave `app.RateLimitingICS4Wrapper`. It only sends packets on contract ports. Contract ICS20 transfers (`IbcMsg::Transfer` and Neutron's `IBCTransfer` binding) go through the transfer keeper, so they are covered.
- **Governance filter.** Since the Proposal 9 recovery (v11.4.0, `app/govfilter`), a neutron-1 proposal may only contain `MsgSoftwareUpgrade`, `MsgStoreCode`, or a text proposal (`messages: []` passes too).
  - The filter runs at submit (ante handler and gov hooks, including messages nested in authz `MsgExec`) and at execution (the message router given to the gov keeper). It has no end height.
  - So governance can never execute `MsgUpdateParams`, `MsgReturnToNoble`, `MsgExecuteContract` or `MsgSetSendEnabled` on Neutron, and an authz grant from gov is not a way around it. [`proposals/neutron.json`](proposals/neutron.json) cannot be executed there.
  - Upgrade handlers are not filtered: they call keepers and message servers directly, as v11.4.0's own `RestrictCodeUpload` does.
- **Upgrade handler.** Neutron's handlers receive their keepers through `upgrades.UpgradeKeepers` (`app/upgrades/types.go`, filled at line 1335). Add `UsdcMigrationKeeper` there. The handler does everything the return proposal does on other chains, except the freeze:
  1. `SetParams`: `noble_channels: ["channel-30"]`, `base_denom: "uusdc"`, both blocks on.
  2. Remove the voucher's rate-limit path: `wasmkeeper.NewMsgServerImpl(keepers.WasmKeeper).ExecuteContract` with the gov module as sender and `{"remove_path":{"channel_id":"any","denom":"<voucher>"}}`, as v11.0.0's `IBCRateLimitsChangeRoles` does. Check `get_quotas` first, and skip this step if there is no path.
  3. `MsgReturnToNoble` through `usdcmigrationkeeper.NewMsgServerImpl`, with the gov module as `authority`, the Noble multisig as a constant `receiver`, and `amount` = the voucher's bank supply at that height (`BankKeeper.GetSupply`). The supply is below Noble's escrow (2,057,515.89 vs 2,058,476.26 USDC on 2026-10-26), so Noble accepts it whichever pending packets land first. Re-check that before tagging the release. Use a long `timeout_seconds` (up to 7 days).
  - No `SetSendEnabled(false)`: lifting a freeze would need another upgrade, and the redemption contract needs bank sends of the voucher (see [repayment.md](repayment.md#neutron)).
  - Run steps 2 and 3 in a `ctx.CacheContext()` and call `write()` only if both succeed; otherwise log the error and return `nil`, so a failed return does not halt the chain at the upgrade height. `write()` also re-emits the events that relayers and the redemption flow need.
  - The return packet is sent from PreBlock, not from a transaction. Neutron's SDK fork indexes pre-block events (`d004b2b3`), but test relaying it with Hermes on an in-place testnet before mainnet.
  - A return that times out or gets an error ack is refunded with nothing minted (`usdcmigration_return_refunded`). Governance cannot resend it, so a retry needs another upgrade, or a permissionless retry message added to the module for Neutron.
- **IBC rate limits:** the return packet goes through `app.RateLimitingICS4Wrapper`. The rate-limit contract (`neutron15aqgplxcavqhurr0g5wwtdw6025dknkqwkfh0n46gp2qjl6236cs2yd3nl`) has send quotas on this voucher for channel `any`: 35% of the supply per day, 70% per week. A full return is above them, so the handler removes that path first (step 2). Since v11.0.0 the gov module holds every role on the contract, with no timelock.
- **Holders:**
  - Self-redemption to Noble ends at the upgrade height, which is also the return height. Schedule it as late as the 2027-01-12 deadline safely allows, and announce it early: holders only regained access to their funds after the Proposal 9 recovery.
  - The native DEX (`x/dex`) is in withdraw-only mode. Its LPs withdraw the voucher, then redeem it in the redemption contract.
  - About 81% of the supply (1,672,301.71 of 2,057,515.89 USDC on 2026-10-26) is in one account, `neutron1yr29fd7uzdjp2jsq8hrta8mvyd6ex7vumn0shy`, which has never sent a transaction. Identify and contact its owner before the upgrade.

## Terra2 (v2.22, ibc-go v8)

**File:** `app/keepers/keepers.go`.

- **Keeper construction**
  - Create `UsdcMigrationKeeper` with `keepers.HooksICS4Wrapper` as its ICS4 wrapper, before `TransferKeeper` (line 366).
  - `TransferKeeper`: pass `keepers.UsdcMigrationKeeper` instead of `keepers.HooksICS4Wrapper`.
- **Transfer stack:** after `keepers.TransferStack = packetforward.NewIBCMiddleware(...)` (line 390), add `keepers.TransferStack = usdcmigration.NewIBCMiddleware(keepers.UsdcMigrationKeeper, keepers.TransferStack)`.

## Haqq (v1.9, ibc-go v8, EVM)

**File:** `app/app.go`.

- **Keeper construction**
  - Create `UsdcMigrationKeeper` with `app.PacketForwardKeeper` as its ICS4 wrapper.
  - Haqq's `transferkeeper.NewKeeper` (line 571): pass `app.UsdcMigrationKeeper` instead of `app.PacketForwardKeeper`.
- **Transfer stack:** after `erc20.NewIBCMiddleware` (line 643), add `transferStack = usdcmigration.NewIBCMiddleware(app.UsdcMigrationKeeper, transferStack)`. This makes it reject before ERC20 extension registration.
- **ERC20 precompile:** ICS20 sends from the EVM precompile go through the transfer keeper, so they are covered.
  - Check `token_pairs` for the voucher: an older native-ERC20 pair would hold vouchers in the erc20 module account.

## Impacthub / ixo (v8, ibc-go v8)

**File:** `app/keepers/keepers.go`.

- **Keeper construction**
  - Create `UsdcMigrationKeeper` with `appKeepers.PacketForwardKeeper` as its ICS4 wrapper.
  - `TransferKeeper` (line 426): pass it instead of `appKeepers.PacketForwardKeeper`.
- **Transfer stack:** after `ibcfee.NewIBCMiddleware` (line 684), add `transferStack = usdcmigration.NewIBCMiddleware(appKeepers.UsdcMigrationKeeper, transferStack)`.

## Archway (v10.1, ibc-go v8)

**File:** `app/app.go`.

- **Keeper construction**
  - Create `UsdcMigrationKeeper` with `app.Keepers.IBCFeeKeeper` as its ICS4 wrapper.
  - `TransferKeeper` (line 503): pass it instead of `app.Keepers.IBCFeeKeeper`.
- **Transfer stack:** after `ibchooks.NewIBCMiddleware` (line 682), add `transferStack = usdcmigration.NewIBCMiddleware(app.Keepers.UsdcMigrationKeeper, transferStack)`.
- **Asymmetric stack:** the ibc-hooks ICS4 wrapper is not on Archway's send path today. Keep it that way; this module doesn't need it.
