package keeper

import (
	errorsmod "cosmossdk.io/errors"
	"cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"

	transfertypes "github.com/cosmos/ibc-go/v8/modules/apps/transfer/types"
	clienttypes "github.com/cosmos/ibc-go/v8/modules/core/02-client/types"
	channeltypes "github.com/cosmos/ibc-go/v8/modules/core/04-channel/types"
	host "github.com/cosmos/ibc-go/v8/modules/core/24-host"

	"github.com/giansalex/usdc-migration/v8/x/usdcmigration/types"
)

// sendReturnPacket builds the ICS20 packet that returns amount of the Noble
// voucher over channelID and hands it to the next ICS4 wrapper. Unlike a
// transfer keeper send, no voucher is burned from, or escrowed for, any
// account: Noble releases the matching USDC from the channel escrow to
// receiver. The packet is sent below this keeper's own SendPacket, so the send
// block (which also covers the authority) does not apply to it. The packet is
// authenticated with the transfer module's channel capability.
func (k *Keeper) sendReturnPacket(ctx sdk.Context, channelID string, amount math.Int, receiver, memo string, timeoutTimestamp uint64) (uint64, error) {
	chanCap, ok := k.scopedTransferKeeper.GetCapability(ctx, host.ChannelCapabilityPath(types.TransferPortID, channelID))
	if !ok {
		return 0, errorsmod.Wrap(channeltypes.ErrChannelCapabilityNotFound, "transfer module does not own channel capability")
	}

	params := k.GetParams(ctx)
	data := transfertypes.NewFungibleTokenPacketData(
		types.VoucherPath(channelID, params.BaseDenom), amount.String(), k.authority, receiver, memo,
	)
	if err := data.ValidateBasic(); err != nil {
		return 0, err
	}
	return k.ics4Wrapper.SendPacket(ctx, chanCap, types.TransferPortID, channelID, clienttypes.ZeroHeight(), timeoutTimestamp, data.GetBytes())
}

// IsReturnPacket reports whether an ICS20 packet sent over sourceChannel is a
// return to Noble built by MsgReturnToNoble. Since the send block rejects the
// voucher for every sender, the authority included, only the module can have
// sent such a packet.
func (k *Keeper) IsReturnPacket(ctx sdk.Context, sourceChannel, denomPath, sender string) bool {
	params := k.GetParams(ctx)
	return sender == k.authority && params.IsNobleChannel(sourceChannel) &&
		denomPath == types.VoucherPath(sourceChannel, params.BaseDenom)
}
