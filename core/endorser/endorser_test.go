/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package endorser_test

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/golang/protobuf/proto"
	mc "github.com/sinochem-tech/fabric/common/mocks/config"
	"github.com/sinochem-tech/fabric/common/mocks/resourcesconfig"
	"github.com/sinochem-tech/fabric/common/util"
	"github.com/sinochem-tech/fabric/core/common/ccprovider"
	"github.com/sinochem-tech/fabric/core/endorser"
	"github.com/sinochem-tech/fabric/core/endorser/mocks"
	"github.com/sinochem-tech/fabric/core/handlers/endorsement/builtin"
	"github.com/sinochem-tech/fabric/core/ledger"
	mockccprovider "github.com/sinochem-tech/fabric/core/mocks/ccprovider"
	em "github.com/sinochem-tech/fabric/core/mocks/endorser"
	"github.com/sinochem-tech/fabric/msp"
	"github.com/sinochem-tech/fabric/msp/mgmt"
	"github.com/sinochem-tech/fabric/msp/mgmt/testtools"
	"github.com/sinochem-tech/fabric/protos/common"
	"github.com/sinochem-tech/fabric/protos/ledger/rwset"
	pb "github.com/sinochem-tech/fabric/protos/peer"
	"github.com/sinochem-tech/fabric/protos/transientstore"
	"github.com/sinochem-tech/fabric/protos/utils"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func pvtEmptyDistributor(_ string, _ string, _ *transientstore.TxPvtReadWriteSetWithConfigInfo, _ uint64) error {
	return nil
}

func getSignedPropWithCHID(ccid, ccver, chid string, t *testing.T) *pb.SignedProposal {
	ccargs := [][]byte{[]byte("args")}

	return getSignedPropWithCHIdAndArgs(chid, ccid, ccver, ccargs, t)
}

func getSignedProp(ccid, ccver string, t *testing.T) *pb.SignedProposal {
	return getSignedPropWithCHID(ccid, ccver, util.GetTestChainID(), t)
}

func getSignedPropWithCHIdAndArgs(chid, ccid, ccver string, ccargs [][]byte, t *testing.T) *pb.SignedProposal {
	spec := &pb.ChaincodeSpec{Type: 1, ChaincodeId: &pb.ChaincodeID{Name: ccid, Version: ccver}, Input: &pb.ChaincodeInput{Args: ccargs}}

	cis := &pb.ChaincodeInvocationSpec{ChaincodeSpec: spec}

	creator, err := signer.Serialize()
	assert.NoError(t, err)
	prop, _, err := utils.CreateChaincodeProposal(common.HeaderType_ENDORSER_TRANSACTION, chid, cis, creator)
	assert.NoError(t, err)
	propBytes, err := utils.GetBytesProposal(prop)
	assert.NoError(t, err)
	signature, err := signer.Sign(propBytes)
	assert.NoError(t, err)
	return &pb.SignedProposal{ProposalBytes: propBytes, Signature: signature}
}

func newMockTxSim() *mockccprovider.MockTxSim {
	return &mockccprovider.MockTxSim{
		GetTxSimulationResultsRv: &ledger.TxSimulationResults{
			PubSimulationResults: &rwset.TxReadWriteSet{},
		},
	}
}

func TestEndorserNilProp(t *testing.T) {
	es := endorser.NewEndorserServer(pvtEmptyDistributor, &em.MockSupport{
		GetApplicationConfigBoolRv: true,
		GetApplicationConfigRv:     &mc.MockApplication{CapabilitiesRv: &mc.MockApplicationCapabilities{}},
		GetTransactionByIDErr:      errors.New(""),
		ChaincodeDefinitionRv:      &ccprovider.ChaincodeData{Escc: "ESCC"},
		ExecuteResp:                &pb.Response{Status: 200, Payload: utils.MarshalOrPanic(&pb.ProposalResponse{Response: &pb.Response{}})},
		GetTxSimulatorRv: &mockccprovider.MockTxSim{
			GetTxSimulationResultsRv: &ledger.TxSimulationResults{
				PubSimulationResults: &rwset.TxReadWriteSet{},
			},
		},
	})

	pResp, err := es.ProcessProposal(context.Background(), nil)
	assert.Error(t, err)
	assert.EqualValues(t, 500, pResp.Response.Status)
	assert.Equal(t, "nil arguments", pResp.Response.Message)
}

func TestEndorserUninvokableSysCC(t *testing.T) {
	es := endorser.NewEndorserServer(pvtEmptyDistributor, &em.MockSupport{
		GetApplicationConfigBoolRv:       true,
		GetApplicationConfigRv:           &mc.MockApplication{CapabilitiesRv: &mc.MockApplicationCapabilities{}},
		GetTransactionByIDErr:            errors.New(""),
		IsSysCCAndNotInvokableExternalRv: true,
	})

	signedProp := getSignedProp("ccid", "0", t)

	pResp, err := es.ProcessProposal(context.Background(), signedProp)
	assert.Error(t, err)
	assert.EqualValues(t, 500, pResp.Response.Status)
	assert.Equal(t, "chaincode ccid cannot be invoked through a proposal", pResp.Response.Message)
}

