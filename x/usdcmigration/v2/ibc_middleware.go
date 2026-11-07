// Package v2 wraps the IBC v2 transfer application. IBC v2 sends never go
// through an ICS4Wrapper, so the voucher send rule is enforced in OnSendPacket.
package v2

import (
	errorsmod "cosmossdk.io/errors"

	sdk "github.com/cosmos/cosmos-sdk/types"

	transfertypes "github.com/cosmos/ibc-go/v10/modules/apps/transfer/types"
	channeltypesv2 "github.com/cosmos/ibc-go/v10/modules/core/04-channel/v2/types"
	"github.com/cosmos/ibc-go/v10/modules/core/api"

	"github.com/disperze/usdc-migration/v10/x/usdcmigration/keeper"
	"github.com/disperze/usdc-migration/v10/x/usdcmigration/types"
)

var (
	_ api.IBCModule             = (*IBCMiddleware)(nil)
	_ api.PacketDataUnmarshaler = (*IBCMiddleware)(nil)
)

// IBCMiddleware must be the outermost layer of the IBC v2 transfer route.
// Receives, acks and timeouts pass through to the wrapped app: Noble is not
// reachable over IBC v2.
type IBCMiddleware struct {
	api.IBCModule
	keeper *keeper.Keeper
}

// NewIBCMiddleware wraps app (the existing IBC v2 transfer stack).
func NewIBCMiddleware(k *keeper.Keeper, app api.IBCModule) IBCMiddleware {
	return IBCMiddleware{IBCModule: app, keeper: k}
}

// OnSendPacket rejects the Noble voucher, whoever the sender is.
func (im IBCMiddleware) OnSendPacket(ctx sdk.Context, sourceClient, destinationClient string, sequence uint64, payload channeltypesv2.Payload, signer sdk.AccAddress) error {
	if payload.SourcePort == types.TransferPortID {
		data, err := transfertypes.UnmarshalPacketData(payload.Value, payload.Version, payload.Encoding)
		if err == nil {
			if err := im.keeper.CheckSend(ctx, data.Token.Denom.Path()); err != nil {
				return err
			}
		}
	}
	return im.IBCModule.OnSendPacket(ctx, sourceClient, destinationClient, sequence, payload, signer)
}

// UnmarshalPacketData delegates to the wrapped app.
func (im IBCMiddleware) UnmarshalPacketData(payload channeltypesv2.Payload) (any, error) {
	unmarshaler, ok := im.IBCModule.(api.PacketDataUnmarshaler)
	if !ok {
		return nil, errorsmod.Wrapf(types.ErrUnsupportedApp, "%T does not implement PacketDataUnmarshaler", im.IBCModule)
	}
	return unmarshaler.UnmarshalPacketData(payload)
}
