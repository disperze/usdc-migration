package usdcmigration_test

import (
	"encoding/json"
	"testing"
	"time"

	dbm "github.com/cosmos/cosmos-db"
	"github.com/stretchr/testify/suite"

	"cosmossdk.io/log"
	sdkmath "cosmossdk.io/math"

	simtestutil "github.com/cosmos/cosmos-sdk/testutil/sims"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	banktestutil "github.com/cosmos/cosmos-sdk/x/bank/testutil"
	distrtypes "github.com/cosmos/cosmos-sdk/x/distribution/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"

	transfertypes "github.com/cosmos/ibc-go/v8/modules/apps/transfer/types"
	clienttypes "github.com/cosmos/ibc-go/v8/modules/core/02-client/types"
	channeltypes "github.com/cosmos/ibc-go/v8/modules/core/04-channel/types"
	ibctesting "github.com/cosmos/ibc-go/v8/testing"

	"github.com/giansalex/usdc-migration/v8/testing/simapp"
	"github.com/giansalex/usdc-migration/v8/x/usdcmigration/keeper"
	"github.com/giansalex/usdc-migration/v8/x/usdcmigration/types"
)

const usdc = "uusdc"

func init() {
	ibctesting.DefaultTestingAppInit = func() (ibctesting.TestingApp, map[string]json.RawMessage) {
		app := simapp.NewSimApp(log.NewNopLogger(), dbm.NewMemDB(), nil, true, simtestutil.EmptyAppOptions{})
		return app, app.DefaultGenesis()
	}
	// Test chains use the "cosmos" account prefix, including the one acting as Noble.
	types.NobleBech32Prefix = sdk.GetConfig().GetBech32AccountAddrPrefix()
}

// MiddlewareTestSuite runs three chains: A is the chain with the module
// enabled, B plays Noble (native uusdc) and C is any third chain.
type MiddlewareTestSuite struct {
	suite.Suite

	coordinator *ibctesting.Coordinator
	chainA      *ibctesting.TestChain
	chainB      *ibctesting.TestChain
	chainC      *ibctesting.TestChain

	pathAB *ibctesting.Path // A <-> Noble
	pathAC *ibctesting.Path // A <-> third chain

	gov      sdk.AccAddress
	multisig sdk.AccAddress // receiver on Noble
	voucher  string         // Noble USDC voucher denom on chain A
	seeded   sdkmath.Int    // voucher held by chain A's sender after setup
}

func TestMiddlewareTestSuite(t *testing.T) {
	suite.Run(t, new(MiddlewareTestSuite))
}

func (s *MiddlewareTestSuite) SetupTest() {
	s.coordinator = ibctesting.NewCoordinator(s.T(), 3)
	s.chainA = s.coordinator.GetChain(ibctesting.GetChainID(1))
	s.chainB = s.coordinator.GetChain(ibctesting.GetChainID(2))
	s.chainC = s.coordinator.GetChain(ibctesting.GetChainID(3))

	s.pathAB = ibctesting.NewTransferPath(s.chainA, s.chainB)
	s.coordinator.Setup(s.pathAB)
	s.pathAC = ibctesting.NewTransferPath(s.chainA, s.chainC)
	s.coordinator.Setup(s.pathAC)

	s.gov = authtypes.NewModuleAddress(govtypes.ModuleName)
	s.multisig = sdk.AccAddress([]byte("noble-multisig______"))
	s.voucher = types.VoucherDenom(s.pathAB.EndpointA.ChannelID, usdc)

	// Noble USDC reaches chain A before the module is enabled.
	s.seeded = sdkmath.NewInt(1_000_000)
	s.fundNobleSender(s.seeded)
	s.relay(s.pathAB, s.transfer(s.chainB, s.pathAB.EndpointB.ChannelID, sdk.NewCoin(usdc, s.seeded), s.chainA.SenderAccount.GetAddress()))
	s.Require().Equal(s.seeded, s.balance(s.chainA, s.chainA.SenderAccount.GetAddress(), s.voucher))

	s.setParams(func(p *types.Params) { p.NobleChannels = []string{s.pathAB.EndpointA.ChannelID} })
}