func TestEndorserCCInvocationFailed(t *testing.T) {
	es := endorser.NewEndorserServer(pvtEmptyDistributor, &em.MockSupport{
		GetApplicationConfigBoolRv: true,
		GetApplicationConfigRv:     &mc.MockApplication{CapabilitiesRv: &mc.MockApplicationCapabilities{}},
		GetTransactionByIDErr:      errors.New(""),
		ChaincodeDefinitionRv:      &ccprovider.ChaincodeData{Escc: "ESCC"},
		ExecuteResp:                &pb.Response{Status: 1000, Payload: utils.MarshalOrPanic(&pb.ProposalResponse{Response: &pb.Response{}}), Message: "Chaincode Error"},
		GetTxSimulatorRv: &mockccprovider.MockTxSim{
			GetTxSimulationResultsRv: &ledger.TxSimulationResults{
				PubSimulationResults: &rwset.TxReadWriteSet{},
			},
		},
	})

	signedProp := getSignedProp("ccid", "0", t)

	pResp, err := es.ProcessProposal(context.Background(), signedProp)
	assert.NoError(t, err)
	assert.EqualValues(t, 1000, pResp.Response.Status)
	assert.Regexp(t, "Chaincode Error", pResp.Response.Message)
}

func TestEndorserNoCCDef(t *testing.T) {
	es := endorser.NewEndorserServer(pvtEmptyDistributor, &em.MockSupport{
		GetApplicationConfigBoolRv: true,
		GetApplicationConfigRv:     &mc.MockApplication{CapabilitiesRv: &mc.MockApplicationCapabilities{}},
		GetTransactionByIDErr:      errors.New(""),
		ChaincodeDefinitionError:   errors.New(""),
		ExecuteResp:                &pb.Response{Status: 200, Payload: utils.MarshalOrPanic(&pb.ProposalResponse{Response: &pb.Response{}})},
		GetTxSimulatorRv: &mockccprovider.MockTxSim{
			GetTxSimulationResultsRv: &ledger.TxSimulationResults{
				PubSimulationResults: &rwset.TxReadWriteSet{},
			},
		},
	})

	signedProp := getSignedProp("ccid", "0", t)

	pResp, err := es.ProcessProposal(context.Background(), signedProp)
	assert.NoError(t, err)
	assert.EqualValues(t, 500, pResp.Response.Status)
	assert.Regexp(t, "make sure the chaincode", pResp.Response.Message)
}

func TestEndorserBadInstPolicy(t *testing.T) {
	es := endorser.NewEndorserServer(pvtEmptyDistributor, &em.MockSupport{
		GetApplicationConfigBoolRv:    true,
		GetApplicationConfigRv:        &mc.MockApplication{CapabilitiesRv: &mc.MockApplicationCapabilities{}},
		GetTransactionByIDErr:         errors.New(""),
		CheckInstantiationPolicyError: errors.New(""),
		ChaincodeDefinitionRv:         &ccprovider.ChaincodeData{Escc: "ESCC"},
		ExecuteResp:                   &pb.Response{Status: 200, Payload: utils.MarshalOrPanic(&pb.ProposalResponse{Response: &pb.Response{}})},
		GetTxSimulatorRv: &mockccprovider.MockTxSim{
			GetTxSimulationResultsRv: &ledger.TxSimulationResults{
				PubSimulationResults: &rwset.TxReadWriteSet{},
			},
		},
	})

	signedProp := getSignedProp("ccid", "0", t)

	pResp, err := es.ProcessProposal(context.Background(), signedProp)
	assert.NoError(t, err)
	assert.EqualValues(t, 500, pResp.Response.Status)
}

func TestEndorserSysCC(t *testing.T) {
	m := &mock.Mock{}
	m.On("Sign", mock.Anything).Return([]byte{1, 2, 3, 4, 5}, nil)
	m.On("Serialize").Return([]byte{1, 1, 1}, nil)
	m.On("GetTxSimulator", mock.Anything, mock.Anything).Return(newMockTxSim(), nil)
	support := &em.MockSupport{
		Mock: m,
		GetApplicationConfigBoolRv: true,
		GetApplicationConfigRv:     &mc.MockApplication{CapabilitiesRv: &mc.MockApplicationCapabilities{}},
		GetTransactionByIDErr:      errors.New(""),
		IsSysCCRv:                  true,
		ChaincodeDefinitionRv:      &ccprovider.ChaincodeData{Escc: "ESCC"},
		ExecuteResp:                &pb.Response{Status: 200, Payload: utils.MarshalOrPanic(&pb.ProposalResponse{Response: &pb.Response{}})},
	}
	attachPluginEndorser(support)
	es := endorser.NewEndorserServer(pvtEmptyDistributor, support)

	signedProp := getSignedProp("ccid", "0", t)

	pResp, err := es.ProcessProposal(context.Background(), signedProp)
	assert.NoError(t, err)
	assert.EqualValues(t, 200, pResp.Response.Status)
}

