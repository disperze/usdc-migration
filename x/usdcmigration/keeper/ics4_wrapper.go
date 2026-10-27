package keeper

import (
	"encoding/json"

	errorsmod "cosmossdk.io/errors"

	sdk "github.com/cosmos/cosmos-sdk/types"

	transfertypes "github.com/cosmos/ibc-go/v10/modules/apps/transfer/types"
	clienttypes "github.com/cosmos/ibc-go/v10/modules/core/02-client/types"
	porttypes "github.com/cosmos/ibc-go/v10/modules/core/05-port/types"
	"github.com/cosmos/ibc-go/v10/modules/core/exported"

	"github.com/disperze/usdc-migration/v10/x/usdcmigration/types"
)

var _ porttypes.ICS4Wrapper = (*Keeper)(nil)

// SendPacket rejects ICS20 packets carrying the Noble voucher, whoever the
// sender is, then forwards to the next ICS4 wrapper. The check runs before
// forwarding so a rejected packet never gets a sequence or commitment.
// MsgReturnToNoble sends directly to the next wrapper and is not affected.
func (k *Keeper) SendPacket(
	ctx sdk.Context,
	sourcePort string,
	sourceChannel string,
	timeoutHeight clienttypes.Height,
	timeoutTimestamp uint64,
	data []byte,
) (uint64, error) {
	if sourcePort == types.TransferPortID {
		if err := k.CheckSend(ctx, data); err != nil {
			return 0, err
		}
	}
	return k.ics4Wrapper.SendPacket(ctx, sourcePort, sourceChannel, timeoutHeight, timeoutTimestamp, data)
}

// WriteAcknowledgement forwards to the next ICS4 wrapper.
func (k *Keeper) WriteAcknowledgement(ctx sdk.Context, packet exported.PacketI, ack exported.Acknowledgement) error {
	return k.ics4Wrapper.WriteAcknowledgement(ctx, packet, ack)
}

// GetAppVersion forwards to the next ICS4 wrapper.
func (k *Keeper) GetAppVersion(ctx sdk.Context, portID, channelID string) (string, bool) {
	return k.ics4Wrapper.GetAppVersion(ctx, portID, channelID)
}

// CheckSend returns ErrSendBlocked if the ICS20 v1 packet data carries the
// Noble voucher. Data that is not ICS20 is ignored.
func (k *Keeper) CheckSend(ctx sdk.Context, data []byte) error {
	var packet transfertypes.FungibleTokenPacketData
	if err := json.Unmarshal(data, &packet); err != nil {
		return nil
	}
	return k.CheckSendDenom(ctx, packet.Denom)
}

// CheckSendDenom is CheckSend for callers that already decoded the packet (IBC v2).
func (k *Keeper) CheckSendDenom(ctx sdk.Context, denomPath string) error {
	if k.GetParams(ctx).ShouldBlockSend(denomPath) {
		return errorsmod.Wrapf(types.ErrSendBlocked, "%s can only be returned to Noble with MsgReturnToNoble", denomPath)
	}
	return nil
}
