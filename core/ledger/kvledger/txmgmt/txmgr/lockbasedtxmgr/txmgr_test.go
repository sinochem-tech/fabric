/*
Copyright IBM Corp. All Rights Reserved.
SPDX-License-Identifier: Apache-2.0
*/

package lockbasedtxmgr

import (
	"bytes"
	"encoding/gob"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/sinochem-tech/fabric/core/ledger/pvtdatapolicy"
	btltestutil "github.com/sinochem-tech/fabric/core/ledger/pvtdatapolicy/testutil"

	"os"

	"github.com/sinochem-tech/fabric/common/flogging"
	"github.com/sinochem-tech/fabric/common/ledger/testutil"
	"github.com/sinochem-tech/fabric/core/ledger"
	"github.com/sinochem-tech/fabric/core/ledger/kvledger/txmgmt/privacyenabledstate"
	"github.com/sinochem-tech/fabric/core/ledger/kvledger/txmgmt/txmgr"
	"github.com/sinochem-tech/fabric/core/ledger/kvledger/txmgmt/version"
	ledgertestutil "github.com/sinochem-tech/fabric/core/ledger/testutil"
	"github.com/sinochem-tech/fabric/core/ledger/util"
	"github.com/sinochem-tech/fabric/protos/ledger/queryresult"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
)

func TestMain(m *testing.M) {
	ledgertestutil.SetupCoreYAMLConfig()
	flogging.SetModuleLevel("lockbasedtxmgr", "debug")
	flogging.SetModuleLevel("statevalidator", "debug")
	flogging.SetModuleLevel("statebasedval", "debug")
	flogging.SetModuleLevel("statecouchdb", "debug")
	flogging.SetModuleLevel("valimpl", "debug")
	flogging.SetModuleLevel("pvtstatepurgemgmt", "debug")

	viper.Set("peer.fileSystemPath", "/tmp/fabric/ledgertests/kvledger/txmgmt/txmgr/lockbasedtxmgr")
	os.Exit(m.Run())
}

func TestTxSimulatorWithNoExistingData(t *testing.T) {
	// run the tests for each environment configured in pkg_test.go
	for _, testEnv := range testEnvs {
		t.Logf("Running test for TestEnv = %s", testEnv.getName())
		testLedgerID := "testtxsimulatorwithnoexistingdata"
		testEnv.init(t, testLedgerID, nil)
		testTxSimulatorWithNoExistingData(t, testEnv)
		testEnv.cleanup()
	}
}

func testTxSimulatorWithNoExistingData(t *testing.T, env testEnv) {
	txMgr := env.getTxMgr()
	s, _ := txMgr.NewTxSimulator("test_txid")
	value, err := s.GetState("ns1", "key1")
	testutil.AssertNoError(t, err, fmt.Sprintf("Error in GetState(): %s", err))
	testutil.AssertNil(t, value)

	s.SetState("ns1", "key1", []byte("value1"))
	s.SetState("ns1", "key2", []byte("value2"))
	s.SetState("ns2", "key3", []byte("value3"))
	s.SetState("ns2", "key4", []byte("value4"))

	value, _ = s.GetState("ns2", "key3")
	testutil.AssertNil(t, value)

	simulationResults, err := s.GetTxSimulationResults()
	assert.NoError(t, err)
	assert.Nil(t, simulationResults.PvtSimulationResults)
}

func TestTxSimulatorGetResults(t *testing.T) {
	testEnv := testEnvsMap[levelDBtestEnvName]
	testEnv.init(t, "testLedger", nil)
	defer testEnv.cleanup()
	txMgr := testEnv.getTxMgr()
	populateCollConfigForTest(t, txMgr.(*LockBasedTxMgr),
		[]collConfigkey{
			{"ns1", "coll1"},
			{"ns1", "coll3"},
			{"ns2", "coll2"},
			{"ns3", "coll3"},
		},
		version.NewHeight(1, 1),
	)

	var err error

	// Create a simulator and get/set keys in one namespace "ns1"
	simulator, _ := testEnv.getTxMgr().NewTxSimulator("test_txid1")
	simulator.GetState("ns1", "key1")
	_, err = simulator.GetPrivateData("ns1", "coll1", "key1")
	assert.NoError(t, err)
	simulator.SetState("ns1", "key1", []byte("value1"))
	// get simulation results and verify that this contains rwset only for one namespace
	simulationResults1, err := simulator.GetTxSimulationResults()
	assert.NoError(t, err)
	assert.Equal(t, 2, len(simulationResults1.PubSimulationResults.NsRwset))
	// clone freeze simulationResults1
	buff1 := new(bytes.Buffer)
	assert.NoError(t, gob.NewEncoder(buff1).Encode(simulationResults1))
	frozenSimulationResults1 := &ledger.TxSimulationResults{}
	assert.NoError(t, gob.NewDecoder(buff1).Decode(&frozenSimulationResults1))

	// use the same simulator after obtaining the simulaiton results by get/set keys in one more namespace "ns2"
	simulator.GetState("ns2", "key2")
	simulator.GetPrivateData("ns2", "coll2", "key2")
	simulator.SetState("ns2", "key2", []byte("value2"))
	// get simulation results and verify that an error is raised when obtaining the simulation results more than once
	_, err = simulator.GetTxSimulationResults()
	assert.Error(t, err) // calling 'GetTxSimulationResults()' more than once should raise error
	// Now, verify that the simulator operations did not have an effect on privously obtained results
	assert.Equal(t, frozenSimulationResults1, simulationResults1)

	// Call 'Done' and all the data get/set operations after calling 'Done' should fail.
	simulator.Done()
	_, err = simulator.GetState("ns3", "key3")
	assert.Errorf(t, err, "An error is expected when using simulator to get/set data after calling `Done` function()")
	err = simulator.SetState("ns3", "key3", []byte("value3"))
	assert.Errorf(t, err, "An error is expected when using simulator to get/set data after calling `Done` function()")
	_, err = simulator.GetPrivateData("ns3", "coll3", "key3")
	assert.Errorf(t, err, "An error is expected when using simulator to get/set data after calling `Done` function()")
	err = simulator.SetPrivateData("ns3", "coll3", "key3", []byte("value3"))
	assert.Errorf(t, err, "An error is expected when using simulator to get/set data after calling `Done` function()")
}