func TestEndorserCCInvocationError(t *testing.T) {
	es := endorser.NewEndorserServer(pvtEmptyDistributor, &em.MockSupport{
		GetApplicationConfigBoolRv: true,
		GetApplicationConfigRv:     &mc.MockApplication{CapabilitiesRv: &mc.MockApplicationCapabilities{}},
		GetTransactionByIDErr:      errors.New(""),
		ExecuteError:               errors.New(""),
		ChaincodeDefinitionRv:      &ccprovider.ChaincodeData{Escc: "ESCC"},
		GetTxSimulatorRv: &mockccprovider.MockTxSim{
			GetTxSimulationResultsRv: &ledger.TxSimulationResults{
				PubSimulationResults: &rwset.TxReadWriteSet{},
			},
		},
	})

	signedProp := getSignedProp("ccid", "0", t)

	pResp, err := es.ProcessProposal(context.Background(), signedProp)
	assert.NoError(t, err)
	assert.EqualValues(t, 500, pResp.Response.Status)
}

func TestEndorserLSCCBadType(t *testing.T) {
	es := endorser.NewEndorserServer(pvtEmptyDistributor, &em.MockSupport{
		GetApplicationConfigBoolRv: true,
		GetApplicationConfigRv:     &mc.MockApplication{CapabilitiesRv: &mc.MockApplicationCapabilities{}},
		GetTransactionByIDErr:      errors.New(""),
		ChaincodeDefinitionRv:      &ccprovider.ChaincodeData{Escc: "ESCC"},
		ExecuteResp:                &pb.Response{Status: 200, Payload: utils.MarshalOrPanic(&pb.ProposalResponse{Response: &pb.Response{}})},
		GetTxSimulatorRv: &mockccprovider.MockTxSim{
			GetTxSimulationResultsRv: &ledger.TxSimulationResults{
				PubSimulationResults: &rwset.TxReadWriteSet{},
			},
		},
	})

	cds := utils.MarshalOrPanic(
		&pb.ChaincodeDeploymentSpec{
			ChaincodeSpec: &pb.ChaincodeSpec{
				ChaincodeId: &pb.ChaincodeID{Name: "barf"},
				Type:        pb.ChaincodeSpec_UNDEFINED,
			},
		},
	)
	signedProp := getSignedPropWithCHIdAndArgs(util.GetTestChainID(), "lscc", "0", [][]byte{[]byte("deploy"), []byte("a"), cds}, t)

	pResp, err := es.ProcessProposal(context.Background(), signedProp)
	assert.NoError(t, err)
	assert.EqualValues(t, 500, pResp.Response.Status)
	assert.Equal(t, "Unknown chaincodeType: UNDEFINED", pResp.Response.Message)
}

func TestEndorserDupTXId(t *testing.T) {
	es := endorser.NewEndorserServer(pvtEmptyDistributor, &em.MockSupport{
		GetApplicationConfigBoolRv: true,
		GetApplicationConfigRv:     &mc.MockApplication{CapabilitiesRv: &mc.MockApplicationCapabilities{}},
		ChaincodeDefinitionRv:      &ccprovider.ChaincodeData{Escc: "ESCC"},
		ExecuteResp:                &pb.Response{Status: 200, Payload: utils.MarshalOrPanic(&pb.ProposalResponse{Response: &pb.Response{}})},
		GetTxSimulatorRv: &mockccprovider.MockTxSim{
			GetTxSimulationResultsRv: &ledger.TxSimulationResults{
				PubSimulationResults: &rwset.TxReadWriteSet{},
			},
		},
	})

	signedProp := getSignedProp("ccid", "0", t)

	pResp, err := es.ProcessProposal(context.Background(), signedProp)
	assert.Error(t, err)
	assert.EqualValues(t, 500, pResp.Response.Status)
	assert.Regexp(t, "duplicate transaction found", pResp.Response.Message)
}

func TestEndorserBadACL(t *testing.T) {
	es := endorser.NewEndorserServer(pvtEmptyDistributor, &em.MockSupport{
		GetApplicationConfigBoolRv: true,
		GetApplicationConfigRv:     &mc.MockApplication{CapabilitiesRv: &mc.MockApplicationCapabilities{}},
		CheckACLErr:                errors.New(""),
		GetTransactionByIDErr:      errors.New(""),
		ChaincodeDefinitionRv:      &ccprovider.ChaincodeData{Escc: "ESCC"},
		ExecuteResp:                &pb.Response{Status: 200, Payload: utils.MarshalOrPanic(&pb.ProposalResponse{Response: &pb.Response{}})},
		GetTxSimulatorRv: &mockccprovider.MockTxSim{
			GetTxSimulationResultsRv: &ledger.TxSimulationResults{
				PubSimulationResults: &rwset.TxReadWriteSet{},
			},
		},
	})

	signedProp := getSignedProp("ccid", "0", t)

	pResp, err := es.ProcessProposal(context.Background(), signedProp)
	assert.Error(t, err)
	assert.EqualValues(t, 500, pResp.Response.Status)
}