// --- helpers ---

func (s *MiddlewareTestSuite) app(chain *ibctesting.TestChain) *simapp.SimApp {
	return chain.App.(*simapp.SimApp)
}

func (s *MiddlewareTestSuite) keeper() *keeper.Keeper {
	return s.app(s.chainA).UsdcMigrationKeeper
}

func (s *MiddlewareTestSuite) balance(chain *ibctesting.TestChain, addr sdk.AccAddress, denom string) sdkmath.Int {
	return s.app(chain).BankKeeper.GetBalance(chain.GetContext(), addr, denom).Amount
}

func (s *MiddlewareTestSuite) nobleEscrow() sdkmath.Int {
	escrow := transfertypes.GetEscrowAddress(transfertypes.PortID, s.pathAB.EndpointB.ChannelID)
	return s.balance(s.chainB, escrow, usdc)
}

func (s *MiddlewareTestSuite) fundNobleSender(amt sdkmath.Int) {
	ctx := s.chainB.GetContext()
	s.Require().NoError(banktestutil.FundAccount(ctx, s.app(s.chainB).BankKeeper, s.chainB.SenderAccount.GetAddress(), sdk.NewCoins(sdk.NewCoin(usdc, amt))))
	s.coordinator.CommitBlock(s.chainB)
}

func (s *MiddlewareTestSuite) setParams(update func(*types.Params)) {
	p := s.keeper().GetParams(s.chainA.GetContext())
	update(&p)
	s.Require().NoError(s.keeper().Params.Set(s.chainA.GetContext(), p))
	s.coordinator.CommitBlock(s.chainA)
}

func (s *MiddlewareTestSuite) transferMsg(from *ibctesting.TestChain, channel string, coin sdk.Coin, to sdk.AccAddress) *transfertypes.MsgTransfer {
	return transfertypes.NewMsgTransfer(transfertypes.PortID, channel, coin,
		from.SenderAccount.GetAddress().String(), to.String(),
		clienttypes.ZeroHeight(), uint64(from.GetContext().BlockTime().Add(time.Hour).UnixNano()), "")
}

// transfer sends an ICS20 transfer from chain's sender and returns the packet.
func (s *MiddlewareTestSuite) transfer(from *ibctesting.TestChain, channel string, coin sdk.Coin, to sdk.AccAddress) channeltypes.Packet {
	res, err := from.SendMsgs(s.transferMsg(from, channel, coin, to))
	s.Require().NoError(err)
	packet, err := ibctesting.ParsePacketFromEvents(res.Events)
	s.Require().NoError(err)
	return packet
}

func (s *MiddlewareTestSuite) relay(path *ibctesting.Path, packet channeltypes.Packet) []byte {
	_, ack, err := path.RelayPacketWithResults(packet)
	s.Require().NoError(err)
	return ack
}

// returnToNoble executes MsgReturnToNoble on chain A as gov and commits the block.
func (s *MiddlewareTestSuite) returnToNoble(amount sdkmath.Int, receiver sdk.AccAddress, timeoutSeconds uint64) (*types.MsgReturnToNobleResponse, channeltypes.Packet, error) {
	ctx := s.chainA.GetContext()
	res, err := keeper.NewMsgServerImpl(s.keeper()).ReturnToNoble(ctx, &types.MsgReturnToNoble{
		Authority:      s.gov.String(),
		ChannelId:      s.pathAB.EndpointA.ChannelID,
		Receiver:       receiver.String(),
		Amount:         amount,
		TimeoutSeconds: timeoutSeconds,
	})
	if err != nil {
		return nil, channeltypes.Packet{}, err
	}
	packet, perr := ibctesting.ParsePacketFromEvents(ctx.EventManager().ABCIEvents())
	s.Require().NoError(perr)
	s.coordinator.CommitBlock(s.chainA)
	return res, packet, nil
}

