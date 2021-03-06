/*
Copyright IBM Corp. 2017 All Rights Reserved.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

		 http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package cauthdsl

import (
	"reflect"
	"testing"

	"github.com/sinochem-tech/fabric/protos/common"
	"github.com/sinochem-tech/fabric/protos/msp"
	"github.com/sinochem-tech/fabric/protos/utils"
	"github.com/stretchr/testify/assert"
)

func TestOutOf1(t *testing.T) {
	p1, err := FromString("OutOf(1, 'A.member', 'B.member')")
	assert.NoError(t, err)

	principals := make([]*msp.MSPPrincipal, 0)

	principals = append(principals, &msp.MSPPrincipal{
		PrincipalClassification: msp.MSPPrincipal_ROLE,
		Principal:               utils.MarshalOrPanic(&msp.MSPRole{Role: msp.MSPRole_MEMBER, MspIdentifier: "A"})})

	principals = append(principals, &msp.MSPPrincipal{
		PrincipalClassification: msp.MSPPrincipal_ROLE,
		Principal:               utils.MarshalOrPanic(&msp.MSPRole{Role: msp.MSPRole_MEMBER, MspIdentifier: "B"})})

	p2 := &common.SignaturePolicyEnvelope{
		Version:    0,
		Rule:       NOutOf(1, []*common.SignaturePolicy{SignedBy(0), SignedBy(1)}),
		Identities: principals,
	}

	assert.Equal(t, p1, p2)
}

func TestOutOf2(t *testing.T) {
	p1, err := FromString("OutOf(2, 'A.member', 'B.member')")
	assert.NoError(t, err)

	principals := make([]*msp.MSPPrincipal, 0)

	principals = append(principals, &msp.MSPPrincipal{
		PrincipalClassification: msp.MSPPrincipal_ROLE,
		Principal:               utils.MarshalOrPanic(&msp.MSPRole{Role: msp.MSPRole_MEMBER, MspIdentifier: "A"})})

	principals = append(principals, &msp.MSPPrincipal{
		PrincipalClassification: msp.MSPPrincipal_ROLE,
		Principal:               utils.MarshalOrPanic(&msp.MSPRole{Role: msp.MSPRole_MEMBER, MspIdentifier: "B"})})

	p2 := &common.SignaturePolicyEnvelope{
		Version:    0,
		Rule:       NOutOf(2, []*common.SignaturePolicy{SignedBy(0), SignedBy(1)}),
		Identities: principals,
	}

	assert.Equal(t, p1, p2)
}

func TestAnd(t *testing.T) {
	p1, err := FromString("AND('A.member', 'B.member')")
	assert.NoError(t, err)

	principals := make([]*msp.MSPPrincipal, 0)

	principals = append(principals, &msp.MSPPrincipal{
		PrincipalClassification: msp.MSPPrincipal_ROLE,
		Principal:               utils.MarshalOrPanic(&msp.MSPRole{Role: msp.MSPRole_MEMBER, MspIdentifier: "A"})})

	principals = append(principals, &msp.MSPPrincipal{
		PrincipalClassification: msp.MSPPrincipal_ROLE,
		Principal:               utils.MarshalOrPanic(&msp.MSPRole{Role: msp.MSPRole_MEMBER, MspIdentifier: "B"})})

	p2 := &common.SignaturePolicyEnvelope{
		Version:    0,
		Rule:       And(SignedBy(0), SignedBy(1)),
		Identities: principals,
	}

	assert.Equal(t, p1, p2)
}

func TestAndClientPeerOrderer(t *testing.T) {
	p1, err := FromString("AND('A.client', 'B.peer')")
	assert.NoError(t, err)

	principals := make([]*msp.MSPPrincipal, 0)

	principals = append(principals, &msp.MSPPrincipal{
		PrincipalClassification: msp.MSPPrincipal_ROLE,
		Principal:               utils.MarshalOrPanic(&msp.MSPRole{Role: msp.MSPRole_CLIENT, MspIdentifier: "A"})})

	principals = append(principals, &msp.MSPPrincipal{
		PrincipalClassification: msp.MSPPrincipal_ROLE,
		Principal:               utils.MarshalOrPanic(&msp.MSPRole{Role: msp.MSPRole_PEER, MspIdentifier: "B"})})

	p2 := &common.SignaturePolicyEnvelope{
		Version:    0,
		Rule:       And(SignedBy(0), SignedBy(1)),
		Identities: principals,
	}

	assert.True(t, reflect.DeepEqual(p1, p2))

}

func TestOr(t *testing.T) {
	p1, err := FromString("OR('A.member', 'B.member')")
	assert.NoError(t, err)

	principals := make([]*msp.MSPPrincipal, 0)

	principals = append(principals, &msp.MSPPrincipal{
		PrincipalClassification: msp.MSPPrincipal_ROLE,
		Principal:               utils.MarshalOrPanic(&msp.MSPRole{Role: msp.MSPRole_MEMBER, MspIdentifier: "A"})})

	principals = append(principals, &msp.MSPPrincipal{
		PrincipalClassification: msp.MSPPrincipal_ROLE,
		Principal:               utils.MarshalOrPanic(&msp.MSPRole{Role: msp.MSPRole_MEMBER, MspIdentifier: "B"})})

	p2 := &common.SignaturePolicyEnvelope{
		Version:    0,
		Rule:       Or(SignedBy(0), SignedBy(1)),
		Identities: principals,
	}

	assert.Equal(t, p1, p2)
}

func TestComplex1(t *testing.T) {
	p1, err := FromString("OR('A.member', AND('B.member', 'C.member'))")
	assert.NoError(t, err)

	principals := make([]*msp.MSPPrincipal, 0)

	principals = append(principals, &msp.MSPPrincipal{
		PrincipalClassification: msp.MSPPrincipal_ROLE,
		Principal:               utils.MarshalOrPanic(&msp.MSPRole{Role: msp.MSPRole_MEMBER, MspIdentifier: "B"})})

	principals = append(principals, &msp.MSPPrincipal{
		PrincipalClassification: msp.MSPPrincipal_ROLE,
		Principal:               utils.MarshalOrPanic(&msp.MSPRole{Role: msp.MSPRole_MEMBER, MspIdentifier: "C"})})

	principals = append(principals, &msp.MSPPrincipal{
		PrincipalClassification: msp.MSPPrincipal_ROLE,
		Principal:               utils.MarshalOrPanic(&msp.MSPRole{Role: msp.MSPRole_MEMBER, MspIdentifier: "A"})})

	p2 := &common.SignaturePolicyEnvelope{
		Version:    0,
		Rule:       Or(SignedBy(2), And(SignedBy(0), SignedBy(1))),
		Identities: principals,
	}

	assert.Equal(t, p1, p2)
}

func TestComplex2(t *testing.T) {
	p1, err := FromString("OR(AND('A.member', 'B.member'), OR('C.admin', 'D.member'))")
	assert.NoError(t, err)

	principals := make([]*msp.MSPPrincipal, 0)

	principals = append(principals, &msp.MSPPrincipal{
		PrincipalClassification: msp.MSPPrincipal_ROLE,
		Principal:               utils.MarshalOrPanic(&msp.MSPRole{Role: msp.MSPRole_MEMBER, MspIdentifier: "A"})})

	principals = append(principals, &msp.MSPPrincipal{
		PrincipalClassification: msp.MSPPrincipal_ROLE,
		Principal:               utils.MarshalOrPanic(&msp.MSPRole{Role: msp.MSPRole_MEMBER, MspIdentifier: "B"})})

	principals = append(principals, &msp.MSPPrincipal{
		PrincipalClassification: msp.MSPPrincipal_ROLE,
		Principal:               utils.MarshalOrPanic(&msp.MSPRole{Role: msp.MSPRole_ADMIN, MspIdentifier: "C"})})

	principals = append(principals, &msp.MSPPrincipal{
		PrincipalClassification: msp.MSPPrincipal_ROLE,
		Principal:               utils.MarshalOrPanic(&msp.MSPRole{Role: msp.MSPRole_MEMBER, MspIdentifier: "D"})})

	p2 := &common.SignaturePolicyEnvelope{
		Version:    0,
		Rule:       Or(And(SignedBy(0), SignedBy(1)), Or(SignedBy(2), SignedBy(3))),
		Identities: principals,
	}

	assert.Equal(t, p1, p2)
}

func TestMSPIDWIthSpecialChars(t *testing.T) {
	p1, err := FromString("OR('MSP.member', 'MSP.WITH.DOTS.member', 'MSP-WITH-DASHES.member')")
	assert.NoError(t, err)

	principals := make([]*msp.MSPPrincipal, 0)

	principals = append(principals, &msp.MSPPrincipal{
		PrincipalClassification: msp.MSPPrincipal_ROLE,
		Principal:               utils.MarshalOrPanic(&msp.MSPRole{Role: msp.MSPRole_MEMBER, MspIdentifier: "MSP"})})

	principals = append(principals, &msp.MSPPrincipal{
		PrincipalClassification: msp.MSPPrincipal_ROLE,
		Principal:               utils.MarshalOrPanic(&msp.MSPRole{Role: msp.MSPRole_MEMBER, MspIdentifier: "MSP.WITH.DOTS"})})

	principals = append(principals, &msp.MSPPrincipal{
		PrincipalClassification: msp.MSPPrincipal_ROLE,
		Principal:               utils.MarshalOrPanic(&msp.MSPRole{Role: msp.MSPRole_MEMBER, MspIdentifier: "MSP-WITH-DASHES"})})

	p2 := &common.SignaturePolicyEnvelope{
		Version:    0,
		Rule:       NOutOf(1, []*common.SignaturePolicy{SignedBy(0), SignedBy(1), SignedBy(2)}),
		Identities: principals,
	}

	assert.Equal(t, p1, p2)
}

func TestBadStringsNoPanic(t *testing.T) {
	_, err := FromString("OR('A.member', 'Bmember')")
	assert.Error(t, err)
	_, err = FromString("OR('A.member', Bmember)")
	assert.Error(t, err)
}