func TestEndorserGoodPathEmptyChannel(t *testing.T) {
	es := endorser.NewEndorserServer(pvtEmptyDistributor, &em.MockSupport{
		GetApplicationConfigBoolRv: true,
		GetApplicationConfigRv:     &mc.MockApplication{CapabilitiesRv: &mc.MockApplicationCapabilities{}},
		GetTransactionByIDErr:      errors.New(""),
		ChaincodeDefinitionRv:      &ccprovider.ChaincodeData{Escc: "ESCC"},
		ExecuteResp:                &pb.Response{Status: 200, Payload: utils.MarshalOrPanic(&pb.ProposalResponse{Response: &pb.Response{}})},
		GetTxSimulatorRv: &mockccprovider.MockTxSim{
			GetTxSimulationResultsRv: &ledger.TxSimulationResults{
				PubSimulationResults: &rwset.TxReadWriteSet{},
			},
		},
	})

	signedProp := getSignedPropWithCHIdAndArgs("", "ccid", "0", [][]byte{[]byte("args")}, t)

	pResp, err := es.ProcessProposal(context.Background(), signedProp)
	assert.NoError(t, err)
	assert.EqualValues(t, 200, pResp.Response.Status)
}

func TestEndorserLSCCInitFails(t *testing.T) {
	es := endorser.NewEndorserServer(pvtEmptyDistributor, &em.MockSupport{
		GetApplicationConfigBoolRv: true,
		GetApplicationConfigRv:     &mc.MockApplication{CapabilitiesRv: &mc.MockApplicationCapabilities{}},
		GetTransactionByIDErr:      errors.New(""),
		ChaincodeDefinitionRv:      &ccprovider.ChaincodeData{Escc: "ESCC"},
		ExecuteResp:                &pb.Response{Status: 200, Payload: utils.MarshalOrPanic(&pb.ProposalResponse{Response: &pb.Response{}})},
		GetTxSimulatorRv: &mockccprovider.MockTxSim{
			GetTxSimulationResultsRv: &ledger.TxSimulationResults{
				PubSimulationResults: &rwset.TxReadWriteSet{},
			},
		},
		ExecuteCDSError: errors.New(""),
	})

	cds := utils.MarshalOrPanic(
		&pb.ChaincodeDeploymentSpec{
			ChaincodeSpec: &pb.ChaincodeSpec{
				ChaincodeId: &pb.ChaincodeID{Name: "barf"},
				Type:        pb.ChaincodeSpec_GOLANG,
			},
		},
	)
	signedProp := getSignedPropWithCHIdAndArgs(util.GetTestChainID(), "lscc", "0", [][]byte{[]byte("deploy"), []byte("a"), cds}, t)

	pResp, err := es.ProcessProposal(context.Background(), signedProp)
	assert.NoError(t, err)
	assert.EqualValues(t, 500, pResp.Response.Status)
}

func TestEndorserLSCCDeploySysCC(t *testing.T) {
	SysCCMap := make(map[string]struct{})
	deployedCCName := "barf"
	SysCCMap[deployedCCName] = struct{}{}
	es := endorser.NewEndorserServer(pvtEmptyDistributor, &em.MockSupport{
		GetApplicationConfigBoolRv: true,
		GetApplicationConfigRv:     &mc.MockApplication{CapabilitiesRv: &mc.MockApplicationCapabilities{}},
		GetTransactionByIDErr:      errors.New(""),
		ChaincodeDefinitionRv:      &ccprovider.ChaincodeData{Escc: "ESCC"},
		ExecuteResp:                &pb.Response{Status: 200, Payload: utils.MarshalOrPanic(&pb.ProposalResponse{Response: &pb.Response{}})},
		GetTxSimulatorRv: &mockccprovider.MockTxSim{
			GetTxSimulationResultsRv: &ledger.TxSimulationResults{
				PubSimulationResults: &rwset.TxReadWriteSet{},
			},
		},
		SysCCMap: SysCCMap,
	})

	cds := utils.MarshalOrPanic(
		&pb.ChaincodeDeploymentSpec{
			ChaincodeSpec: &pb.ChaincodeSpec{
				ChaincodeId: &pb.ChaincodeID{Name: deployedCCName},
				Type:        pb.ChaincodeSpec_GOLANG,
			},
		},
	)
	signedProp := getSignedPropWithCHIdAndArgs(util.GetTestChainID(), "lscc", "0", [][]byte{[]byte("deploy"), []byte("a"), cds}, t)

	pResp, err := es.ProcessProposal(context.Background(), signedProp)
	assert.NoError(t, err)
	assert.EqualValues(t, 500, pResp.Response.Status)
	assert.Equal(t, "attempting to deploy a system chaincode barf/testchainid", pResp.Response.Message)
}