func TestTxSimulatorWithExistingData(t *testing.T) {
	for _, testEnv := range testEnvs {
		t.Run(testEnv.getName(), func(t *testing.T) {
			testLedgerID := "testtxsimulatorwithexistingdata"
			testEnv.init(t, testLedgerID, nil)
			testTxSimulatorWithExistingData(t, testEnv)
			testEnv.cleanup()
		})
	}
}

func testTxSimulatorWithExistingData(t *testing.T, env testEnv) {
	txMgr := env.getTxMgr()
	txMgrHelper := newTxMgrTestHelper(t, txMgr)
	// simulate tx1
	s1, _ := txMgr.NewTxSimulator("test_tx1")
	s1.SetState("ns1", "key1", []byte("value1"))
	s1.SetState("ns1", "key2", []byte("value2"))
	s1.SetState("ns2", "key3", []byte("value3"))
	s1.SetState("ns2", "key4", []byte("value4"))
	s1.Done()
	// validate and commit RWset
	txRWSet1, _ := s1.GetTxSimulationResults()
	txMgrHelper.validateAndCommitRWSet(txRWSet1.PubSimulationResults)

	// simulate tx2 that make changes to existing data
	s2, _ := txMgr.NewTxSimulator("test_tx2")
	value, _ := s2.GetState("ns1", "key1")
	testutil.AssertEquals(t, value, []byte("value1"))
	s2.SetState("ns1", "key1", []byte("value1_1"))
	s2.DeleteState("ns2", "key3")
	value, _ = s2.GetState("ns1", "key1")
	testutil.AssertEquals(t, value, []byte("value1"))
	s2.Done()
	// validate and commit RWset for tx2
	txRWSet2, _ := s2.GetTxSimulationResults()
	txMgrHelper.validateAndCommitRWSet(txRWSet2.PubSimulationResults)

	// simulate tx3
	s3, _ := txMgr.NewTxSimulator("test_tx3")
	value, _ = s3.GetState("ns1", "key1")
	testutil.AssertEquals(t, value, []byte("value1_1"))
	value, _ = s3.GetState("ns2", "key3")
	testutil.AssertEquals(t, value, nil)
	s3.Done()

	// verify the versions of keys in persistence
	vv, _ := env.getVDB().GetState("ns1", "key1")
	testutil.AssertEquals(t, vv.Version, version.NewHeight(2, 0))
	vv, _ = env.getVDB().GetState("ns1", "key2")
	testutil.AssertEquals(t, vv.Version, version.NewHeight(1, 0))
}

func TestTxValidation(t *testing.T) {
	for _, testEnv := range testEnvs {
		t.Logf("Running test for TestEnv = %s", testEnv.getName())
		testLedgerID := "testtxvalidation"
		testEnv.init(t, testLedgerID, nil)
		testTxValidation(t, testEnv)
		testEnv.cleanup()
	}
}

