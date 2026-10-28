//go:build auditpoc

// Proof-of-concept tests for audit/security-review.md. Tests named
// TestPoC_* assert the safe behaviour, so they FAIL while the finding is
// present. TestMitigation_* pass and document an upstream protection.
//
//	go test -tags auditpoc ./x/usdcmigration/ -run 'TestMiddlewareTestSuite/Test(PoC|Mitigation)' -v
package usdcmigration_test

import (
	sdkmath "cosmossdk.io/math"

	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	distrtypes "github.com/cosmos/cosmos-sdk/x/distribution/types"

	"github.com/cosmos/ibc-go/v10/modules/apps/transfer"
	transfertypes "github.com/cosmos/ibc-go/v10/modules/apps/transfer/types"
	channeltypes "github.com/cosmos/ibc-go/v10/modules/core/04-channel/types"
	host "github.com/cosmos/ibc-go/v10/modules/core/24-host"

	"github.com/disperze/usdc-migration/v10/x/usdcmigration"
	"github.com/disperze/usdc-migration/v10/x/usdcmigration/types"
)

// ambiguousAck sets both members of the Acknowledgement oneof. gogoproto's
// jsonpb fills oneofs by ranging over a Go map, so which member wins changes
// from one decode to the next.
var ambiguousAck = []byte(`{"result":"AQ==","error":"ABCI code: 1: error handling packet: see events for details"}`)

// rawAck lets the test write arbitrary acknowledgement bytes on the Noble chain.
type rawAck []byte

func (a rawAck) Success() bool           { return true }
func (a rawAck) Acknowledgement() []byte { return a }

// I-01: called directly, the middleware decides between "refund event" and
// "pass to transfer" at random for the same packet and ack bytes.
func (s *MiddlewareTestSuite) TestPoC_I01_MiddlewareAckDecodeIsNonDeterministic() {
	_, packet, err := s.returnToNoble(s.seeded, s.multisig, 0)
	s.Require().NoError(err)

	app := s.app(s.chainA)
	im := usdcmigration.NewIBCMiddleware(app.UsdcMigrationKeeper, transfer.NewIBCModule(app.TransferKeeper))

	accepted, rejected := 0, 0
	for range 200 {
		ctx, _ := s.chainA.GetContext().CacheContext()
		if err := im.OnAcknowledgementPacket(ctx, transfertypes.V1, packet, ambiguousAck, s.chainA.SenderAccount.GetAddress()); err != nil {
			rejected++
		} else {
			accepted++
		}
	}
	s.T().Logf("200 runs, same packet, same ack: %d accepted as an error ack, %d rejected by the transfer app", accepted, rejected)
	s.Require().True(accepted == 0 || rejected == 0, "outcome depends on Go map iteration order")
}

// I-01, end to end: core ibc-go (>= v10.1.0, ISA-2025-001) re-marshals the
// ack in AcknowledgePacket and rejects non-canonical bytes before any app
// callback, so the ambiguous ack never reaches the middleware.
func (s *MiddlewareTestSuite) TestMitigation_I01_CoreRejectsAmbiguousAck() {
	nobleIBC := s.app(s.chainB).IBCKeeper.ChannelKeeper

	for range 12 {
		_, packet, err := s.returnToNoble(sdkmath.OneInt(), s.multisig, 0)
		s.Require().NoError(err)

		// Noble receives the packet and commits the ambiguous bytes as its ack.
		s.Require().NoError(s.pathAB.EndpointB.UpdateClient())
		proof, proofHeight := s.pathAB.EndpointA.QueryProof(host.PacketCommitmentKey(packet.SourcePort, packet.SourceChannel, packet.Sequence))
		ctxB := s.chainB.GetContext()
		_, err = nobleIBC.RecvPacket(ctxB, packet, proof, proofHeight)
		s.Require().NoError(err)
		s.Require().NoError(nobleIBC.WriteAcknowledgement(ctxB, packet, rawAck(ambiguousAck)))
		s.coordinator.CommitBlock(s.chainB)

		// A relayer submits it to chain A with a valid proof.
		s.Require().NoError(s.pathAB.EndpointA.UpdateClient())
		_, err = s.pathAB.EndpointA.AcknowledgePacketWithResult(packet, ambiguousAck)
		s.Require().ErrorContains(err, channeltypes.ErrInvalidAcknowledgement.Error())
		s.Require().ErrorContains(err, "acknowledgement marshalling error")
	}
}

// M-01: gov winds the module down (clears noble_channels) while a return is
// in flight. The return's error ack is no longer recognised, and the transfer
// app refunds it by minting unbacked voucher to gov.
func (s *MiddlewareTestSuite) TestPoC_M01_ParamChangeInFlightMintsVoucherToGov() {
	supplyBefore := s.app(s.chainA).BankKeeper.GetSupply(s.chainA.GetContext(), s.voucher).Amount

	// Noble rejects a blocked receiver, so the return gets an error ack.
	blocked := authtypes.NewModuleAddress(distrtypes.ModuleName)
	_, packet, err := s.returnToNoble(s.seeded, blocked, 0)
	s.Require().NoError(err)

	// A param proposal that was already in voting executes before the ack lands.
	s.setParams(func(p *types.Params) { p.NobleChannels = []string{} })

	s.requireErrorAck(s.relay(s.pathAB, packet))

	supplyAfter := s.app(s.chainA).BankKeeper.GetSupply(s.chainA.GetContext(), s.voucher).Amount
	s.T().Logf("voucher supply %s -> %s, gov voucher balance %s", supplyBefore, supplyAfter, s.balance(s.chainA, s.gov, s.voucher))
	s.Require().Equal(supplyBefore, supplyAfter, "the error ack of a return minted voucher")
}