func TestEndorserLSCCJava1(t *testing.T) {
	if endorser.JavaEnabled() {
		t.Skip("Java chaincode is supported")
	}

	es := endorser.NewEndorserServer(pvtEmptyDistributor, &em.MockSupport{
		GetApplicationConfigBoolRv: true,
		GetApplicationConfigRv:     &mc.MockApplication{CapabilitiesRv: &mc.MockApplicationCapabilities{}},
		IsJavaRV:                   true,
		GetTransactionByIDErr:      errors.New(""),
		ChaincodeDefinitionRv:      &ccprovider.ChaincodeData{Escc: "ESCC"},
		ExecuteResp:                &pb.Response{Status: 200, Payload: utils.MarshalOrPanic(&pb.ProposalResponse{Response: &pb.Response{}})},
		GetTxSimulatorRv: &mockccprovider.MockTxSim{
			GetTxSimulationResultsRv: &ledger.TxSimulationResults{
				PubSimulationResults: &rwset.TxReadWriteSet{},
			},
		},
	})

	cds := utils.MarshalOrPanic(
		&pb.ChaincodeDeploymentSpec{
			ChaincodeSpec: &pb.ChaincodeSpec{
				ChaincodeId: &pb.ChaincodeID{Name: "barf"},
				Type:        pb.ChaincodeSpec_JAVA,
			},
		},
	)
	signedProp := getSignedPropWithCHIdAndArgs(util.GetTestChainID(), "lscc", "0", [][]byte{[]byte("deploy"), []byte("a"), cds}, t)

	pResp, err := es.ProcessProposal(context.Background(), signedProp)
	assert.NoError(t, err)
	assert.EqualValues(t, 500, pResp.Response.Status)
	assert.Equal(t, "Java chaincode is work-in-progress and disabled", pResp.Response.Message)
}

func TestEndorserLSCCJava2(t *testing.T) {
	if endorser.JavaEnabled() {
		t.Skip("Java chaincode is supported")
	}

	es := endorser.NewEndorserServer(pvtEmptyDistributor, &em.MockSupport{
		GetApplicationConfigBoolRv: true,
		GetApplicationConfigRv:     &mc.MockApplication{CapabilitiesRv: &mc.MockApplicationCapabilities{}},
		IsJavaErr:                  errors.New(""),
		GetTransactionByIDErr:      errors.New(""),
		ChaincodeDefinitionRv:      &ccprovider.ChaincodeData{Escc: "ESCC"},
		ExecuteResp:                &pb.Response{Status: 200, Payload: utils.MarshalOrPanic(&pb.ProposalResponse{Response: &pb.Response{}})},
		GetTxSimulatorRv: &mockccprovider.MockTxSim{
			GetTxSimulationResultsRv: &ledger.TxSimulationResults{
				PubSimulationResults: &rwset.TxReadWriteSet{},
			},
		},
	})

	cds := utils.MarshalOrPanic(
		&pb.ChaincodeDeploymentSpec{
			ChaincodeSpec: &pb.ChaincodeSpec{
				ChaincodeId: &pb.ChaincodeID{Name: "barf"},
				Type:        pb.ChaincodeSpec_JAVA,
			},
		},
	)
	signedProp := getSignedPropWithCHIdAndArgs(util.GetTestChainID(), "lscc", "0", [][]byte{[]byte("deploy"), []byte("a"), cds}, t)

	pResp, err := es.ProcessProposal(context.Background(), signedProp)
	assert.NoError(t, err)
	assert.EqualValues(t, 500, pResp.Response.Status)
}

func TestEndorserGoodPathWEvents(t *testing.T) {
	m := &mock.Mock{}
	m.On("Sign", mock.Anything).Return([]byte{1, 2, 3, 4, 5}, nil)
	m.On("Serialize").Return([]byte{1, 1, 1}, nil)
	m.On("GetTxSimulator", mock.Anything, mock.Anything).Return(newMockTxSim(), nil)
	support := &em.MockSupport{
		Mock: m,
		GetApplicationConfigBoolRv: true,
		GetApplicationConfigRv:     &mc.MockApplication{CapabilitiesRv: &mc.MockApplicationCapabilities{}},
		GetTransactionByIDErr:      errors.New(""),
		ChaincodeDefinitionRv:      &ccprovider.ChaincodeData{Escc: "ESCC"},
		ExecuteResp:                &pb.Response{Status: 200, Payload: utils.MarshalOrPanic(&pb.ProposalResponse{Response: &pb.Response{}})},
		ExecuteEvent:               &pb.ChaincodeEvent{},
	}
	attachPluginEndorser(support)
	es := endorser.NewEndorserServer(pvtEmptyDistributor, support)

	signedProp := getSignedProp("ccid", "0", t)

	pResp, err := es.ProcessProposal(context.Background(), signedProp)
	assert.NoError(t, err)
	assert.EqualValues(t, 200, pResp.Response.Status)
}