func testTxValidation(t *testing.T, env testEnv) {
	txMgr := env.getTxMgr()
	txMgrHelper := newTxMgrTestHelper(t, txMgr)
	// simulate tx1
	s1, _ := txMgr.NewTxSimulator("test_tx1")
	s1.SetState("ns1", "key1", []byte("value1"))
	s1.SetState("ns1", "key2", []byte("value2"))
	s1.SetState("ns2", "key3", []byte("value3"))
	s1.SetState("ns2", "key4", []byte("value4"))
	s1.Done()
	// validate and commit RWset
	txRWSet1, _ := s1.GetTxSimulationResults()
	txMgrHelper.validateAndCommitRWSet(txRWSet1.PubSimulationResults)

	// simulate tx2 that make changes to existing data.
	// tx2: Read/Update ns1:key1, Delete ns2:key3.
	s2, _ := txMgr.NewTxSimulator("test_tx2")
	value, _ := s2.GetState("ns1", "key1")
	testutil.AssertEquals(t, value, []byte("value1"))

	s2.SetState("ns1", "key1", []byte("value1_2"))
	s2.DeleteState("ns2", "key3")
	s2.Done()

	// simulate tx3 before committing tx2 changes. Reads and modifies the key changed by tx2.
	// tx3: Read/Update ns1:key1
	s3, _ := txMgr.NewTxSimulator("test_tx3")
	s3.GetState("ns1", "key1")
	s3.SetState("ns1", "key1", []byte("value1_3"))
	s3.Done()

	// simulate tx4 before committing tx2 changes. Reads and Deletes the key changed by tx2
	// tx4: Read/Delete ns2:key3
	s4, _ := txMgr.NewTxSimulator("test_tx4")
	s4.GetState("ns2", "key3")
	s4.DeleteState("ns2", "key3")
	s4.Done()

	// simulate tx5 before committing tx2 changes. Modifies and then Reads the key changed by tx2 and writes a new key
	// tx5: Update/Read ns1:key1
	s5, _ := txMgr.NewTxSimulator("test_tx5")
	s5.SetState("ns1", "key1", []byte("new_value"))
	s5.GetState("ns1", "key1")
	s5.Done()

	// simulate tx6 before committing tx2 changes. Only writes a new key, does not reads/writes a key changed by tx2
	// tx6: Update ns1:new_key
	s6, _ := txMgr.NewTxSimulator("test_tx6")
	s6.SetState("ns1", "new_key", []byte("new_value"))
	s6.Done()

	// Summary of simulated transactions
	// tx2: Read/Update ns1:key1, Delete ns2:key3.
	// tx3: Read/Update ns1:key1
	// tx4: Read/Delete ns2:key3
	// tx5: Update/Read ns1:key1
	// tx6: Update ns1:new_key

	// validate and commit RWset for tx2
	txRWSet2, _ := s2.GetTxSimulationResults()
	txMgrHelper.validateAndCommitRWSet(txRWSet2.PubSimulationResults)

	//RWSet for tx3 and tx4 and tx5 should be invalid now due to read conflicts
	txRWSet3, _ := s3.GetTxSimulationResults()
	txMgrHelper.checkRWsetInvalid(txRWSet3.PubSimulationResults)

	txRWSet4, _ := s4.GetTxSimulationResults()
	txMgrHelper.checkRWsetInvalid(txRWSet4.PubSimulationResults)

	txRWSet5, _ := s5.GetTxSimulationResults()
	txMgrHelper.checkRWsetInvalid(txRWSet5.PubSimulationResults)

	// tx6 should still be valid as it only writes a new key
	txRWSet6, _ := s6.GetTxSimulationResults()
	txMgrHelper.validateAndCommitRWSet(txRWSet6.PubSimulationResults)
}

func TestTxPhantomValidation(t *testing.T) {
	for _, testEnv := range testEnvs {
		t.Logf("Running test for TestEnv = %s", testEnv.getName())
		testLedgerID := "testtxphantomvalidation"
		testEnv.init(t, testLedgerID, nil)
		testTxPhantomValidation(t, testEnv)
		testEnv.cleanup()
	}
}

func testTxPhantomValidation(t *testing.T, env testEnv) {
	txMgr := env.getTxMgr()
	txMgrHelper := newTxMgrTestHelper(t, txMgr)
	// simulate tx1
	s1, _ := txMgr.NewTxSimulator("test_tx1")
	s1.SetState("ns", "key1", []byte("value1"))
	s1.SetState("ns", "key2", []byte("value2"))
	s1.SetState("ns", "key3", []byte("value3"))
	s1.SetState("ns", "key4", []byte("value4"))
	s1.SetState("ns", "key5", []byte("value5"))
	s1.SetState("ns", "key6", []byte("value6"))
	// validate and commit RWset
	txRWSet1, _ := s1.GetTxSimulationResults()
	s1.Done() // explicitly calling done after obtaining the results to verify FAB-10788
	txMgrHelper.validateAndCommitRWSet(txRWSet1.PubSimulationResults)

	// simulate tx2
	s2, _ := txMgr.NewTxSimulator("test_tx2")
	itr2, _ := s2.GetStateRangeScanIterator("ns", "key2", "key5")
	for {
		if result, _ := itr2.Next(); result == nil {
			break
		}
	}
	s2.DeleteState("ns", "key3")
	txRWSet2, _ := s2.GetTxSimulationResults()
	s2.Done()

	// simulate tx3
	s3, _ := txMgr.NewTxSimulator("test_tx3")
	itr3, _ := s3.GetStateRangeScanIterator("ns", "key2", "key5")
	for {
		if result, _ := itr3.Next(); result == nil {
			break
		}
	}
	s3.SetState("ns", "key3", []byte("value3_new"))
	txRWSet3, _ := s3.GetTxSimulationResults()
	s3.Done()
	// simulate tx4
	s4, _ := txMgr.NewTxSimulator("test_tx4")
	itr4, _ := s4.GetStateRangeScanIterator("ns", "key4", "key6")
	for {
		if result, _ := itr4.Next(); result == nil {
			break
		}
	}
	s4.SetState("ns", "key3", []byte("value3_new"))
	txRWSet4, _ := s4.GetTxSimulationResults()
	s4.Done()

	// txRWSet2 should be valid
	txMgrHelper.validateAndCommitRWSet(txRWSet2.PubSimulationResults)
	// txRWSet2 makes txRWSet3 invalid as it deletes a key in the range
	txMgrHelper.checkRWsetInvalid(txRWSet3.PubSimulationResults)
	// txRWSet4 should be valid as it iterates over a different range
	txMgrHelper.validateAndCommitRWSet(txRWSet4.PubSimulationResults)
}

