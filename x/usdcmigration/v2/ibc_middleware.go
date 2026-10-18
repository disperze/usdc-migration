// Package v2 wraps the IBC v2 transfer application. IBC v2 sends never go
// through an ICS4Wrapper, so the voucher send rule is enforced in OnSendPacket.
package v2

import (
	errorsmod "cosmossdk.io/errors"

	sdk "github.com/cosmos/cosmos-sdk/types"

	transfertypes "github.com/cosmos/ibc-go/v10/modules/apps/transfer/types"
	channeltypesv2 "github.com/cosmos/ibc-go/v10/modules/core/04-channel/v2/types"
	"github.com/cosmos/ibc-go/v10/modules/core/api"

	"github.com/giansalex/usdc-migration/v10/x/usdcmigration/keeper"
	"github.com/giansalex/usdc-migration/v10/x/usdcmigration/types"
)

var (
	_ api.IBCModule             = (*IBCMiddleware)(nil)
	_ api.PacketDataUnmarshaler = (*IBCMiddleware)(nil)
)

// IBCMiddleware must be the outermost layer of the IBC v2 transfer route.
type IBCMiddleware struct {
	app    api.IBCModule
	keeper *keeper.Keeper
}

// NewIBCMiddleware wraps app (the existing IBC v2 transfer stack).
func NewIBCMiddleware(k *keeper.Keeper, app api.IBCModule) IBCMiddleware {
	return IBCMiddleware{app: app, keeper: k}
}

// OnSendPacket rejects the Noble voucher unless it is sent by the authority.
func (im IBCMiddleware) OnSendPacket(ctx sdk.Context, sourceClient, destinationClient string, sequence uint64, payload channeltypesv2.Payload, signer sdk.AccAddress) error {
	if payload.SourcePort == types.TransferPortID {
		data, err := transfertypes.UnmarshalPacketData(payload.Value, payload.Version, payload.Encoding)
		if err == nil {
			if err := im.keeper.CheckSendDenom(ctx, data.Token.Denom.Path(), data.Sender); err != nil {
				return err
			}
		}
	}
	return im.app.OnSendPacket(ctx, sourceClient, destinationClient, sequence, payload, signer)
}

// OnRecvPacket implements api.IBCModule. Noble is not reachable over IBC v2,
// so receives pass through.
func (im IBCMiddleware) OnRecvPacket(ctx sdk.Context, sourceClient, destinationClient string, sequence uint64, payload channeltypesv2.Payload, relayer sdk.AccAddress) channeltypesv2.RecvPacketResult {
	return im.app.OnRecvPacket(ctx, sourceClient, destinationClient, sequence, payload, relayer)
}

// OnTimeoutPacket implements api.IBCModule.
func (im IBCMiddleware) OnTimeoutPacket(ctx sdk.Context, sourceClient, destinationClient string, sequence uint64, payload channeltypesv2.Payload, relayer sdk.AccAddress) error {
	return im.app.OnTimeoutPacket(ctx, sourceClient, destinationClient, sequence, payload, relayer)
}

// OnAcknowledgementPacket implements api.IBCModule.
func (im IBCMiddleware) OnAcknowledgementPacket(ctx sdk.Context, sourceClient, destinationClient string, sequence uint64, acknowledgement []byte, payload channeltypesv2.Payload, relayer sdk.AccAddress) error {
	return im.app.OnAcknowledgementPacket(ctx, sourceClient, destinationClient, sequence, acknowledgement, payload, relayer)
}

// UnmarshalPacketData delegates to the wrapped app.
func (im IBCMiddleware) UnmarshalPacketData(payload channeltypesv2.Payload) (any, error) {
	unmarshaler, ok := im.app.(api.PacketDataUnmarshaler)
	if !ok {
		return nil, errorsmod.Wrapf(types.ErrInvalidParams, "%T does not implement PacketDataUnmarshaler", im.app)
	}
	return unmarshaler.UnmarshalPacketData(payload)
}