func (s *MiddlewareTestSuite) requireErrorAck(ack []byte) {
	var a channeltypes.Acknowledgement
	s.Require().NoError(transfertypes.ModuleCdc.UnmarshalJSON(ack, &a))
	s.Require().False(a.Success(), "expected error acknowledgement, got %s", ack)
}

// --- receive rules ---

func (s *MiddlewareTestSuite) TestRecvNobleUSDCRejected() {
	amt := sdkmath.NewInt(500)
	s.fundNobleSender(amt)
	nobleSender := s.chainB.SenderAccount.GetAddress()
	before := s.balance(s.chainB, nobleSender, usdc)

	packet := s.transfer(s.chainB, s.pathAB.EndpointB.ChannelID, sdk.NewCoin(usdc, amt), s.chainA.SenderAccount.GetAddress())
	s.requireErrorAck(s.relay(s.pathAB, packet))

	s.Require().Equal(s.seeded, s.balance(s.chainA, s.chainA.SenderAccount.GetAddress(), s.voucher), "nothing minted on A")
	s.Require().Equal(before, s.balance(s.chainB, nobleSender, usdc), "Noble sender refunded")
}

func (s *MiddlewareTestSuite) TestRecvOtherNobleDenomAllowed() {
	coin := sdk.NewCoin(sdk.DefaultBondDenom, sdkmath.NewInt(100))
	packet := s.transfer(s.chainB, s.pathAB.EndpointB.ChannelID, coin, s.chainA.SenderAccount.GetAddress())
	ack := s.relay(s.pathAB, packet)

	var a channeltypes.Acknowledgement
	s.Require().NoError(transfertypes.ModuleCdc.UnmarshalJSON(ack, &a))
	s.Require().True(a.Success())
}

func (s *MiddlewareTestSuite) TestRecvAllowedWhenDisabled() {
	s.setParams(func(p *types.Params) { p.RecvBlockEnabled = false })
	amt := sdkmath.NewInt(500)
	s.fundNobleSender(amt)
	s.relay(s.pathAB, s.transfer(s.chainB, s.pathAB.EndpointB.ChannelID, sdk.NewCoin(usdc, amt), s.chainA.SenderAccount.GetAddress()))
	s.Require().Equal(s.seeded.Add(amt), s.balance(s.chainA, s.chainA.SenderAccount.GetAddress(), s.voucher))
}

// --- send rules ---

func (s *MiddlewareTestSuite) TestUserCannotSendVoucherOnAnyChannel() {
	coin := sdk.NewCoin(s.voucher, sdkmath.NewInt(10))
	for _, ch := range []string{s.pathAB.EndpointA.ChannelID, s.pathAC.EndpointA.ChannelID} {
		_, err := s.chainA.SendMsgs(s.transferMsg(s.chainA, ch, coin, s.chainB.SenderAccount.GetAddress()))
		s.Require().ErrorContains(err, types.ErrSendBlocked.Error(), ch)
	}
	s.Require().Equal(s.seeded, s.balance(s.chainA, s.chainA.SenderAccount.GetAddress(), s.voucher))
}

func (s *MiddlewareTestSuite) TestUserCanSendOtherDenoms() {
	coin := sdk.NewCoin(sdk.DefaultBondDenom, sdkmath.NewInt(10))
	s.relay(s.pathAB, s.transfer(s.chainA, s.pathAB.EndpointA.ChannelID, coin, s.chainB.SenderAccount.GetAddress()))
	s.relay(s.pathAC, s.transfer(s.chainA, s.pathAC.EndpointA.ChannelID, coin, s.chainC.SenderAccount.GetAddress()))
}