func TestIterator(t *testing.T) {
	for _, testEnv := range testEnvs {
		t.Logf("Running test for TestEnv = %s", testEnv.getName())

		testLedgerID := "testiterator.1"
		testEnv.init(t, testLedgerID, nil)
		testIterator(t, testEnv, 10, 2, 7)
		testEnv.cleanup()

		testLedgerID = "testiterator.2"
		testEnv.init(t, testLedgerID, nil)
		testIterator(t, testEnv, 10, 1, 11)
		testEnv.cleanup()

		testLedgerID = "testiterator.3"
		testEnv.init(t, testLedgerID, nil)
		testIterator(t, testEnv, 10, 0, 0)
		testEnv.cleanup()

		testLedgerID = "testiterator.4"
		testEnv.init(t, testLedgerID, nil)
		testIterator(t, testEnv, 10, 5, 0)
		testEnv.cleanup()

		testLedgerID = "testiterator.5"
		testEnv.init(t, testLedgerID, nil)
		testIterator(t, testEnv, 10, 0, 5)
		testEnv.cleanup()
	}
}

func testIterator(t *testing.T, env testEnv, numKeys int, startKeyNum int, endKeyNum int) {
	cID := "cid"
	txMgr := env.getTxMgr()
	txMgrHelper := newTxMgrTestHelper(t, txMgr)
	s, _ := txMgr.NewTxSimulator("test_tx1")
	for i := 1; i <= numKeys; i++ {
		k := createTestKey(i)
		v := createTestValue(i)
		t.Logf("Adding k=[%s], v=[%s]", k, v)
		s.SetState(cID, k, v)
	}
	s.Done()
	// validate and commit RWset
	txRWSet, _ := s.GetTxSimulationResults()
	txMgrHelper.validateAndCommitRWSet(txRWSet.PubSimulationResults)

	var startKey string
	var endKey string
	var begin int
	var end int

	if startKeyNum != 0 {
		begin = startKeyNum
		startKey = createTestKey(startKeyNum)
	} else {
		begin = 1 //first key in the db
		startKey = ""
	}

	if endKeyNum != 0 {
		endKey = createTestKey(endKeyNum)
		end = endKeyNum
	} else {
		endKey = ""
		end = numKeys + 1 //last key in the db
	}

	expectedCount := end - begin

	queryExecuter, _ := txMgr.NewQueryExecutor("test_tx2")
	itr, _ := queryExecuter.GetStateRangeScanIterator(cID, startKey, endKey)
	count := 0
	for {
		kv, _ := itr.Next()
		if kv == nil {
			break
		}
		keyNum := begin + count
		k := kv.(*queryresult.KV).Key
		v := kv.(*queryresult.KV).Value
		t.Logf("Retrieved k=%s, v=%s at count=%d start=%s end=%s", k, v, count, startKey, endKey)
		testutil.AssertEquals(t, k, createTestKey(keyNum))
		testutil.AssertEquals(t, v, createTestValue(keyNum))
		count++
	}
	testutil.AssertEquals(t, count, expectedCount)
}

func TestIteratorWithDeletes(t *testing.T) {
	for _, testEnv := range testEnvs {
		t.Logf("Running test for TestEnv = %s", testEnv.getName())
		testLedgerID := "testiteratorwithdeletes"
		testEnv.init(t, testLedgerID, nil)
		testIteratorWithDeletes(t, testEnv)
		testEnv.cleanup()
	}
}

func testIteratorWithDeletes(t *testing.T, env testEnv) {
	cID := "cid"
	txMgr := env.getTxMgr()
	txMgrHelper := newTxMgrTestHelper(t, txMgr)
	s, _ := txMgr.NewTxSimulator("test_tx1")
	for i := 1; i <= 10; i++ {
		k := createTestKey(i)
		v := createTestValue(i)
		t.Logf("Adding k=[%s], v=[%s]", k, v)
		s.SetState(cID, k, v)
	}
	s.Done()
	// validate and commit RWset
	txRWSet1, _ := s.GetTxSimulationResults()
	txMgrHelper.validateAndCommitRWSet(txRWSet1.PubSimulationResults)

	s, _ = txMgr.NewTxSimulator("test_tx2")
	s.DeleteState(cID, createTestKey(4))
	s.Done()
	// validate and commit RWset
	txRWSet2, _ := s.GetTxSimulationResults()
	txMgrHelper.validateAndCommitRWSet(txRWSet2.PubSimulationResults)

	queryExecuter, _ := txMgr.NewQueryExecutor("test_tx3")
	itr, _ := queryExecuter.GetStateRangeScanIterator(cID, createTestKey(3), createTestKey(6))
	defer itr.Close()
	kv, _ := itr.Next()
	testutil.AssertEquals(t, kv.(*queryresult.KV).Key, createTestKey(3))
	kv, _ = itr.Next()
	testutil.AssertEquals(t, kv.(*queryresult.KV).Key, createTestKey(5))
}

