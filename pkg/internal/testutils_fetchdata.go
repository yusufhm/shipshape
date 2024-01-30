package internal

import (
	"reflect"
	"testing"

	"github.com/salsadigitalauorg/shipshape/pkg/config"
	"github.com/salsadigitalauorg/shipshape/pkg/result"
	"github.com/stretchr/testify/assert"
)

// FetchDataTest can be used to create test scenarios using test tables,
// for the FetchData method using TestFetchData below.
type FetchDataTest struct {
	// Name of the test.
	Name  string
	Check config.Check
	// Initialise the check before testing.
	Init bool
	// Func to run before running the check
	PreRun func(t *testing.T)
	// Expected values after running the check.
	ExpectPasses     []string
	ExpectBreaches   []result.Breach
	ExpectStatusFail bool
	ExpectStatusPass bool
	ExpectDataMap    map[string][]byte
}

// TestFetchData can be used to run test scenarios in test tables.
func TestFetchData(t *testing.T, ctest FetchDataTest) {
	t.Helper()
	assert := assert.New(t)
	ctest.Check.FetchData()

	r := ctest.Check.GetResult()

	if ctest.ExpectStatusFail {
		assert.Equal(result.Fail, r.Status)
	} else if ctest.ExpectStatusPass {
		assert.Equal(result.Pass, r.Status)
	} else {
		assert.NotEqual(result.Fail, r.Status)
		assert.NotEqual(result.Pass, r.Status)
	}

	if len(ctest.ExpectPasses) > 0 {
		assert.ElementsMatch(ctest.ExpectPasses, r.Passes)
	} else {
		assert.Empty(r.Passes)
	}

	if len(ctest.ExpectBreaches) > 0 {
		assert.ElementsMatch(ctest.ExpectBreaches, r.Breaches)
	} else {
		assert.Empty(r.Breaches)
	}

	if ctest.ExpectDataMap != nil {
		dataMap := reflect.ValueOf(ctest.Check).Elem().FieldByName("DataMap").Interface().(map[string][]byte)
		assert.EqualValues(ctest.ExpectDataMap, dataMap)
	}
}
