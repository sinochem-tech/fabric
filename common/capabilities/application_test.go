/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package capabilities

import (
	"testing"

	cb "github.com/sinochem-tech/fabric/protos/common"

	"github.com/stretchr/testify/assert"
)

func TestApplicationV10(t *testing.T) {
	op := NewApplicationProvider(map[string]*cb.Capability{})
	assert.NoError(t, op.Supported())
}

func TestApplicationV11(t *testing.T) {
	op := NewApplicationProvider(map[string]*cb.Capability{
		ApplicationV1_1: {},
	})
	assert.NoError(t, op.Supported())
	assert.True(t, op.ForbidDuplicateTXIdInBlock())
	assert.True(t, op.V1_1Validation())
}

func TestApplicationV12(t *testing.T) {
	op := NewApplicationProvider(map[string]*cb.Capability{
		ApplicationV1_2: {},
	})
	assert.NoError(t, op.Supported())
	assert.True(t, op.ForbidDuplicateTXIdInBlock())
	assert.True(t, op.V1_1Validation())
	assert.True(t, op.V1_2Validation())
	assert.True(t, op.KeyLevelEndorsement())
}

func TestApplicationPvtDataExperimental(t *testing.T) {
	op := NewApplicationProvider(map[string]*cb.Capability{
		ApplicationPvtDataExperimental: {},
	})
	assert.True(t, op.PrivateChannelData())

	op = NewApplicationProvider(map[string]*cb.Capability{
		ApplicationV1_2: {},
	})
	assert.True(t, op.PrivateChannelData())

}

func TestApplicationACLs(t *testing.T) {
	ap := NewApplicationProvider(map[string]*cb.Capability{
		ApplicationV1_2: {},
	})
	assert.True(t, ap.ACLs())
}

func TestApplicationCollectionUpgrade(t *testing.T) {
	op := NewApplicationProvider(map[string]*cb.Capability{
		ApplicationV1_2: {},
	})
	assert.True(t, op.CollectionUpgrade())
}

func TestChaincodeLifecycleExperimental(t *testing.T) {
	op := NewApplicationProvider(map[string]*cb.Capability{
		ApplicationChaincodeLifecycleExperimental: {},
	})
	assert.True(t, op.MetadataLifecycle())
}