func TestTxValidationWithItr(t *testing.T) {
	for _, testEnv := range testEnvs {
		t.Logf("Running test for TestEnv = %s", testEnv.getName())
		testLedgerID := "testtxvalidationwithitr"
		testEnv.init(t, testLedgerID, nil)
		testTxValidationWithItr(t, testEnv)
		testEnv.cleanup()
	}
}

func testTxValidationWithItr(t *testing.T, env testEnv) {
	cID := "cid"
	txMgr := env.getTxMgr()
	txMgrHelper := newTxMgrTestHelper(t, txMgr)

	// simulate tx1
	s1, _ := txMgr.NewTxSimulator("test_tx1")
	for i := 1; i <= 10; i++ {
		k := createTestKey(i)
		v := createTestValue(i)
		t.Logf("Adding k=[%s], v=[%s]", k, v)
		s1.SetState(cID, k, v)
	}
	s1.Done()
	// validate and commit RWset
	txRWSet1, _ := s1.GetTxSimulationResults()
	txMgrHelper.validateAndCommitRWSet(txRWSet1.PubSimulationResults)

	// simulate tx2 that reads key_001 and key_002
	s2, _ := txMgr.NewTxSimulator("test_tx2")
	itr, _ := s2.GetStateRangeScanIterator(cID, createTestKey(1), createTestKey(5))
	// read key_001 and key_002
	itr.Next()
	itr.Next()
	itr.Close()
	s2.Done()

	// simulate tx3 that reads key_004 and key_005
	s3, _ := txMgr.NewTxSimulator("test_tx3")
	itr, _ = s3.GetStateRangeScanIterator(cID, createTestKey(4), createTestKey(6))
	// read key_001 and key_002
	itr.Next()
	itr.Next()
	itr.Close()
	s3.Done()

	// simulate tx4 before committing tx2 and tx3. Modifies a key read by tx3
	s4, _ := txMgr.NewTxSimulator("test_tx4")
	s4.DeleteState(cID, createTestKey(5))
	s4.Done()

	// validate and commit RWset for tx4
	txRWSet4, _ := s4.GetTxSimulationResults()
	txMgrHelper.validateAndCommitRWSet(txRWSet4.PubSimulationResults)

	//RWSet tx3 should be invalid now
	txRWSet3, _ := s3.GetTxSimulationResults()
	txMgrHelper.checkRWsetInvalid(txRWSet3.PubSimulationResults)

	// tx2 should still be valid
	txRWSet2, _ := s2.GetTxSimulationResults()
	txMgrHelper.validateAndCommitRWSet(txRWSet2.PubSimulationResults)

}

func TestGetSetMultipeKeys(t *testing.T) {
	for _, testEnv := range testEnvs {
		t.Logf("Running test for TestEnv = %s", testEnv.getName())
		testLedgerID := "testgetsetmultipekeys"
		testEnv.init(t, testLedgerID, nil)
		testGetSetMultipeKeys(t, testEnv)
		testEnv.cleanup()
	}
}

func testGetSetMultipeKeys(t *testing.T, env testEnv) {
	cID := "cid"
	txMgr := env.getTxMgr()
	txMgrHelper := newTxMgrTestHelper(t, txMgr)
	// simulate tx1
	s1, _ := txMgr.NewTxSimulator("test_tx1")
	multipleKeyMap := make(map[string][]byte)
	for i := 1; i <= 10; i++ {
		k := createTestKey(i)
		v := createTestValue(i)
		multipleKeyMap[k] = v
	}
	s1.SetStateMultipleKeys(cID, multipleKeyMap)
	s1.Done()
	// validate and commit RWset
	txRWSet, _ := s1.GetTxSimulationResults()
	txMgrHelper.validateAndCommitRWSet(txRWSet.PubSimulationResults)
	qe, _ := txMgr.NewQueryExecutor("test_tx2")
	defer qe.Done()
	multipleKeys := []string{}
	for k := range multipleKeyMap {
		multipleKeys = append(multipleKeys, k)
	}
	values, _ := qe.GetStateMultipleKeys(cID, multipleKeys)
	testutil.AssertEquals(t, len(values), 10)
	for i, v := range values {
		testutil.AssertEquals(t, v, multipleKeyMap[multipleKeys[i]])
	}

	s2, _ := txMgr.NewTxSimulator("test_tx3")
	defer s2.Done()
	values, _ = s2.GetStateMultipleKeys(cID, multipleKeys[5:7])
	testutil.AssertEquals(t, len(values), 2)
	for i, v := range values {
		testutil.AssertEquals(t, v, multipleKeyMap[multipleKeys[i+5]])
	}
}

