# Integrating `x/usdcmigration`

Line references point to the release analysed for each chain. Re-check them against the release you upgrade from.

| Chain | Line | Import | Noble channel | Voucher denom |
|---|---|---|---|---|
| Juno | v10 | `…/usdc-migration/v10` | `channel-224` | `ibc/4A482FA914A4B9B05801ED81C33713899F322B24F76A06F4B8FE872485EA22FF` |
| Agoric | v10 | `…/usdc-migration/v10` | `channel-62` | `ibc/FE98AAD68F02F03565E9FA39A5E627946699B2B07115889ED812D8BA639576A9` |
| Xion | v10 | `…/usdc-migration/v10` | `channel-2` | `ibc/F082B65C88E4B6D5EF1DB243CDA1D331D002759E938A0F5CD3FFDC5D53B3E349` |
| Coreum | v10 | `…/usdc-migration/v10` | `channel-19` | `ibc/E1E3674A0E4E1EF9C69646F9AF8D9497173821826074622D831BAB73CCB99A2D` * |
| Terra2 | v8 | `…/usdc-migration/v8` | `channel-253` | `ibc/2C962DAB9F57FE0921435426AE75196009FAA1981BF86991203C8411F8980FDB` |
| Haqq | v8 | `…/usdc-migration/v8` | `channel-4` | `ibc/BFF0D3805B50D93E2FA5C0B2DDF7E0B30A631076CD80BC12A48C0E95404B4A41` * |
| Impacthub (ixo) | v8 | `…/usdc-migration/v8` | `channel-26` | `ibc/6BBE9BD4246F8E04948D5A4EEE7164B2630263B9EBB5E7DC5F0A46C62A2FF97B` * |
| Archway | v8 | `…/usdc-migration/v8` | `channel-29` | `ibc/43897B9739BD63E3A08A88191999C632E052724AB96BD4C74AE31375C991F48D` |

\* Computed as `sha256("transfer/<channel>/uusdc")`, because the denom is missing from that chain's chain-registry assetlist. Confirm it on-chain before the upgrade: `query ibc-transfer denom-hash transfer/<channel>/uusdc`.

## Generic wiring

```go
import (
	"github.com/giansalex/usdc-migration/v10/x/usdcmigration"
	usdcmigrationkeeper "github.com/giansalex/usdc-migration/v10/x/usdcmigration/keeper"
	usdcmigrationtypes "github.com/giansalex/usdc-migration/v10/x/usdcmigration/types"
	usdcmigrationv2 "github.com/giansalex/usdc-migration/v10/x/usdcmigration/v2" // v10 + IBC v2 only
)

// app struct
UsdcMigrationKeeper *usdcmigrationkeeper.Keeper

// maccPerms
usdcmigrationtypes.ModuleName: {authtypes.Minter},

// store keys
usdcmigrationtypes.StoreKey,

// 1. BEFORE the transfer keeper: give it the ICS4 wrapper the transfer keeper used until now.
app.UsdcMigrationKeeper = usdcmigrationkeeper.NewKeeper(
	appCodec, runtime.NewKVStoreService(keys[usdcmigrationtypes.StoreKey]),
	app.BankKeeper,
	<previous transfer ICS4Wrapper>, // e.g. app.PacketForwardKeeper or app.IBCKeeper.ChannelKeeper
	authtypes.NewModuleAddress(govtypes.ModuleName).String(),
)

// 2. The transfer keeper sends through it. Every later copy of the transfer
//    keeper (PFM, wasm, ...) then shares this pointer.
app.TransferKeeper = ibctransferkeeper.NewKeeper(..., app.UsdcMigrationKeeper /* ics4Wrapper */, ...)

// 3. Once the transfer keeper is final:
app.UsdcMigrationKeeper.SetTransferKeeper(app.TransferKeeper)

// 4. Outermost on the ICS20 stack:
transferStack = usdcmigration.NewIBCMiddleware(app.UsdcMigrationKeeper, transferStack)

// 5. (v10, only if an IBC v2 transfer route exists) outermost on the v2 route:
ibcv2TransferStack = usdcmigrationv2.NewIBCMiddleware(app.UsdcMigrationKeeper, ibcv2TransferStack)

// 6. Module manager + InitGenesis/ExportGenesis order (after transfer):
usdcmigration.NewAppModule(appCodec, app.UsdcMigrationKeeper),
```

**Rules that matter:**

- **Never call `TransferKeeper.WithICS4Wrapper(x)` after this.** Use `app.UsdcMigrationKeeper.WithICS4Wrapper(x)` instead.
  - The transfer keeper is a value type. A copy taken before a late `WithICS4Wrapper`, such as PFM's from `SetTransferKeeper`, would keep its old wrapper and skip the send block.
  - With our keeper fixed in the constructor, every copy points at the same `*Keeper`. Re-pointing its inner wrapper later then reaches all of them.