func TestEndorserBadChannel(t *testing.T) {
	es := endorser.NewEndorserServer(pvtEmptyDistributor, &em.MockSupport{
		GetApplicationConfigBoolRv: true,
		GetApplicationConfigRv:     &mc.MockApplication{CapabilitiesRv: &mc.MockApplicationCapabilities{}},
		GetTransactionByIDErr:      errors.New(""),
		ChaincodeDefinitionRv:      &ccprovider.ChaincodeData{Escc: "ESCC"},
		ExecuteResp:                &pb.Response{Status: 200, Payload: utils.MarshalOrPanic(&pb.ProposalResponse{Response: &pb.Response{}})},
		GetTxSimulatorRv: &mockccprovider.MockTxSim{
			GetTxSimulationResultsRv: &ledger.TxSimulationResults{
				PubSimulationResults: &rwset.TxReadWriteSet{},
			},
		},
	})

	signedProp := getSignedPropWithCHID("ccid", "0", "barfchain", t)

	pResp, err := es.ProcessProposal(context.Background(), signedProp)
	assert.Error(t, err)
	assert.EqualValues(t, 500, pResp.Response.Status)
	assert.Equal(t, "access denied: channel [barfchain] creator org [SampleOrg]", pResp.Response.Message)
}

func TestEndorserGoodPath(t *testing.T) {
	m := &mock.Mock{}
	m.On("Sign", mock.Anything).Return([]byte{1, 2, 3, 4, 5}, nil)
	m.On("Serialize").Return([]byte{1, 1, 1}, nil)
	m.On("GetTxSimulator", mock.Anything, mock.Anything).Return(newMockTxSim(), nil)
	support := &em.MockSupport{
		Mock: m,
		GetApplicationConfigBoolRv: true,
		GetApplicationConfigRv:     &mc.MockApplication{CapabilitiesRv: &mc.MockApplicationCapabilities{}},
		GetTransactionByIDErr:      errors.New(""),
		ChaincodeDefinitionRv:      &ccprovider.ChaincodeData{Escc: "ESCC"},
		ExecuteResp:                &pb.Response{Status: 200, Payload: utils.MarshalOrPanic(&pb.ProposalResponse{Response: &pb.Response{}})},
	}
	attachPluginEndorser(support)
	es := endorser.NewEndorserServer(pvtEmptyDistributor, support)

	signedProp := getSignedProp("ccid", "0", t)

	pResp, err := es.ProcessProposal(context.Background(), signedProp)
	assert.NoError(t, err)
	assert.EqualValues(t, 200, pResp.Response.Status)
}

func TestEndorserLSCC(t *testing.T) {
	m := &mock.Mock{}
	m.On("Sign", mock.Anything).Return([]byte{1, 2, 3, 4, 5}, nil)
	m.On("Serialize").Return([]byte{1, 1, 1}, nil)
	m.On("GetTxSimulator", mock.Anything, mock.Anything).Return(newMockTxSim(), nil)
	support := &em.MockSupport{
		Mock: m,
		GetApplicationConfigBoolRv: true,
		GetApplicationConfigRv:     &mc.MockApplication{CapabilitiesRv: &mc.MockApplicationCapabilities{}},
		GetTransactionByIDErr:      errors.New(""),
		ChaincodeDefinitionRv:      &ccprovider.ChaincodeData{Escc: "ESCC"},
		ExecuteResp:                &pb.Response{Status: 200, Payload: utils.MarshalOrPanic(&pb.ProposalResponse{Response: &pb.Response{}})},
	}
	attachPluginEndorser(support)
	es := endorser.NewEndorserServer(pvtEmptyDistributor, support)

	cds := utils.MarshalOrPanic(
		&pb.ChaincodeDeploymentSpec{
			ChaincodeSpec: &pb.ChaincodeSpec{
				ChaincodeId: &pb.ChaincodeID{Name: "barf"},
				Type:        pb.ChaincodeSpec_GOLANG,
			},
		},
	)
	signedProp := getSignedPropWithCHIdAndArgs(util.GetTestChainID(), "lscc", "0", [][]byte{[]byte("deploy"), []byte("a"), cds}, t)

	pResp, err := es.ProcessProposal(context.Background(), signedProp)
	assert.NoError(t, err)
	assert.EqualValues(t, 200, pResp.Response.Status)
}