func createTestKey(i int) string {
	if i == 0 {
		return ""
	}
	return fmt.Sprintf("key_%03d", i)
}

func createTestValue(i int) []byte {
	return []byte(fmt.Sprintf("value_%03d", i))
}

//TestExecuteQueryQuery is only tested on the CouchDB testEnv
func TestExecuteQuery(t *testing.T) {

	for _, testEnv := range testEnvs {
		// Query is only supported and tested on the CouchDB testEnv
		if testEnv.getName() == couchDBtestEnvName {
			t.Logf("Running test for TestEnv = %s", testEnv.getName())
			testLedgerID := "testexecutequery"
			testEnv.init(t, testLedgerID, nil)
			testExecuteQuery(t, testEnv)
			testEnv.cleanup()
		}
	}
}

func testExecuteQuery(t *testing.T, env testEnv) {

	type Asset struct {
		ID        string `json:"_id"`
		Rev       string `json:"_rev"`
		AssetName string `json:"asset_name"`
		Color     string `json:"color"`
		Size      string `json:"size"`
		Owner     string `json:"owner"`
	}

	txMgr := env.getTxMgr()
	txMgrHelper := newTxMgrTestHelper(t, txMgr)

	s1, _ := txMgr.NewTxSimulator("test_tx1")

	s1.SetState("ns1", "key1", []byte("value1"))
	s1.SetState("ns1", "key2", []byte("value2"))
	s1.SetState("ns1", "key3", []byte("value3"))
	s1.SetState("ns1", "key4", []byte("value4"))
	s1.SetState("ns1", "key5", []byte("value5"))
	s1.SetState("ns1", "key6", []byte("value6"))
	s1.SetState("ns1", "key7", []byte("value7"))
	s1.SetState("ns1", "key8", []byte("value8"))

	s1.SetState("ns1", "key9", []byte(`{"asset_name":"marble1","color":"red","size":"25","owner":"jerry"}`))
	s1.SetState("ns1", "key10", []byte(`{"asset_name":"marble2","color":"blue","size":"10","owner":"bob"}`))
	s1.SetState("ns1", "key11", []byte(`{"asset_name":"marble3","color":"blue","size":"35","owner":"jerry"}`))
	s1.SetState("ns1", "key12", []byte(`{"asset_name":"marble4","color":"green","size":"15","owner":"bob"}`))
	s1.SetState("ns1", "key13", []byte(`{"asset_name":"marble5","color":"red","size":"35","owner":"jerry"}`))
	s1.SetState("ns1", "key14", []byte(`{"asset_name":"marble6","color":"blue","size":"25","owner":"bob"}`))

	s1.Done()

	// validate and commit RWset
	txRWSet, _ := s1.GetTxSimulationResults()
	txMgrHelper.validateAndCommitRWSet(txRWSet.PubSimulationResults)

	queryExecuter, _ := txMgr.NewQueryExecutor("test_tx2")
	queryString := "{\"selector\":{\"owner\": {\"$eq\": \"bob\"}},\"limit\": 10,\"skip\": 0}"

	itr, err := queryExecuter.ExecuteQuery("ns1", queryString)
	testutil.AssertNoError(t, err, "Error upon ExecuteQuery()")
	counter := 0
	for {
		queryRecord, _ := itr.Next()
		if queryRecord == nil {
			break
		}
		//Unmarshal the document to Asset structure
		assetResp := &Asset{}
		json.Unmarshal(queryRecord.(*queryresult.KV).Value, &assetResp)
		//Verify the owner retrieved matches
		testutil.AssertEquals(t, assetResp.Owner, "bob")
		counter++
	}
	//Ensure the query returns 3 documents
	testutil.AssertEquals(t, counter, 3)
}

func TestValidateKey(t *testing.T) {
	nonUTF8Key := string([]byte{0xff, 0xff})
	dummyValue := []byte("dummyValue")
	for _, testEnv := range testEnvs {
		testLedgerID := "test.validate.key"
		testEnv.init(t, testLedgerID, nil)
		txSimulator, _ := testEnv.getTxMgr().NewTxSimulator("test_tx1")
		err := txSimulator.SetState("ns1", nonUTF8Key, dummyValue)
		if testEnv.getName() == levelDBtestEnvName {
			testutil.AssertNoError(t, err, "")
		}
		if testEnv.getName() == couchDBtestEnvName {
			testutil.AssertError(t, err, "")
		}
		testEnv.cleanup()
	}
}

