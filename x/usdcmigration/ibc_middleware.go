package usdcmigration

import (
	"encoding/json"

	errorsmod "cosmossdk.io/errors"

	sdk "github.com/cosmos/cosmos-sdk/types"

	transfertypes "github.com/cosmos/ibc-go/v10/modules/apps/transfer/types"
	channeltypes "github.com/cosmos/ibc-go/v10/modules/core/04-channel/types"
	porttypes "github.com/cosmos/ibc-go/v10/modules/core/05-port/types"
	"github.com/cosmos/ibc-go/v10/modules/core/exported"

	"github.com/disperze/usdc-migration/v10/x/usdcmigration/keeper"
	"github.com/disperze/usdc-migration/v10/x/usdcmigration/types"
)

var (
	_ porttypes.Middleware            = (*IBCMiddleware)(nil)
	_ porttypes.PacketDataUnmarshaler = (*IBCMiddleware)(nil)
)

// IBCMiddleware must be the outermost layer of the ICS20 transfer stack so it
// sees packets before PFM, ibc-hooks or other middleware act on them.
// Channel handshake callbacks pass through to the wrapped app, and sends go
// through the keeper, which enforces the send block.
type IBCMiddleware struct {
	porttypes.IBCModule
	porttypes.ICS4Wrapper
	keeper *keeper.Keeper
}

// NewIBCMiddleware wraps app (the existing transfer stack).
func NewIBCMiddleware(k *keeper.Keeper, app porttypes.IBCModule) IBCMiddleware {
	return IBCMiddleware{IBCModule: app, ICS4Wrapper: k, keeper: k}
}

// OnRecvPacket rejects the deprecated Noble denom arriving on a Noble channel
// with an error acknowledgement, so Noble refunds the sender.
func (im IBCMiddleware) OnRecvPacket(ctx sdk.Context, channelVersion string, packet channeltypes.Packet, relayer sdk.AccAddress) exported.Acknowledgement {
	var data transfertypes.FungibleTokenPacketData
	if err := json.Unmarshal(packet.GetData(), &data); err == nil &&
		im.keeper.GetParams(ctx).ShouldBlockRecv(packet.DestinationChannel, data.Denom) {
		return channeltypes.NewErrorAcknowledgement(
			errorsmod.Wrapf(types.ErrRecvBlocked, "%s on %s", data.Denom, packet.DestinationChannel),
		)
	}
	return im.IBCModule.OnRecvPacket(ctx, channelVersion, packet, relayer)
}

// OnAcknowledgementPacket releases the returned total when a return to Noble
// is answered with an error acknowledgement. The error ack of a return packet
// never reaches the transfer app: the return burned nothing, and the app's
// refund would mint the voucher to the authority.
func (im IBCMiddleware) OnAcknowledgementPacket(ctx sdk.Context, channelVersion string, packet channeltypes.Packet, acknowledgement []byte, relayer sdk.AccAddress) error {
	if data, ok := im.returnPacketData(ctx, packet); ok {
		var ack channeltypes.Acknowledgement
		if err := transfertypes.ModuleCdc.UnmarshalJSON(acknowledgement, &ack); err == nil && !ack.Success() {
			return im.keeper.OnReturnRefunded(ctx, packet.SourceChannel, packet.Sequence, data.Amount, types.RefundReasonErrorAck)
		}
	}
	return im.IBCModule.OnAcknowledgementPacket(ctx, channelVersion, packet, acknowledgement, relayer)
}

// OnTimeoutPacket releases the returned total when a return to Noble times
// out. Like an error ack, the timeout of a return packet never reaches the
// transfer app, so no voucher is minted back.
func (im IBCMiddleware) OnTimeoutPacket(ctx sdk.Context, channelVersion string, packet channeltypes.Packet, relayer sdk.AccAddress) error {
	if data, ok := im.returnPacketData(ctx, packet); ok {
		return im.keeper.OnReturnRefunded(ctx, packet.SourceChannel, packet.Sequence, data.Amount, types.RefundReasonTimeout)
	}
	return im.IBCModule.OnTimeoutPacket(ctx, channelVersion, packet, relayer)
}

// returnPacketData decodes the ICS20 data of packet and reports whether it is
// a return packet sent by MsgReturnToNoble.
func (im IBCMiddleware) returnPacketData(ctx sdk.Context, packet channeltypes.Packet) (transfertypes.FungibleTokenPacketData, bool) {
	var data transfertypes.FungibleTokenPacketData
	if packet.SourcePort != types.TransferPortID {
		return data, false
	}
	if err := json.Unmarshal(packet.GetData(), &data); err != nil {
		return data, false
	}
	return data, im.keeper.IsReturnPacket(ctx, packet.SourceChannel, data.Denom, data.Sender)
}

// UnmarshalPacketData delegates to the wrapped app so middleware stacked on
// top of this one (e.g. callbacks) keeps working.
func (im IBCMiddleware) UnmarshalPacketData(ctx sdk.Context, portID, channelID string, bz []byte) (any, string, error) {
	unmarshaler, ok := im.IBCModule.(porttypes.PacketDataUnmarshaler)
	if !ok {
		return nil, "", errorsmod.Wrapf(types.ErrUnsupportedApp, "%T does not implement PacketDataUnmarshaler", im.IBCModule)
	}
	return unmarshaler.UnmarshalPacketData(ctx, portID, channelID, bz)
}