func attachPluginEndorser(support *em.MockSupport) {
	csr := &mocks.ChannelStateRetriever{}
	queryCreator := &mocks.QueryCreator{}
	csr.On("NewQueryCreator", mock.Anything).Return(queryCreator, nil)
	sif := &mocks.SigningIdentityFetcher{}
	sif.On("SigningIdentityForRequest", mock.Anything).Return(support, nil)
	pm := &mocks.PluginMapper{}
	pm.On("PluginFactoryByName", mock.Anything).Return(&builtin.DefaultEndorsementFactory{})
	support.PluginEndorser = endorser.NewPluginEndorser(&endorser.PluginSupport{
		ChannelStateRetriever:   csr,
		SigningIdentityFetcher:  sif,
		PluginMapper:            pm,
		TransientStoreRetriever: mockTransientStoreRetriever,
	})
}

func TestEndorseWithPlugin(t *testing.T) {
	m := &mock.Mock{}
	m.On("Sign", mock.Anything).Return([]byte{1, 2, 3, 4, 5}, nil)
	m.On("Serialize").Return([]byte{1, 1, 1}, nil)
	m.On("GetTxSimulator", mock.Anything, mock.Anything).Return(newMockTxSim(), nil)
	support := &em.MockSupport{
		Mock: m,
		GetApplicationConfigBoolRv: true,
		GetApplicationConfigRv:     &mc.MockApplication{CapabilitiesRv: &mc.MockApplicationCapabilities{}},
		GetTransactionByIDErr:      errors.New("can't find this transaction in the index"),
		ChaincodeDefinitionRv:      &resourceconfig.MockChaincodeDefinition{EndorsementStr: "ESCC"},
		ExecuteResp:                &pb.Response{Status: 200, Payload: []byte{1}},
	}
	attachPluginEndorser(support)

	es := endorser.NewEndorserServer(pvtEmptyDistributor, support)

	signedProp := getSignedProp("ccid", "0", t)

	resp, err := es.ProcessProposal(context.Background(), signedProp)
	assert.NoError(t, err)
	assert.Equal(t, []byte{1, 2, 3, 4, 5}, resp.Endorsement.Signature)
	assert.Equal(t, []byte{1, 1, 1}, resp.Endorsement.Endorser)
	assert.Equal(t, 200, int(resp.Response.Status))
}

func TestSimulateProposal(t *testing.T) {
	es := endorser.NewEndorserServer(pvtEmptyDistributor, &em.MockSupport{
		GetApplicationConfigBoolRv: true,
		GetApplicationConfigRv:     &mc.MockApplication{CapabilitiesRv: &mc.MockApplicationCapabilities{}},
		GetTransactionByIDErr:      errors.New(""),
		ChaincodeDefinitionRv:      &ccprovider.ChaincodeData{Escc: "ESCC"},
		ExecuteResp:                &pb.Response{Status: 200, Payload: utils.MarshalOrPanic(&pb.ProposalResponse{Response: &pb.Response{}})},
		GetTxSimulatorRv: &mockccprovider.MockTxSim{
			GetTxSimulationResultsRv: &ledger.TxSimulationResults{
				PubSimulationResults: &rwset.TxReadWriteSet{},
			},
		},
	})

	_, _, _, _, err := es.SimulateProposal(nil, "", "", nil, nil, nil, nil)
	assert.Error(t, err)
}

func TestEndorserJavaChecks(t *testing.T) {
	if endorser.JavaEnabled() {
		t.Skip("Java chaincode is supported")
	}

	es := endorser.NewEndorserServer(pvtEmptyDistributor, &em.MockSupport{
		GetApplicationConfigBoolRv: true,
		GetApplicationConfigRv:     &mc.MockApplication{CapabilitiesRv: &mc.MockApplicationCapabilities{}},
		GetTransactionByIDErr:      errors.New(""),
		ChaincodeDefinitionRv:      &ccprovider.ChaincodeData{Escc: "ESCC"},
		ExecuteResp:                &pb.Response{Status: 200, Payload: utils.MarshalOrPanic(&pb.ProposalResponse{Response: &pb.Response{}})},
		GetTxSimulatorRv: &mockccprovider.MockTxSim{
			GetTxSimulationResultsRv: &ledger.TxSimulationResults{
				PubSimulationResults: &rwset.TxReadWriteSet{},
			},
		},
	})

	err := es.DisableJavaCCInst(&pb.ChaincodeID{Name: "lscc"}, &pb.ChaincodeInvocationSpec{})
	assert.NoError(t, err)
	err = es.DisableJavaCCInst(&pb.ChaincodeID{Name: "lscc"}, &pb.ChaincodeInvocationSpec{ChaincodeSpec: &pb.ChaincodeSpec{Input: &pb.ChaincodeInput{}}})
	assert.NoError(t, err)
	err = es.DisableJavaCCInst(&pb.ChaincodeID{Name: "lscc"}, &pb.ChaincodeInvocationSpec{ChaincodeSpec: &pb.ChaincodeSpec{Input: &pb.ChaincodeInput{Args: [][]byte{[]byte("foo")}}}})
	assert.NoError(t, err)
	err = es.DisableJavaCCInst(&pb.ChaincodeID{Name: "lscc"}, &pb.ChaincodeInvocationSpec{ChaincodeSpec: &pb.ChaincodeSpec{Input: &pb.ChaincodeInput{Args: [][]byte{[]byte("install")}}}})
	assert.Error(t, err)
}