func (s *MiddlewareTestSuite) TestUserCanSendVoucherWhenDisabled() {
	s.setParams(func(p *types.Params) { p.SendBlockEnabled = false })
	coin := sdk.NewCoin(s.voucher, sdkmath.NewInt(10))
	s.relay(s.pathAB, s.transfer(s.chainA, s.pathAB.EndpointA.ChannelID, coin, s.chainB.SenderAccount.GetAddress()))
}

// --- mint-and-return ---

func (s *MiddlewareTestSuite) TestReturnToNobleDrainsEscrow() {
	s.Require().Equal(s.seeded, s.nobleEscrow())
	user := s.chainA.SenderAccount.GetAddress()

	res, packet, err := s.returnToNoble(sdkmath.ZeroInt(), s.multisig, 0)
	s.Require().NoError(err)
	s.Require().Equal(s.seeded, res.Amount)
	s.Require().Equal(s.seeded, res.Minted)
	s.Require().Equal(s.voucher, res.Denom)

	ack := s.relay(s.pathAB, packet)
	var a channeltypes.Acknowledgement
	s.Require().NoError(transfertypes.ModuleCdc.UnmarshalJSON(ack, &a))
	s.Require().True(a.Success())

	// Noble: escrow released to the multisig
	s.Require().True(s.nobleEscrow().IsZero())
	s.Require().Equal(s.seeded, s.balance(s.chainB, s.multisig, usdc))

	// chain A: user balance untouched, minted voucher burned, cap exhausted
	ctx := s.chainA.GetContext()
	s.Require().Equal(s.seeded, s.balance(s.chainA, user, s.voucher))
	s.Require().Equal(s.seeded, s.app(s.chainA).BankKeeper.GetSupply(ctx, s.voucher).Amount)
	s.Require().True(s.balance(s.chainA, s.gov, s.voucher).IsZero())
	s.Require().Equal(s.seeded, s.keeper().GetReturnedTotal(ctx, s.pathAB.EndpointA.ChannelID))

	_, _, err = s.returnToNoble(sdkmath.ZeroInt(), s.multisig, 0)
	s.Require().ErrorIs(err, types.ErrNothingToReturn)
}

func (s *MiddlewareTestSuite) TestReturnToNoblePartialAndCap() {
	half := s.seeded.QuoRaw(2)
	_, packet, err := s.returnToNoble(half, s.multisig, 0)
	s.Require().NoError(err)
	s.relay(s.pathAB, packet)

	_, _, err = s.returnToNoble(half.AddRaw(1), s.multisig, 0)
	s.Require().ErrorIs(err, types.ErrExceedsReturnCap)

	_, packet, err = s.returnToNoble(sdkmath.ZeroInt(), s.multisig, 0)
	s.Require().NoError(err)
	s.relay(s.pathAB, packet)

	s.Require().True(s.nobleEscrow().IsZero())
	s.Require().Equal(s.seeded, s.balance(s.chainB, s.multisig, usdc))
}

func (s *MiddlewareTestSuite) TestReturnToNobleErrorAckRefundsAndRetries() {
	// Noble rejects a blocked receiver -> error ack -> voucher refunded to gov.
	blocked := authtypes.NewModuleAddress(distrtypes.ModuleName)
	_, packet, err := s.returnToNoble(sdkmath.ZeroInt(), blocked, 0)
	s.Require().NoError(err)
	s.requireErrorAck(s.relay(s.pathAB, packet))

	ctx := s.chainA.GetContext()
	s.Require().Equal(s.seeded, s.balance(s.chainA, s.gov, s.voucher), "refund minted back to gov")
	s.Require().True(s.keeper().GetReturnedTotal(ctx, s.pathAB.EndpointA.ChannelID).IsZero())
	s.Require().Equal(s.seeded, s.nobleEscrow())

	// Retry reuses the refunded balance: nothing new is minted.
	res, packet, err := s.returnToNoble(sdkmath.ZeroInt(), s.multisig, 0)
	s.Require().NoError(err)
	s.Require().True(res.Minted.IsZero())
	s.Require().Equal(s.seeded, res.Amount)
	s.relay(s.pathAB, packet)

	s.Require().True(s.nobleEscrow().IsZero())
	s.Require().Equal(s.seeded, s.balance(s.chainB, s.multisig, usdc))
	s.Require().Equal(s.seeded, s.app(s.chainA).BankKeeper.GetSupply(s.chainA.GetContext(), s.voucher).Amount)
}

