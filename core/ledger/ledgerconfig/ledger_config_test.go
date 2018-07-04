/*h
Copyright IBM Corp. 2016 All Rights Reserved.

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

package ledgerconfig

import (
	"testing"

	"github.com/sinochem-tech/fabric/common/ledger/testutil"
	ledgertestutil "github.com/sinochem-tech/fabric/core/ledger/testutil"
	"github.com/spf13/viper"
)

func TestIsCouchDBEnabledDefault(t *testing.T) {
	setUpCoreYAMLConfig()
	// During a build the default values should be false.

	// If the  ledger test are run with CouchDb enabled, need to provide a mechanism
	// To let this test run but still test default values.
	if IsCouchDBEnabled() == true {
		ledgertestutil.ResetConfigToDefaultValues()
		defer viper.Set("ledger.state.stateDatabase", "CouchDB")
	}
	defaultValue := IsCouchDBEnabled()
	testutil.AssertEquals(t, defaultValue, false) //test default config is false
}

func TestIsCouchDBEnabled(t *testing.T) {
	setUpCoreYAMLConfig()
	defer ledgertestutil.ResetConfigToDefaultValues()
	viper.Set("ledger.state.stateDatabase", "CouchDB")
	updatedValue := IsCouchDBEnabled()
	testutil.AssertEquals(t, updatedValue, true) //test config returns true
}

func TestLedgerConfigPathDefault(t *testing.T) {
	setUpCoreYAMLConfig()
	testutil.AssertEquals(t,
		GetRootPath(),
		"/var/hyperledger/production/ledgersData")
	testutil.AssertEquals(t,
		GetLedgerProviderPath(),
		"/var/hyperledger/production/ledgersData/ledgerProvider")
	testutil.AssertEquals(t,
		GetStateLevelDBPath(),
		"/var/hyperledger/production/ledgersData/stateLeveldb")
	testutil.AssertEquals(t,
		GetHistoryLevelDBPath(),
		"/var/hyperledger/production/ledgersData/historyLeveldb")
	testutil.AssertEquals(t,
		GetBlockStorePath(),
		"/var/hyperledger/production/ledgersData/chains")
	testutil.AssertEquals(t,
		GetPvtdataStorePath(),
		"/var/hyperledger/production/ledgersData/pvtdataStore")
	testutil.AssertEquals(t,
		GetInternalBookkeeperPath(),
		"/var/hyperledger/production/ledgersData/bookkeeper")

}

func TestLedgerConfigPath(t *testing.T) {
	setUpCoreYAMLConfig()
	defer ledgertestutil.ResetConfigToDefaultValues()
	viper.Set("peer.fileSystemPath", "/tmp/hyperledger/production")
	testutil.AssertEquals(t,
		GetRootPath(),
		"/tmp/hyperledger/production/ledgersData")
	testutil.AssertEquals(t,
		GetLedgerProviderPath(),
		"/tmp/hyperledger/production/ledgersData/ledgerProvider")
	testutil.AssertEquals(t,
		GetStateLevelDBPath(),
		"/tmp/hyperledger/production/ledgersData/stateLeveldb")
	testutil.AssertEquals(t,
		GetHistoryLevelDBPath(),
		"/tmp/hyperledger/production/ledgersData/historyLeveldb")
	testutil.AssertEquals(t,
		GetBlockStorePath(),
		"/tmp/hyperledger/production/ledgersData/chains")
	testutil.AssertEquals(t,
		GetPvtdataStorePath(),
		"/tmp/hyperledger/production/ledgersData/pvtdataStore")
	testutil.AssertEquals(t,
		GetInternalBookkeeperPath(),
		"/tmp/hyperledger/production/ledgersData/bookkeeper")
}

func TestGetQueryLimitDefault(t *testing.T) {
	setUpCoreYAMLConfig()
	defaultValue := GetQueryLimit()
	testutil.AssertEquals(t, defaultValue, 10000) //test default config is 10000
}

func TestGetQueryLimitUnset(t *testing.T) {
	viper.Reset()
	defaultValue := GetQueryLimit()
	testutil.AssertEquals(t, defaultValue, 10000) //test default config is 10000
}

func TestGetQueryLimit(t *testing.T) {
	setUpCoreYAMLConfig()
	defer ledgertestutil.ResetConfigToDefaultValues()
	viper.Set("ledger.state.couchDBConfig.queryLimit", 5000)
	updatedValue := GetQueryLimit()
	testutil.AssertEquals(t, updatedValue, 5000) //test config returns 5000
}

func TestMaxBatchUpdateSizeDefault(t *testing.T) {
	setUpCoreYAMLConfig()
	defaultValue := GetMaxBatchUpdateSize()
	testutil.AssertEquals(t, defaultValue, 1000) //test default config is 1000
}

func TestMaxBatchUpdateSizeUnset(t *testing.T) {
	viper.Reset()
	defaultValue := GetMaxBatchUpdateSize()
	testutil.AssertEquals(t, defaultValue, 500) // 500 if maxBatchUpdateSize is not set
}

func TestMaxBatchUpdateSize(t *testing.T) {
	setUpCoreYAMLConfig()
	defer ledgertestutil.ResetConfigToDefaultValues()
	viper.Set("ledger.state.couchDBConfig.maxBatchUpdateSize", 2000)
	updatedValue := GetMaxBatchUpdateSize()
	testutil.AssertEquals(t, updatedValue, 2000) //test config returns 2000
}

func TestPvtdataStorePurgeIntervalDefault(t *testing.T) {
	setUpCoreYAMLConfig()
	defaultValue := GetPvtdataStorePurgeInterval()
	testutil.AssertEquals(t, defaultValue, uint64(100)) //test default config is 100
}

func TestPvtdataStorePurgeIntervalUnset(t *testing.T) {
	viper.Reset()
	defaultValue := GetPvtdataStorePurgeInterval()
	testutil.AssertEquals(t, defaultValue, uint64(100)) // 100 if purgeInterval is not set
}

func TestIsQueryReadHasingEnabled(t *testing.T) {
	testutil.AssertEquals(t, IsQueryReadsHashingEnabled(), true)
}

func TestGetMaxDegreeQueryReadsHashing(t *testing.T) {
	testutil.AssertEquals(t, GetMaxDegreeQueryReadsHashing(), uint32(50))
}

func TestPvtdataStorePurgeInterval(t *testing.T) {
	setUpCoreYAMLConfig()
	defer ledgertestutil.ResetConfigToDefaultValues()
	viper.Set("ledger.pvtdataStore.purgeInterval", 1000)
	updatedValue := GetPvtdataStorePurgeInterval()
	testutil.AssertEquals(t, updatedValue, uint64(1000)) //test config returns 1000
}

func TestIsHistoryDBEnabledDefault(t *testing.T) {
	setUpCoreYAMLConfig()
	defaultValue := IsHistoryDBEnabled()
	testutil.AssertEquals(t, defaultValue, false) //test default config is false
}

func TestIsHistoryDBEnabledTrue(t *testing.T) {
	setUpCoreYAMLConfig()
	defer ledgertestutil.ResetConfigToDefaultValues()
	viper.Set("ledger.history.enableHistoryDatabase", true)
	updatedValue := IsHistoryDBEnabled()
	testutil.AssertEquals(t, updatedValue, true) //test config returns true
}

func TestIsHistoryDBEnabledFalse(t *testing.T) {
	setUpCoreYAMLConfig()
	defer ledgertestutil.ResetConfigToDefaultValues()
	viper.Set("ledger.history.enableHistoryDatabase", false)
	updatedValue := IsHistoryDBEnabled()
	testutil.AssertEquals(t, updatedValue, false) //test config returns false
}

func TestIsAutoWarmIndexesEnabledDefault(t *testing.T) {
	setUpCoreYAMLConfig()
	defaultValue := IsAutoWarmIndexesEnabled()
	testutil.AssertEquals(t, defaultValue, true) //test default config is true
}

func TestIsAutoWarmIndexesEnabledUnset(t *testing.T) {
	viper.Reset()
	defaultValue := IsAutoWarmIndexesEnabled()
	testutil.AssertEquals(t, defaultValue, true) //test default config is true
}

func TestIsAutoWarmIndexesEnabledTrue(t *testing.T) {
	setUpCoreYAMLConfig()
	defer ledgertestutil.ResetConfigToDefaultValues()
	viper.Set("ledger.state.couchDBConfig.autoWarmIndexes", true)
	updatedValue := IsAutoWarmIndexesEnabled()
	testutil.AssertEquals(t, updatedValue, true) //test config returns true
}

func TestIsAutoWarmIndexesEnabledFalse(t *testing.T) {
	setUpCoreYAMLConfig()
	defer ledgertestutil.ResetConfigToDefaultValues()
	viper.Set("ledger.state.couchDBConfig.autoWarmIndexes", false)
	updatedValue := IsAutoWarmIndexesEnabled()
	testutil.AssertEquals(t, updatedValue, false) //test config returns false
}

func TestGetWarmIndexesAfterNBlocksDefault(t *testing.T) {
	setUpCoreYAMLConfig()
	defaultValue := GetWarmIndexesAfterNBlocks()
	testutil.AssertEquals(t, defaultValue, 1) //test default config is true
}

func TestGetWarmIndexesAfterNBlocksUnset(t *testing.T) {
	viper.Reset()
	defaultValue := GetWarmIndexesAfterNBlocks()
	testutil.AssertEquals(t, defaultValue, 1) //test default config is true
}

func TestGetWarmIndexesAfterNBlocks(t *testing.T) {
	setUpCoreYAMLConfig()
	defer ledgertestutil.ResetConfigToDefaultValues()
	viper.Set("ledger.state.couchDBConfig.warmIndexesAfterNBlocks", 10)
	updatedValue := GetWarmIndexesAfterNBlocks()
	testutil.AssertEquals(t, updatedValue, 10)
}

func TestGetMaxBlockfileSize(t *testing.T) {
	testutil.AssertEquals(t, GetMaxBlockfileSize(), 67108864)
}

func setUpCoreYAMLConfig() {
	//call a helper method to load the core.yaml
	ledgertestutil.SetupCoreYAMLConfig()
}
