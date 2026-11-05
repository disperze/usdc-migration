# Repayment plan

Holders of the Noble USDC voucher are repaid in **USDC (Injective)**, 1:1. Each chain picks one of three options:

1. **[Redemption contract](#option-1-redemption-contract)**, on chains that support smart contracts: holders swap the old voucher for the new USDC in a contract, at any time.
2. **[Distribution in a software upgrade](#option-2-distribution-in-a-software-upgrade)**, on any chain: the upgrade handler pays every holder from a snapshot. Holders do nothing.
3. **[Merkle claim on Injective](#option-3-merkle-claim-on-injective)**, on any chain: holders claim the new USDC on Injective with a merkle proof from a snapshot, proving ownership of their address with a Keplr signature. Nothing more runs on the chain after the return.

Both start once the [return](../README.md#governance-runbook) has settled and the Noble multisig has converted the USDC. Neither is bound by Circle's 2027-01-12 deadline, which only applies to the return.

## Common steps

1. **Return.** The return proposal (`MsgReturnToNoble` + `MsgSetSendEnabled`) releases Noble's escrow for the chain to the Noble multisig and freezes the old voucher at the same height. That height is in the `usdcmigration_return_to_noble` event.
   - **Neutron** has no return proposal: its governance cannot execute these messages. The upgrade that adds the module returns the escrow in its handler and does not freeze the voucher (see [Neutron](#neutron)).
2. **Conversion.** The multisig converts the USDC to USDC (Injective). For options 1 and 2 it sends it to the chain over IBC; for option 3 it stays on Injective.
   - Use the chain's direct transfer channel to Injective, the one in the chain registry, so the new USDC lands as the voucher DEXes and wallets already list: `ibc/<hash of transfer/<injective channel>/<injective usdc denom>>`.
   - The receiver depends on the option: the redemption contract (option 1), the distribution account (option 2), or the claim contract on Injective (option 3).
   - Conversion and bridge costs come out of the converted amount (see [Cost](#cost)).
3. **Repayment**, with option 1, 2 or 3 below.

**The amount to repay** is the voucher supply at the return height (`query bank total-supply-of ibc/<voucher hash> --height <h>`), not the escrow released. The two differ on some chains ([table](proposals/README.md#values-already-filled-in-checked-on-chain-2026-10-26)):

- Escrow **above** supply (Agoric, Neutron, Terra2): the surplus is kept for holders not in the bank supply (Agoric contracts, below) and for costs. What remains goes to the chain's community pool.
- Escrow **below** supply (Archway, 1.69 USDC short): the multisig tops it up, or holders are paid pro rata.

If the converted amount is below what is owed, every holder is paid pro rata (`rate = funded / owed`), so nobody is left at the end of the queue with nothing.

## Options per chain

Smart contract runtime and upload access (`cosmwasm/wasm/v1/codes/params`, checked on 2026-10-29):

| Chain | Runtime | Code upload | Options |
|---|---|---|---|
| Agoric | Hardened JS (Zoe), deployed by core-eval | governance | 2, or 1 with a Zoe contract |
| Coreum | CosmWasm | anyone | 1 or 2 |
| Neutron | CosmWasm | governance (`Nobody` since v11.4.0; gov `MsgStoreCode` only) | 1, with Astroport's transmuter (already uploaded) |
| Terra2 | CosmWasm | anyone | 1 or 2 |
| Haqq | EVM | anyone | 2, or 1 with a Solidity contract |
| Impacthub (ixo) | CosmWasm | governance (`Nobody`, instantiate too) | 1 or 2 |
| Archway | CosmWasm | anyone | 1 or 2 |

Option 3 needs nothing on the chain itself, so it is open to every chain as well.

Where upload is restricted, the contract is stored and instantiated by a gov proposal (`MsgStoreAndInstantiateContract`), which can be the same proposal that unfreezes the voucher. Neutron is the exception: it reuses a pool contract already on-chain, so no upload is needed (see [Neutron](#neutron)).

## Option 1: redemption contract

A contract holds the new USDC. A holder sends it old vouchers and gets the same amount of new USDC back. There is no snapshot and no deadline: whoever holds the voucher can redeem it, now or years later.

### Flow

1. **Deploy.** Store and instantiate the contract with:
   - `old_denom`: the Noble USDC voucher, `ibc/<voucher hash>`.
   - `new_denom`: the USDC (Injective) voucher.
   - `rate`: `1` unless the funding is short (see [Common steps](#common-steps)).
   - wasm `admin`: the gov module address, so only governance can migrate it.
2. **Fund.** The multisig sends the new USDC from Injective straight to the contract address. Anyone can check that the contract balance covers the voucher supply × `rate`.
3. **Unfreeze.** A gov proposal re-enables bank sends of the old voucher (`MsgSetSendEnabled` with `enabled: true`).
   - It is needed because the freeze also blocks sending the voucher to a contract: wasmd checks `SendEnabled` on the funds attached to `MsgExecuteContract`.
   - From this height the voucher is backed again, by the contract, so trading it is safe. The IBC send and receive blocks of `x/usdcmigration` stay on: the voucher still cannot leave the chain.
   - Submit it only once the contract is funded, never before.
4. **Redeem.** Holders execute `redeem` with their old vouchers attached.

### Neutron

Neutron uses option 1, but with a contract that is already on-chain, because of its governance filter (v11.4.0, `app/govfilter`). A proposal there may only contain `MsgSoftwareUpgrade`, `MsgStoreCode` or a text proposal. Governance cannot instantiate, execute or migrate contracts, or send `MsgSetSendEnabled`.

The redemption contract is a new instance of **Astroport's transmuter pool**: a pool that swaps 1:1 with no fees. Its code is already uploaded, so this needs no gov proposal at all. Values checked on-chain on 2026-11-05:

| | Value |
|---|---|
| Code ID | `5401`: `astroport-pair-transmuter` 1.1.2, checksum `595d1368…8dcc3`, `instantiate_permission: Everybody` |
| Old denom (Noble USDC voucher) | `ibc/B559A80D62249C8AA07A380E2A2BEA6E5CA9A6F079C912C3A9E9B494105E4F81` (6 decimals in the coin registry) |
| New denom (USDC Injective) | `ibc/02BCFF6F786C7CBCDFE1AAFD8CFAC70D2AA9D57CF63AC0915048B666C650D8C5`: Injective's `erc20:0xa00C59fF5a080D2b954d0c75e46E22a0c371235a` over Neutron `channel-60` (Injective `channel-177`) |
| Astroport factory | `neutron1hptk0k5kng7hjy35vmh009qd5m6l33609nypgf2yc6nqnewduqasxplt4e` |
| Astroport native coin registry | `neutron1jzzv6r5uckwd64n6qan3suzker0kct5w565f6529zjyumfcx96kqtcswn3` (v1.1.0) |

**Use code 5401 only.** Transmuter builds 1.1.0 and 1.1.1 paid out for offer assets that were not pool assets. 1.1.2 rejects them, and its `migrate` only accepts those two versions. Do not reuse an older transmuter instance: create a new one.

#### Flow

1. **Upgrade.** The software upgrade that adds the module also returns the escrow in its handler (see [integration.md](integration.md#neutron-v1140-ibc-go-v10)).
   - It does **not** freeze the voucher. Lifting a freeze would need another upgrade, and holders need bank sends to put the voucher into the pool.
   - The IBC send and receive blocks keep the voucher on Neutron.
2. **Bridge.** The Noble multisig converts the USDC to USDC (Injective) and sends it over Injective `channel-177` to a **Neutron multisig** with the same signers.
   - Before sending, check Neutron's rate-limit quotas on the new denom (`get_quotas`). A path on that channel could reject one transfer of the whole amount, and only an upgrade can remove it, so split the transfer if needed.
3. **Register the decimals.** The pool reads each denom's decimals from Astroport's native coin registry when it is created. The new denom is not in it yet.
   - The registry's `register` is permissionless: it only requires attaching some of the coin, which it sends back.

   ```sh
   neutrond tx wasm execute neutron1jzzv6r5uckwd64n6qan3suzker0kct5w565f6529zjyumfcx96kqtcswn3 \
     '{"register":{"native_coins":[["ibc/02BCFF6F786C7CBCDFE1AAFD8CFAC70D2AA9D57CF63AC0915048B666C650D8C5",6]]}}' \
     --amount 1ibc/02BCFF6F786C7CBCDFE1AAFD8CFAC70D2AA9D57CF63AC0915048B666C650D8C5 --from <key>
   ```

   - Do it early, then check that `{"native_token":{"denom":"<new denom>"}}` returns `6`.
   - The first registration wins and only Astroport's owner can change it. Wrong decimals would make every swap off by a power of ten, because the pool fixes its normalisation at instantiation.
4. **Instantiate** code 5401 from the Neutron multisig, with the gov module as wasm `admin`, so only a software upgrade can migrate it:

   ```sh
   neutrond tx wasm instantiate 5401 '{"pair_type":{"custom":"transmuter"},"asset_infos":[{"native_token":{"denom":"ibc/B559A80D62249C8AA07A380E2A2BEA6E5CA9A6F079C912C3A9E9B494105E4F81"}},{"native_token":{"denom":"ibc/02BCFF6F786C7CBCDFE1AAFD8CFAC70D2AA9D57CF63AC0915048B666C650D8C5"}}],"token_code_id":36,"factory_addr":"neutron1hptk0k5kng7hjy35vmh009qd5m6l33609nypgf2yc6nqnewduqasxplt4e","init_params":null}' \
     --label "usdc-noble-redemption" --admin neutron10d07y265gmmuvt4z0w9aw880jnsr700j7a68v5 --from <multisig>
   ```

   - The factory's transmuter pair type is `permissioned`, so only Astroport's factory owner can create one through the factory. Instantiating the code directly needs no permission. The pool is then not registered in the factory, so it does not appear in Astroport's UI or router.
   - Holders use it from the CLI or a simple redeem page. Astroport can also create the pool through its factory, if it agrees, to list it.
   - The pool creates its LP denom with tokenfactory, which has no creation fee on Neutron.
5. **Fund.** The multisig sends `provide_liquidity` with only the new denom, and `receiver` set to **the pool's own address**:

   ```sh
   neutrond tx wasm execute <pool> '{"provide_liquidity":{"assets":[{"info":{"native_token":{"denom":"ibc/02BCFF6F786C7CBCDFE1AAFD8CFAC70D2AA9D57CF63AC0915048B666C650D8C5"}},"amount":"<amount>"}],"receiver":"<pool>"}}' \
     --amount <amount>ibc/02BCFF6F786C7CBCDFE1AAFD8CFAC70D2AA9D57CF63AC0915048B666C650D8C5 --from <multisig>
   ```

   - Whoever holds the LP tokens can withdraw the pool's assets. Minted to the pool, they can never be withdrawn, because the pool does not spend its own LP tokens. Nobody keeps custody of the funds.
   - Do not use the gov module as receiver: it is a module account, so the bank keeper refuses to send it the minted LP tokens.
   - `<amount>` must be **at least the voucher supply at the upgrade height**. The pool pays strictly 1:1, first come first served, with no pro-rata rate. An underfunded pool would leave the last holders with nothing.
   - The upgrade returns the supply, not the larger escrow, so conversion and bridge costs must be covered on top.
   - If the funding is short anyway, the multisig tops it up before redemptions open.
6. **Redeem.** Holders swap the old voucher for the new one:

   ```sh
   neutrond tx wasm execute <pool> '{"swap":{"offer_asset":{"info":{"native_token":{"denom":"ibc/B559A80D62249C8AA07A380E2A2BEA6E5CA9A6F079C912C3A9E9B494105E4F81"}},"amount":"1000000"}}}' \
     --amount 1000000ibc/B559A80D62249C8AA07A380E2A2BEA6E5CA9A6F079C912C3A9E9B494105E4F81 --from <key>
   ```

#### Differences from the generic contract

- **Rate:** strictly 1:1, with no `rate` parameter; see the funding requirement in step 5.
- **Old vouchers are kept, not burned.** They stay in the pool, so the amount still to redeem is the voucher supply minus the pool's voucher balance.
- **Swaps go both ways.** Anyone can swap USDC (Injective) back into the pool for vouchers. That adds as much as it takes out, so the backing of the remaining vouchers is unchanged.
- **Anyone can also `provide_liquidity`.** Providing vouchers and withdrawing the new denom is the same as a swap, so it needs no protection.
- **Before step 5 the voucher moves on Neutron without backing.** A swap is paid in full or rejected, so nobody can redeem early at another holder's expense.

### Contract

| Message | Behaviour |
|---|---|
| `redeem { recipient? }` | Accepts only `old_denom` in `funds`. Sends `amount × rate` of `new_denom` to `recipient` (default: the sender) and burns the received vouchers with `BankMsg::Burn`, so the old supply left is exactly what is still unredeemed. Fails if the contract cannot pay in full. |
| query `config` | `old_denom`, `new_denom`, `rate`. |
| query `stats` | Total redeemed and new USDC still available. |

No owner, no pause, no withdraw function. Anything else (a sweep after some years, a rate change) needs a gov `MsgMigrateContract`.

### Why it fits

- **Contracts and modules that hold the voucher need no special handling.** DEX pools, vaults and Neutron's withdraw-only `x/dex` let their users withdraw the voucher, and the user then redeems it. The upgrade option has to look through each of them instead.
- **Inactive holders lose nothing.** Their share waits in the contract.
- **Agoric**: USDC held in contracts (the Fast USDC pool, the PSM) is not in the bank supply, because `x/vbank` burns it when it enters a contract and mints it again when it leaves. With a redemption contract, it becomes a voucher again on withdrawal and can be redeemed like any other. Fund the contract from the escrow released, not just the bank supply, to cover it. On Agoric the contract is a Zoe contract deployed by core-eval, not the CosmWasm one.
- **Haqq**: the contract is a Solidity contract working on the ERC-20 representations of both denoms (`x/erc20` token pairs). Confirm with the Haqq core team that both pairs exist and that converting the voucher is not blocked; otherwise use option 2.

## Option 2: distribution in a software upgrade

A software upgrade pays every holder at the return height directly from the upgrade handler. Holders do nothing, and the voucher stays frozen.

### Flow

1. **Snapshot.** Read every balance of the old voucher at the return height, from a node that still has that state (archive, or pruning off):

   ```sh
   <binary> query bank denom-owners ibc/<voucher hash> --height <h> --page-limit 1000 -o json
   ```

   The sum must equal the voucher supply at that height.
2. **Look through module accounts and contracts.** A balance held by a DEX, a vault or a pool belongs to its depositors, and the new USDC would be stuck if paid to the contract itself. For each such holder, split its balance among its depositors by their position at the same height (for example, LP shares of a pool). Known cases:
   - Neutron: the `x/dex` module account (LP positions), and pair contracts holding the voucher.
   - Agoric: the Fast USDC pool and the PSM, read from vstorage (see [proposals/README.md](proposals/README.md#before-submitting)).
   - Coreum: vouchers locked in DEX orders are still in the owner's balance, so nothing to do.
3. **Publish** the payout list (`address,amount` in the smallest unit, sorted by address) in this repo, with its `sha256`, for holders to check before the upgrade proposal.
4. **Fund.** The multisig sends the new USDC from Injective to the distribution account: the `usdcmigration` module address, `authtypes.NewModuleAddress("usdcmigration")` in the chain's bech32 prefix.
   - Nobody holds a key for it: only code, such as the upgrade handler, can move its funds.
   - The module has no module-account permissions, so the address is normally not in the bank blocked addresses. Check the chain's `BlockedAddrs` before sending.
5. **Upgrade.** A software upgrade proposal whose handler pays the list (below). It goes through the chain's usual release process: the core team ships the binary, validators upgrade at the height.

### Upgrade handler

The payout list is embedded in the binary, so the reviewed file is exactly what is paid:

```go
//go:embed payouts.csv
var payoutsCSV []byte // sha256 must match the published list

func distribute(ctx sdk.Context, bk bankkeeper.Keeper, dk distrkeeper.Keeper) error {
	from := authtypes.NewModuleAddress(usdcmigrationtypes.ModuleName)
	payouts, total, err := parsePayouts(payoutsCSV) // []{addr, amount}, sum
	if err != nil {
		return err
	}

	funded := bk.GetBalance(ctx, from, newUSDCDenom).Amount
	rate := math.LegacyOneDec() // pro rata only if the funding is short
	if funded.LT(total) {
		rate = math.LegacyNewDecFromInt(funded).QuoInt(total)
	}

	for _, p := range payouts {
		amt := rate.MulInt(p.Amount).TruncateInt()
		if err := bk.SendCoins(ctx, from, p.Addr, sdk.NewCoins(sdk.NewCoin(newUSDCDenom, amt))); err != nil {
			return err
		}
	}

	// The surplus, if any, goes to the community pool.
	rest := bk.GetBalance(ctx, from, newUSDCDenom)
	if rest.IsPositive() {
		return dk.FundCommunityPool(ctx, sdk.NewCoins(rest), from)
	}
	return nil
}
```

- An error in the handler halts the chain at the upgrade height. Run the upgrade on a mainnet export first (in-place testnet), with the real distribution account balance.
- The keeper's `SendCoins` checks neither `SendEnabled` nor blocked addresses, so the handler must not pay module accounts: they were resolved in the look-through step.
- Optional: burn the old voucher from every plain account that was paid, so it disappears from wallets. Never burn it from module accounts or contracts: it would break their internal accounting.

## Option 3: merkle claim on Injective

The new USDC stays on Injective, in a claim contract. Holders claim their share there with a merkle proof from a snapshot, and prove they own their address on the chain with a Keplr signature. After the return proposal nothing else has to run on the chain: no contract, no unfreeze, no second upgrade. It suits chains with little activity left, and keeps the funds safe if the chain later halts.

### Flow

1. **Snapshot and look-through**, as in [option 2](#option-2-distribution-in-a-software-upgrade) (steps 1 and 2), at the return height.
   - Only accounts with a key can sign a claim. Balances held by contracts, module accounts and multisigs go through the look-through to their depositors.
   - What cannot be resolved that way is left out of the tree, listed separately, and paid by the admin case by case once the owner proves ownership.
2. **Build the tree.** One merkle tree per chain. Each leaf is `sha256("<source chain id>:<address>:<amount>")`, with `amount` in the smallest unit and already scaled by the chain's `rate` (see [Common steps](#common-steps)). Pairs are hashed sorted, `sha256(min(a, b) || max(a, b))`.
3. **Publish** the list, the root and the list's `sha256` in this repo, for holders to check before the root is registered.
4. **Deploy** the claim contract on Injective. Its code is stored under Injective's wasm upload rules (a governance proposal or an allowed uploader). One contract serves every chain that picks this option.
5. **Fund.** The multisig sends the chain's total USDC (Injective) to the contract, on Injective. No IBC transfer to the chain is needed.
6. **Register** the chain's root (`register_merkle_root`). A root cannot be changed once set.
7. **Claim.** Holders claim from a web page, any time before the expiry.

### Ownership with Keplr

A holder's address on the chain and their Injective address come from different keys, so the claim cannot rely on converting one address into the other. The holder proves ownership instead by signing the claim with the chain's key in Keplr, using ADR-36 arbitrary signing (`signArbitrary`), which costs nothing and sends no transaction on the chain:

```js
const data = JSON.stringify({
  claim_contract: "inj1<claim contract>",
  source_chain_id: sourceChainId,
  address,                 // the holder's address on the chain, in the snapshot
  amount,                  // from the published list
  recipient: "inj1<recipient>",
});
const { pub_key, signature } = await window.keplr.signArbitrary(sourceChainId, address, data);
```

The fields are in this exact order, with no spaces, so the contract can rebuild the same bytes. `claim_contract` stops the signature from being replayed on another contract. The `recipient` can be any Injective address: the holder adds Injective in Keplr if they have no address there yet.

The page then sends `claim` on Injective:

```json
{
  "claim": {
    "source_chain_id": "<source chain id>",
    "address": "<holder address on the chain>",
    "amount": "<amount>",
    "recipient": "inj1<recipient>",
    "proof": ["<hex>", "..."],
    "pub_key": "<base64, from Keplr>",
    "signature": "<base64, from Keplr>"
  }
}
```

The signature fixes the recipient, so anyone can submit the transaction: the holder, or a claim service that pays the INJ gas for holders who have none.

### Contract

| Message | Behaviour |
|---|---|
| `register_merkle_root { source_chain_id, bech32_prefix, merkle_root, total }` | Admin only, once per chain. Fails if the contract's balance does not cover the totals of all registered roots. |
| `claim { … }` | Anyone can send it. Checks, in order: not expired; `(source_chain_id, address)` not claimed yet; the leaf and `proof` match the chain's root; `address` has the chain's `bech32_prefix` and its bytes are `ripemd160(sha256(pub_key))`; `signature` is valid for `pub_key` over the ADR-36 sign doc built from `data` (below). Then sends `amount` to `recipient` and marks the address claimed. |
| `sweep {}` | Admin only, after the expiry. Sends what is left to a destination fixed at instantiation. |
| query `config`, `merkle_root { source_chain_id }`, `is_claimed { source_chain_id, address }` | |

The ADR-36 sign doc that Keplr signs, with keys sorted and no spaces, is hashed with `sha256` and checked with CosmWasm's `secp256k1_verify`:

```json
{"account_number":"0","chain_id":"","fee":{"amount":[],"gas":"0"},"memo":"","msgs":[{"type":"sign/MsgSignData","value":{"data":"<base64 of data>","signer":"<address>"}}],"sequence":"0"}
```

- Accounts with Ethereum-style keys (`eth_secp256k1`) derive the address with keccak-256 instead. The contract supports both key types; test each one against real Keplr signatures before deploying.
- Instantiate with an expiry (for example two years) and the sweep destination. The wasm admin is the repayment multisig on Injective, with the same signers as the Noble multisig. State this in the return proposal: the claim contract is not controlled by the chain's governance.

## Comparison

| | Option 1: contract | Option 2: upgrade | Option 3: claim on Injective |
|---|---|---|---|
| Chains | smart contract chains | any | any |
| Where holders are paid | on the chain | on the chain | on Injective |
| Holder action | redeem, any time | none | sign with Keplr and claim, before the expiry |
| Snapshot | not needed | at the return height, published for review | at the return height, published with the root |
| Vouchers in DEXes, vaults, contracts | users withdraw, then redeem | look-through per holder, by hand | look-through per holder, by hand |
| Old voucher after repayment | unfrozen, backed by the contract, burned on redeem | stays frozen (optionally burned) | stays frozen |
| Governance on the chain after the return | store/instantiate (if restricted) + unfreeze proposal | software upgrade (binary release, validator coordination) | none |
| Unclaimed funds | stay in the contract, for gov to decide later | none; surplus to the community pool | swept after the expiry to a fixed destination |
| New code to review | the contract | the upgrade handler and the payout list | the claim contract, the claim page and the merkle trees |

## Cost

The current plan is to first implement this solution, and then apply for a grant from other larger chains (Osmosis, Cosmos Hub). Costs to cover:

- Converting and bridging the USDC to Injective, and the IBC transfer to each chain.
- Writing and auditing the redemption contract (one CosmWasm contract for all CosmWasm chains, plus Zoe and Solidity versions if Agoric or Haqq choose option 1).
- Building the snapshot and look-through lists, and testing the upgrade handler on a mainnet export, for chains that choose option 2.
- Writing and auditing the claim contract on Injective, the claim page, and a claim service paying INJ gas, for chains that choose option 3.