func (s *MiddlewareTestSuite) TestReturnToNobleTimeoutRefunds() {
	_, packet, err := s.returnToNoble(sdkmath.ZeroInt(), s.multisig, 1)
	s.Require().NoError(err)

	s.coordinator.IncrementTimeBy(time.Minute)
	s.coordinator.CommitBlock(s.chainB)
	s.Require().NoError(s.pathAB.EndpointA.UpdateClient())
	s.Require().NoError(s.pathAB.EndpointA.TimeoutPacket(packet))

	ctx := s.chainA.GetContext()
	s.Require().Equal(s.seeded, s.balance(s.chainA, s.gov, s.voucher))
	s.Require().True(s.keeper().GetReturnedTotal(ctx, s.pathAB.EndpointA.ChannelID).IsZero())
	s.Require().Equal(s.seeded, s.nobleEscrow())
}

func (s *MiddlewareTestSuite) TestReturnToNobleRejections() {
	srv := keeper.NewMsgServerImpl(s.keeper())
	ctx := s.chainA.GetContext()
	valid := types.MsgReturnToNoble{
		Authority: s.gov.String(),
		ChannelId: s.pathAB.EndpointA.ChannelID,
		Receiver:  s.multisig.String(),
		Amount:    sdkmath.ZeroInt(),
	}

	msg := valid
	msg.Authority = s.chainA.SenderAccount.GetAddress().String()
	_, err := srv.ReturnToNoble(ctx, &msg)
	s.Require().ErrorIs(err, types.ErrInvalidAuthority)

	msg = valid
	msg.ChannelId = s.pathAC.EndpointA.ChannelID
	_, err = srv.ReturnToNoble(ctx, &msg)
	s.Require().ErrorIs(err, types.ErrUnknownChannel)

	msg = valid
	msg.Amount = s.seeded.AddRaw(1)
	_, err = srv.ReturnToNoble(ctx, &msg)
	s.Require().ErrorIs(err, types.ErrExceedsReturnCap)

	msg = valid
	msg.Receiver = "noble1invalid"
	_, err = srv.ReturnToNoble(ctx, &msg)
	s.Require().ErrorIs(err, types.ErrInvalidReceiver)
}

func (s *MiddlewareTestSuite) TestQueryReturnCap() {
	q := keeper.NewQueryServerImpl(s.keeper())
	res, err := q.ReturnCap(s.chainA.GetContext(), &types.QueryReturnCapRequest{ChannelId: s.pathAB.EndpointA.ChannelID})
	s.Require().NoError(err)
	s.Require().Equal(s.voucher, res.Denom)
	s.Require().Equal(s.seeded, res.Supply)
	s.Require().Equal(s.seeded, res.Cap)

	_, err = q.ReturnCap(s.chainA.GetContext(), &types.QueryReturnCapRequest{ChannelId: s.pathAC.EndpointA.ChannelID})
	s.Require().Error(err)
}

func (s *MiddlewareTestSuite) TestGenesisRoundTrip() {
	_, packet, err := s.returnToNoble(sdkmath.ZeroInt(), s.multisig, 0)
	s.Require().NoError(err)
	s.relay(s.pathAB, packet)

	ctx := s.chainA.GetContext()
	gs, err := s.keeper().ExportGenesis(ctx)
	s.Require().NoError(err)
	s.Require().NoError(gs.Validate())
	s.Require().Equal([]string{s.pathAB.EndpointA.ChannelID}, gs.Params.NobleChannels)
	s.Require().Len(gs.ReturnedTotals, 1)
	s.Require().Equal(s.seeded, gs.ReturnedTotals[0].Amount)
}
