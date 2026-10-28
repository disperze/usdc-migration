package keeper

import (
	"time"

	errorsmod "cosmossdk.io/errors"

	sdk "github.com/cosmos/cosmos-sdk/types"

	transfertypes "github.com/cosmos/ibc-go/v8/modules/apps/transfer/types"
	clienttypes "github.com/cosmos/ibc-go/v8/modules/core/02-client/types"
	channeltypes "github.com/cosmos/ibc-go/v8/modules/core/04-channel/types"
	host "github.com/cosmos/ibc-go/v8/modules/core/24-host"

	"github.com/disperze/usdc-migration/v8/x/usdcmigration/types"
)

// sendReturnPacket builds the ICS20 packet of msg that returns the Noble
// voucher and hands it to the next ICS4 wrapper. Unlike a transfer
// keeper send, no voucher is burned from, or escrowed for, any account: Noble
// releases the matching USDC from the channel escrow to the receiver. The
// packet is sent below this keeper's own SendPacket, so the send block (which
// also covers the authority) does not apply to it. The packet is
// authenticated with the transfer module's channel capability.
func (k *Keeper) sendReturnPacket(ctx sdk.Context, msg *types.MsgReturnToNoble, baseDenom string) (uint64, error) {
	chanCap, ok := k.scopedTransferKeeper.GetCapability(ctx, host.ChannelCapabilityPath(types.TransferPortID, msg.ChannelId))
	if !ok {
		return 0, errorsmod.Wrap(channeltypes.ErrChannelCapabilityNotFound, "transfer module does not own channel capability")
	}

	data := transfertypes.NewFungibleTokenPacketData(
		types.VoucherPath(msg.ChannelId, baseDenom), msg.Amount.String(), k.authority, msg.Receiver, msg.Memo,
	)
	if err := data.ValidateBasic(); err != nil {
		return 0, err
	}

	timeoutSeconds := msg.TimeoutSeconds
	if timeoutSeconds == 0 {
		timeoutSeconds = types.DefaultTimeoutSeconds
	}
	timeout := uint64(ctx.BlockTime().Add(time.Duration(timeoutSeconds) * time.Second).UnixNano())

	return k.ics4Wrapper.SendPacket(ctx, chanCap, types.TransferPortID, msg.ChannelId, clienttypes.ZeroHeight(), timeout, data.GetBytes())
}

// IsReturnPacket reports whether an ICS20 packet sent over sourceChannel is a
// return to Noble built by MsgReturnToNoble. Since the send block rejects the
// voucher for every sender, the authority included, only the module can have
// sent such a packet. The match uses the current params, so they must not
// change while a return is in flight (see docs/integration.md).
func (k *Keeper) IsReturnPacket(ctx sdk.Context, sourceChannel, denomPath, sender string) bool {
	params := k.GetParams(ctx)
	return sender == k.authority && params.IsNobleChannel(sourceChannel) &&
		denomPath == types.VoucherPath(sourceChannel, params.BaseDenom)
}