func TestEndorserAcquireTxSimulator(t *testing.T) {
	tc := []struct {
		name          string
		chainID       string
		chaincodeName string
		simAcquired   bool
	}{
		{"empty channel", "", "ignored", false},
		{"query scc", util.GetTestChainID(), "qscc", false},
		{"config scc", util.GetTestChainID(), "cscc", false},
		{"mainline", util.GetTestChainID(), "chaincode", true},
	}

	expectedResponse := &pb.Response{Status: 200, Payload: utils.MarshalOrPanic(&pb.ProposalResponse{Response: &pb.Response{}})}
	for _, tt := range tc {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			m := &mock.Mock{}
			m.On("Sign", mock.Anything).Return([]byte{1, 2, 3, 4, 5}, nil)
			m.On("Serialize").Return([]byte{1, 1, 1}, nil)
			m.On("GetTxSimulator", mock.Anything, mock.Anything).Return(newMockTxSim(), nil)
			support := &em.MockSupport{
				Mock: m,
				GetApplicationConfigBoolRv: true,
				GetApplicationConfigRv:     &mc.MockApplication{CapabilitiesRv: &mc.MockApplicationCapabilities{}},
				GetTransactionByIDErr:      errors.New(""),
				ChaincodeDefinitionRv:      &ccprovider.ChaincodeData{Escc: "ESCC"},
				ExecuteResp:                expectedResponse,
			}
			attachPluginEndorser(support)
			es := endorser.NewEndorserServer(pvtEmptyDistributor, support)

			t.Parallel()
			args := [][]byte{[]byte("args")}
			signedProp := getSignedPropWithCHIdAndArgs(tt.chainID, tt.chaincodeName, "version", args, t)

			resp, err := es.ProcessProposal(context.Background(), signedProp)
			assert.NoError(t, err)
			assert.Equal(t, expectedResponse, resp.Response)

			if tt.simAcquired {
				m.AssertCalled(t, "GetTxSimulator", mock.Anything, mock.Anything)
			} else {
				m.AssertNotCalled(t, "GetTxSimulator", mock.Anything, mock.Anything)
			}
		})
	}
}

var signer msp.SigningIdentity

func TestMain(m *testing.M) {
	// setup the MSP manager so that we can sign/verify
	err := msptesttools.LoadMSPSetupForTesting()
	if err != nil {
		fmt.Printf("Could not initialize msp/signer, err %s", err)
		os.Exit(-1)
		return
	}
	signer, err = mgmt.GetLocalMSP().GetDefaultSigningIdentity()
	if err != nil {
		fmt.Printf("Could not initialize msp/signer")
		os.Exit(-1)
		return
	}

	retVal := m.Run()
	os.Exit(retVal)
}

//go:generate counterfeiter -o mocks/support.go --fake-name Support . support
type support interface {
	endorser.Support
}

func TestUserCDSSanitization(t *testing.T) {
	fakeSupport := &mocks.Support{}
	e := endorser.NewEndorserServer(nil, fakeSupport)

	userCDS := &pb.ChaincodeDeploymentSpec{
		ChaincodeSpec: &pb.ChaincodeSpec{
			ChaincodeId: &pb.ChaincodeID{
				Name:    "user-cc-name",
				Version: "user-cc-version",
				Path:    "user-cc-path",
			},
			Input: &pb.ChaincodeInput{
				Args: [][]byte{[]byte("foo"), []byte("bar")},
			},
			Type: pb.ChaincodeSpec_GOLANG,
		},
		CodePackage: []byte("user-code"),
	}

	fsCDS := &pb.ChaincodeDeploymentSpec{
		ChaincodeSpec: &pb.ChaincodeSpec{
			ChaincodeId: &pb.ChaincodeID{
				Name:    "fs-cc-name",
				Version: "fs-cc-version",
				Path:    "fs-cc-path",
			},
			Type: pb.ChaincodeSpec_GOLANG,
		},
		CodePackage: []byte("fs-code"),
	}

	fakeSupport.GetChaincodeDeploymentSpecFSReturns(fsCDS, nil)

	sanitizedCDS, err := e.SanitizeUserCDS(userCDS)
	assert.NoError(t, err)
	assert.Nil(t, sanitizedCDS.CodePackage)
	assert.True(t, proto.Equal(userCDS.ChaincodeSpec.Input, sanitizedCDS.ChaincodeSpec.Input))
	assert.True(t, proto.Equal(fsCDS.ChaincodeSpec.ChaincodeId, sanitizedCDS.ChaincodeSpec.ChaincodeId))

	t.Run("BadPath", func(t *testing.T) {
		fakeSupport.GetChaincodeDeploymentSpecFSReturns(nil, fmt.Errorf("fake-error"))
		_, err := e.SanitizeUserCDS(userCDS)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "fake-error")
	})
}