// TestTxSimulatorUnsupportedTx verifies that a simulation must throw an error when an unsupported transaction
// is perfromed - queries on private data are supported in a read-only tran
func TestTxSimulatorUnsupportedTx(t *testing.T) {
	testEnv := testEnvs[0]
	testEnv.init(t, "TestTxSimulatorUnsupportedTxQueries", nil)
	defer testEnv.cleanup()
	txMgr := testEnv.getTxMgr()
	populateCollConfigForTest(t, txMgr.(*LockBasedTxMgr),
		[]collConfigkey{
			{"ns1", "coll1"},
			{"ns1", "coll2"},
			{"ns1", "coll3"},
			{"ns1", "coll4"},
		},
		version.NewHeight(1, 1))

	simulator, _ := txMgr.NewTxSimulator("txid1")
	err := simulator.SetState("ns", "key", []byte("value"))
	testutil.AssertNoError(t, err, "")
	_, err = simulator.GetPrivateDataRangeScanIterator("ns1", "coll1", "startKey", "endKey")
	_, ok := err.(*txmgr.ErrUnsupportedTransaction)
	testutil.AssertEquals(t, ok, true)

	simulator, _ = txMgr.NewTxSimulator("txid2")
	_, err = simulator.GetPrivateDataRangeScanIterator("ns1", "coll1", "startKey", "endKey")
	testutil.AssertNoError(t, err, "")
	err = simulator.SetState("ns", "key", []byte("value"))
	_, ok = err.(*txmgr.ErrUnsupportedTransaction)
	testutil.AssertEquals(t, ok, true)
}

func TestTxSimulatorMissingPvtdata(t *testing.T) {
	testEnv := testEnvs[0]
	testEnv.init(t, "TestTxSimulatorUnsupportedTxQueries", nil)
	defer testEnv.cleanup()

	txMgr := testEnv.getTxMgr()
	populateCollConfigForTest(t, txMgr.(*LockBasedTxMgr),
		[]collConfigkey{
			{"ns1", "coll1"},
			{"ns1", "coll2"},
			{"ns1", "coll3"},
			{"ns1", "coll4"},
		},
		version.NewHeight(1, 1),
	)

	db := testEnv.getVDB()
	updateBatch := privacyenabledstate.NewUpdateBatch()
	updateBatch.HashUpdates.Put("ns1", "coll1", util.ComputeStringHash("key1"), util.ComputeStringHash("value1"), version.NewHeight(1, 1))
	updateBatch.PvtUpdates.Put("ns1", "coll1", "key1", []byte("value1"), version.NewHeight(1, 1))
	db.ApplyPrivacyAwareUpdates(updateBatch, version.NewHeight(1, 1))

	simulator, _ := txMgr.NewTxSimulator("testTxid1")
	val, _ := simulator.GetPrivateData("ns1", "coll1", "key1")
	testutil.AssertEquals(t, val, []byte("value1"))
	simulator.Done()

	updateBatch = privacyenabledstate.NewUpdateBatch()
	updateBatch.HashUpdates.Put("ns1", "coll1", util.ComputeStringHash("key1"), util.ComputeStringHash("value1"), version.NewHeight(2, 1))
	updateBatch.HashUpdates.Put("ns1", "coll2", util.ComputeStringHash("key2"), util.ComputeStringHash("value2"), version.NewHeight(2, 1))
	updateBatch.HashUpdates.Put("ns1", "coll3", util.ComputeStringHash("key3"), util.ComputeStringHash("value3"), version.NewHeight(2, 1))
	updateBatch.PvtUpdates.Put("ns1", "coll3", "key3", []byte("value3"), version.NewHeight(2, 1))
	db.ApplyPrivacyAwareUpdates(updateBatch, version.NewHeight(2, 1))
	simulator, _ = txMgr.NewTxSimulator("testTxid2")
	defer simulator.Done()

	val, err := simulator.GetPrivateData("ns1", "coll1", "key1")
	_, ok := err.(*txmgr.ErrPvtdataNotAvailable)
	testutil.AssertEquals(t, ok, true)

	val, err = simulator.GetPrivateData("ns1", "coll2", "key2")
	_, ok = err.(*txmgr.ErrPvtdataNotAvailable)
	testutil.AssertEquals(t, ok, true)

	val, err = simulator.GetPrivateData("ns1", "coll3", "key3")
	testutil.AssertEquals(t, val, []byte("value3"))

	val, err = simulator.GetPrivateData("ns1", "coll4", "key4")
	testutil.AssertNoError(t, err, "")
	testutil.AssertNil(t, val)
}