- **Gov must not be in the bank keeper's blocked addresses.** `MsgReturnToNoble` returns `ErrAuthorityBlocked` otherwise.
  - Unblocking only means anyone may send funds to the gov module account, which only proposals can spend. simapp has done this since SDK v0.47.

**Upgrade handler:**

```go
storeUpgrades := storetypes.StoreUpgrades{Added: []string{usdcmigrationtypes.StoreKey}}

// in the handler, after RunMigrations:
params := usdcmigrationtypes.DefaultParams()
params.NobleChannels = []string{"channel-224"} // this chain's Noble channel
if err := app.UsdcMigrationKeeper.Params.Set(ctx, params); err != nil {
	return nil, err
}
```

`RunMigrations` calls `InitGenesis` with the default (inert) genesis, so set the params **after** it.

---

## Juno (v31, ibc-go v10)

**Files:** `app/keepers/keepers.go`; gov is not blocked.

- **Keeper construction**
  - Create `UsdcMigrationKeeper` with `appKeepers.PacketForwardKeeper` as its ICS4 wrapper, before `TransferKeeper` (line 372).
  - `TransferKeeper` (line 372): pass `appKeepers.UsdcMigrationKeeper` instead of `appKeepers.PacketForwardKeeper`.
  - After `PacketForwardKeeper.SetTransferKeeper` (line 385), call `appKeepers.UsdcMigrationKeeper.SetTransferKeeper(appKeepers.TransferKeeper)`.
- **Transfer stack** (lines 608-615): after the PFM wrap, add `transferStack = usdcmigration.NewIBCMiddleware(appKeepers.UsdcMigrationKeeper, transferStack)`.
- **IBC v2:** none.

## Agoric (u23, ibc-go v10)

**File:** `golang/cosmos/app/app.go`.

- **Keeper construction**
  - Create `UsdcMigrationKeeper` with `app.PacketForwardKeeper` as its ICS4 wrapper.
  - `TransferKeeper` (line 647): pass `app.UsdcMigrationKeeper` instead of `app.PacketForwardKeeper`.
  - After line 659, call `app.UsdcMigrationKeeper.SetTransferKeeper(app.TransferKeeper)`.
- **Transfer stack**
  - Wrap after vtransfer, before `ibcRouter.AddRoute` (line 695): `ics20TransferStack = usdcmigration.NewIBCMiddleware(app.UsdcMigrationKeeper, ics20TransferStack)`.
  - Do this on the inner `ibcRouter`, not on `vibcDynamicRouter`.
- **Gov is blocked:** in `BlockedAddresses()` (line 1229), skip `govtypes.ModuleName` in the switch.
- **Holders:**
  - Vouchers held by SwingSet (vbank) accounts and the Fast USDC contracts are ordinary bank balances.
  - The bank freeze also stops Fast USDC flows that use this voucher, so coordinate with the Fast USDC operators.

## Xion (v31, ibc-go v10)

**File:** `app/app.go`; gov is not blocked.

- **Keeper construction**
  - Create `UsdcMigrationKeeper` with `app.IBCKeeper.ChannelKeeper` as its ICS4 wrapper, before `TransferKeeper` (line 665).
  - `TransferKeeper` constructor: pass `app.UsdcMigrationKeeper` as the ICS4 wrapper argument (the first `app.IBCKeeper.ChannelKeeper`).
- **Late `WithICS4Wrapper`:** line 819, `app.TransferKeeper.WithICS4Wrapper(cbStack)`, becomes
  `app.UsdcMigrationKeeper.WithICS4Wrapper(cbStack)`. Then call `app.UsdcMigrationKeeper.SetTransferKeeper(app.TransferKeeper)`.
  - This also fixes an existing gap: PFM's copy of the transfer keeper (set at line 677) currently skips `cbStack`.
- **Transfer stack:** after the PFM wrap (lines 820-825), add `transferStack = usdcmigration.NewIBCMiddleware(app.UsdcMigrationKeeper, transferStack)`.
- **IBC v2** (lines 836-846): add `ibcv2TransferStack = usdcmigrationv2.NewIBCMiddleware(app.UsdcMigrationKeeper, ibcv2TransferStack)` before `AddRoute`.

## Coreum (tx-chain v8, ibc-go v10)

**File:** `app/app.go`.

- **Keeper construction**
  - Create `UsdcMigrationKeeper` with `app.PacketForwardKeeper` as its ICS4 wrapper.
  - `NewTransferKeeperWrapper` (line 629): pass `app.UsdcMigrationKeeper` instead of `app.PacketForwardKeeper`.
- **Late `WithICS4Wrapper`:** line 825, `app.TransferKeeper.WithICS4Wrapper(cbStack)`, becomes `app.UsdcMigrationKeeper.WithICS4Wrapper(cbStack)`. Then call `app.UsdcMigrationKeeper.SetTransferKeeper(app.TransferKeeper)`.
  - Pass the wrapper (not `.Keeper`) so Coreum's transfer-purpose context is kept.