func TestDeleteOnCursor(t *testing.T) {
	cID := "cid"
	env := testEnvs[0]
	env.init(t, "TestDeleteOnCursor", nil)
	defer env.cleanup()

	txMgr := env.getTxMgr()
	txMgrHelper := newTxMgrTestHelper(t, txMgr)

	// Simulate and commit tx1 to populate sample data (key_001 through key_010)
	s, _ := txMgr.NewTxSimulator("test_tx1")
	for i := 1; i <= 10; i++ {
		k := createTestKey(i)
		v := createTestValue(i)
		t.Logf("Adding k=[%s], v=[%s]", k, v)
		s.SetState(cID, k, v)
	}
	s.Done()
	txRWSet1, _ := s.GetTxSimulationResults()
	txMgrHelper.validateAndCommitRWSet(txRWSet1.PubSimulationResults)

	// simulate and commit tx2 that reads keys key_001 through key_004 and deletes them one by one (in a loop - itr.Next() followed by Delete())
	s2, _ := txMgr.NewTxSimulator("test_tx2")
	itr2, _ := s2.GetStateRangeScanIterator(cID, createTestKey(1), createTestKey(5))
	for i := 1; i <= 4; i++ {
		kv, err := itr2.Next()
		testutil.AssertNoError(t, err, "")
		testutil.AssertNotNil(t, kv)
		key := kv.(*queryresult.KV).Key
		s2.DeleteState(cID, key)
	}
	itr2.Close()
	s2.Done()
	txRWSet2, _ := s2.GetTxSimulationResults()
	txMgrHelper.validateAndCommitRWSet(txRWSet2.PubSimulationResults)

	// simulate tx3 to verify that the keys key_001 through key_004 got deleted
	s3, _ := txMgr.NewTxSimulator("test_tx3")
	itr3, _ := s3.GetStateRangeScanIterator(cID, createTestKey(1), createTestKey(10))
	kv, err := itr3.Next()
	testutil.AssertNoError(t, err, "")
	testutil.AssertNotNil(t, kv)
	key := kv.(*queryresult.KV).Key
	testutil.AssertEquals(t, key, "key_005")
	itr3.Close()
	s3.Done()
}

func TestTxSimulatorMissingPvtdataExpiry(t *testing.T) {
	ledgerid := "TestTxSimulatorMissingPvtdataExpiry"
	testEnv := testEnvs[0]
	cs := btltestutil.NewMockCollectionStore()
	cs.SetBTL("ns", "coll", 1)
	testEnv.init(t, ledgerid, pvtdatapolicy.ConstructBTLPolicy(cs))
	defer testEnv.cleanup()

	txMgr := testEnv.getTxMgr()
	populateCollConfigForTest(t, txMgr.(*LockBasedTxMgr), []collConfigkey{{"ns", "coll"}}, version.NewHeight(1, 1))

	viper.Set(fmt.Sprintf("ledger.pvtdata.btlpolicy.%s.ns.coll", ledgerid), 1)
	bg, _ := testutil.NewBlockGenerator(t, ledgerid, false)

	blkAndPvtdata := prepareNextBlockForTest(t, txMgr, bg, "txid-1",
		map[string]string{"pubkey1": "pub-value1"}, map[string]string{"pvtkey1": "pvt-value1"})
	testutil.AssertNoError(t, txMgr.ValidateAndPrepare(blkAndPvtdata, true), "")
	testutil.AssertNoError(t, txMgr.Commit(), "")

	simulator, _ := txMgr.NewTxSimulator("tx-tmp")
	pvtval, err := simulator.GetPrivateData("ns", "coll", "pvtkey1")
	testutil.AssertNoError(t, err, "")
	testutil.AssertEquals(t, pvtval, []byte("pvt-value1"))
	simulator.Done()

	blkAndPvtdata = prepareNextBlockForTest(t, txMgr, bg, "txid-2",
		map[string]string{"pubkey1": "pub-value2"}, map[string]string{"pvtkey2": "pvt-value2"})
	testutil.AssertNoError(t, txMgr.ValidateAndPrepare(blkAndPvtdata, true), "")
	testutil.AssertNoError(t, txMgr.Commit(), "")

	simulator, _ = txMgr.NewTxSimulator("tx-tmp")
	pvtval, _ = simulator.GetPrivateData("ns", "coll", "pvtkey1")
	testutil.AssertEquals(t, pvtval, []byte("pvt-value1"))
	simulator.Done()

	blkAndPvtdata = prepareNextBlockForTest(t, txMgr, bg, "txid-2",
		map[string]string{"pubkey1": "pub-value3"}, map[string]string{"pvtkey3": "pvt-value3"})
	testutil.AssertNoError(t, txMgr.ValidateAndPrepare(blkAndPvtdata, true), "")
	testutil.AssertNoError(t, txMgr.Commit(), "")

	simulator, _ = txMgr.NewTxSimulator("tx-tmp")
	pvtval, _ = simulator.GetPrivateData("ns", "coll", "pvtkey1")
	testutil.AssertNil(t, pvtval)
	simulator.Done()
}

func prepareNextBlockForTest(t *testing.T, txMgr txmgr.TxMgr, bg *testutil.BlockGenerator,
	txid string, pubKVs map[string]string, pvtKVs map[string]string) *ledger.BlockAndPvtData {
	simulator, _ := txMgr.NewTxSimulator(txid)
	//simulating transaction
	for k, v := range pubKVs {
		simulator.SetState("ns", k, []byte(v))
	}
	for k, v := range pvtKVs {
		simulator.SetPrivateData("ns", "coll", k, []byte(v))
	}
	simulator.Done()
	simRes, _ := simulator.GetTxSimulationResults()
	pubSimBytes, _ := simRes.GetPubSimulationBytes()
	block := bg.NextBlock([][]byte{pubSimBytes})
	return &ledger.BlockAndPvtData{Block: block,
		BlockPvtData: map[uint64]*ledger.TxPvtData{0: {SeqInBlock: 0, WriteSet: simRes.PvtSimulationResults}},
	}
}