- **Transfer stack:** after `wibctransfer.NewPurposeMiddleware` (line 834), add `ibcTransferStack = usdcmigration.NewIBCMiddleware(app.UsdcMigrationKeeper, ibcTransferStack)`.
- **IBC v2** (lines 845-853): wrap `transferStackV2` with `usdcmigrationv2.NewIBCMiddleware` before `AddRoute`.
- **Gov is blocked:** `app.ModuleAccountAddrs()` is passed to the bank keeper at line 422. Pass a copy with the gov address deleted.
- **DEX-locked balances:** voucher locked in DEX orders is not spendable, but this module never moves user balances, so nothing changes. The bank freeze will make DEX orders on the voucher fail to settle.

## Terra2 (v2.22, ibc-go v8)

**File:** `app/keepers/keepers.go`.

- **Keeper construction**
  - Create `UsdcMigrationKeeper` with `keepers.HooksICS4Wrapper` as its ICS4 wrapper, before `TransferKeeper` (line 366).
  - `TransferKeeper`: pass `keepers.UsdcMigrationKeeper` instead of `keepers.HooksICS4Wrapper`.
  - Then call `keepers.UsdcMigrationKeeper.SetTransferKeeper(keepers.TransferKeeper)`.
- **Transfer stack:** after `keepers.TransferStack = packetforward.NewIBCMiddleware(...)` (line 390), add `keepers.TransferStack = usdcmigration.NewIBCMiddleware(keepers.UsdcMigrationKeeper, keepers.TransferStack)`.
- **Gov is blocked:** `ModuleAccountAddrs()` (line 562) feeds the bank keeper (line 236). Delete the gov address the same way `alliance` is deleted.
- **Hooks:** every bank send runs tokenfactory `BlockBeforeSend` hooks. The mint and the IBC burn go through module accounts, so they are not affected.

## Haqq (v1.9, ibc-go v8, EVM)

**File:** `app/app.go`.

- **Keeper construction**
  - Create `UsdcMigrationKeeper` with `app.PacketForwardKeeper` as its ICS4 wrapper.
  - Haqq's `transferkeeper.NewKeeper` (line 571): pass `app.UsdcMigrationKeeper` instead of `app.PacketForwardKeeper`.
  - After `PacketForwardKeeper.SetTransferKeeper` (line 606), call `app.UsdcMigrationKeeper.SetTransferKeeper(app.TransferKeeper)`.
- **Transfer stack:** after `erc20.NewIBCMiddleware` (line 643), add `transferStack = usdcmigration.NewIBCMiddleware(app.UsdcMigrationKeeper, transferStack)`. This makes it reject before ERC20 extension registration.
- **Gov is blocked:** in `BlockedAddrs()` (line 1050), delete the gov module address.
- **ERC20 precompile:** ICS20 sends from the EVM precompile go through the transfer keeper, so they are covered.
  - Check `token_pairs` for the voucher: an older native-ERC20 pair would hold vouchers in the erc20 module account.

## Impacthub / ixo (v8, ibc-go v8)

**File:** `app/keepers/keepers.go`.

- **Keeper construction**
  - Create `UsdcMigrationKeeper` with `appKeepers.PacketForwardKeeper` as its ICS4 wrapper.
  - `TransferKeeper` (line 426): pass it instead of `appKeepers.PacketForwardKeeper`.
  - After line 441, call `SetTransferKeeper`.
- **Transfer stack:** after `ibcfee.NewIBCMiddleware` (line 684), add `transferStack = usdcmigration.NewIBCMiddleware(appKeepers.UsdcMigrationKeeper, transferStack)`.
- **Gov is blocked:** in `app/blocked.go`, add `govtypes.ModuleName: true` to `allowedReceivingModAcc`.
- **Mint restriction:** `WithMintCoinsRestriction` (line 290) only restricts liquid-stake pool denoms, so minting `ibc/…` is allowed.

## Archway (v10.1, ibc-go v8)

**File:** `app/app.go`; gov is not blocked.

- **Keeper construction**
  - Create `UsdcMigrationKeeper` with `app.Keepers.IBCFeeKeeper` as its ICS4 wrapper.
  - `TransferKeeper` (line 503): pass it instead of `app.Keepers.IBCFeeKeeper`.
  - Then call `SetTransferKeeper`.
- **Transfer stack:** after `ibchooks.NewIBCMiddleware` (line 682), add `transferStack = usdcmigration.NewIBCMiddleware(app.Keepers.UsdcMigrationKeeper, transferStack)`.
- **Asymmetric stack:** the ibc-hooks ICS4 wrapper is not on Archway's send path today. Keep it that way; this module doesn't need it.
