/**
 * @Author: lidonglin
 * @Description:
 * @File:  tcfg_test.go
 * @Version: 1.0.0
 * @Date: 2022/11/05 21:32
 */

package tcfg

import (
	"os"
	"strings"
	"testing"
	"time"

	"github.com/choveylee/terror"
	"github.com/stretchr/testify/assert"
)

func TestTryGetEnv(t *testing.T) {
	err := os.Setenv("ENV_STRING_A", "evn_string_a")
	assert.Equal(t, err, nil)

	err = os.Setenv("ENV_STRING_B", "evn_string_b")
	assert.Equal(t, err, nil)

	err = os.Setenv("ENV_STRING_C_DEV", "evn_string_c")
	assert.Equal(t, err, nil)

	stringC, err := String("DEV::ENV_STRING_C")
	assert.Equal(t, stringC, "evn_string_c")
	assert.Equal(t, err, nil)

	stringA, err := String("ENV_STRING_A")
	assert.Equal(t, stringA, "evn_string_a")
	assert.Equal(t, err, nil)

	stringB, err := String("ENV_STRING_B")
	assert.Equal(t, stringB, "evn_string_b")
	assert.Equal(t, err, nil)
}

func TestTryGetLocal(t *testing.T) {
	stringA, err := String("STRING_A")
	assert.Equal(t, stringA, "string_a")
	assert.Equal(t, err, nil)

	testStringA, err := String("TEST_STRING_A")
	assert.Equal(t, testStringA, "test_string_a")
	assert.Equal(t, err, nil)

	stringB, err := String("STRING_B")
	assert.Equal(t, stringB, "val_a_val_b_${TMP_VAL_F}")
	assert.Equal(t, err, terror.ErrDataNotExist("STRING_B"))

	testStringC, err := String("TCFG_STRING_C")
	assert.Equal(t, testStringC, "string_c")
	assert.Equal(t, err, nil)

	stringE, err := String("STRING_E")
	stringERets := map[string]string{
		"val_a:val_c1@val_d1:3306": "",
		"val_a:val_c2@val_d1:3306": "",
		"val_a:val_c3@val_d1:3306": "",
		"val_a:val_c1@val_d2:3306": "",
		"val_a:val_c2@val_d2:3306": "",
		"val_a:val_c3@val_d2:3306": "",
	}
	stringEVals := strings.Split(stringE, ",")
	for _, stringEVal := range stringEVals {
		_, ok := stringERets[stringEVal]
		assert.Equal(t, ok, true)
	}
	assert.Equal(t, err, nil)

	stringF := DefaultString("STRING_F", "rep_string")
	assert.Equal(t, stringF, `string_f
string_f`)

	stringZ, err := String("STRING_Z")
	assert.Equal(t, stringZ, "")
	assert.Equal(t, err.Error(), terror.ErrDataNotExist("STRING_Z").Error())

	stringC, err := String("DEV::STRING_C")
	assert.Equal(t, stringC, "dev_string_c")
	assert.Equal(t, err, nil)

	boolA, err := Bool("BOOL_A")
	assert.Equal(t, boolA, true)
	assert.Equal(t, err, nil)

	boolB, err := Bool("BOOL_B")
	assert.Equal(t, boolB, false)
	assert.NotEqual(t, err, nil)

	boolZ, err := Bool("BOOL_Z")
	assert.Equal(t, boolZ, false)
	assert.Equal(t, err.Error(), terror.ErrDataNotExist("BOOL_Z").Error())

	intA, err := Int("INT_A")
	assert.Equal(t, intA, 1)
	assert.Equal(t, err, nil)

	intB, err := Int("INT_B")
	assert.Equal(t, intB, 0)
	assert.NotEqual(t, err, nil)

	intZ, err := Int("INT_Z")
	assert.Equal(t, intZ, 0)
	assert.Equal(t, err.Error(), terror.ErrDataNotExist("INT_Z").Error())

	int64A, err := Int64("INT64_A")
	assert.Equal(t, int64A, int64(2))
	assert.Equal(t, err, nil)

	int64B, err := Int64("INT64_B")
	assert.Equal(t, int64B, int64(0))
	assert.NotEqual(t, err, nil)

	int64Z, err := Int64("INT64_Z")
	assert.Equal(t, int64Z, int64(0))
	assert.Equal(t, err.Error(), terror.ErrDataNotExist("INT64_Z").Error())

	floatA, err := Float64("FLOAT_A")
	assert.Equal(t, floatA, 3.4)
	assert.Equal(t, err, nil)

	floatB, err := Float64("FLOAT_B")
	assert.Equal(t, floatB, float64(0))
	assert.NotEqual(t, err, nil)

	floatZ, err := Float64("FLOAT_Z")
	assert.Equal(t, floatZ, float64(0))
	assert.Equal(t, err.Error(), terror.ErrDataNotExist("FLOAT_Z").Error())

	durationA, err := Duration("DURATION_A")
	assert.Equal(t, durationA, 3*time.Minute)
	assert.Equal(t, err, nil)

	durationB, err := Duration("DURATION_B")
	assert.Equal(t, durationB, time.Duration(0))
	assert.NotEqual(t, err, nil)

	durationZ, err := Duration("DURATION_Z")
	assert.Equal(t, durationZ, time.Duration(0))
	assert.Equal(t, err.Error(), terror.ErrDataNotExist("DURATION_Z").Error())

	stringsA, err := Strings("STRINGS_A", ",")
	assert.Equal(t, stringsA, []string{"strings_a1", "strings_a2"})
	assert.Equal(t, err, nil)

	stringsD, err := Strings("STRINGS_D", ",")
	assert.Equal(t, stringsD, []string{"string_a", "2"})
	assert.Equal(t, err, nil)

	stringsZ, err := Strings("STRINGS_Z", ",")
	assert.Equal(t, stringsZ, []string(nil))
	assert.Equal(t, err.Error(), terror.ErrDataNotExist("STRINGS_Z").Error())
}

func TestGetLocal(t *testing.T) {
	stringA := DefaultString("STRING_A", "rep_string")
	assert.Equal(t, stringA, "string_a")

	stringC := DefaultString("DEV::STRING_C", "rep_string")
	assert.Equal(t, stringC, "dev_string_c")

	stringZ := DefaultString("STRING_Z", "rep_string")
	assert.Equal(t, stringZ, "rep_string")

	stringY := DefaultString("DEV::STRING_Y", "rep_string")
	assert.Equal(t, stringY, "rep_string")

	boolA := DefaultBool("BOOL_A", true)
	assert.Equal(t, boolA, true)

	boolB := DefaultBool("BOOL_B", true)
	assert.Equal(t, boolB, true)

	boolZ := DefaultBool("BOOL_Z", true)
	assert.Equal(t, boolZ, true)

	intA := DefaultInt("INT_A", -1)
	assert.Equal(t, intA, 1)

	intB := DefaultInt("INT_B", -1)
	assert.Equal(t, intB, -1)

	intZ := DefaultInt("INT_Z", -1)
	assert.Equal(t, intZ, -1)

	int64A := DefaultInt64("INT64_A", -2)
	assert.Equal(t, int64A, int64(2))

	int64B := DefaultInt64("INT64_B", -2)
	assert.Equal(t, int64B, int64(-2))

	int64Z := DefaultInt64("INT64_Z", -2)
	assert.Equal(t, int64Z, int64(-2))

	floatA := DefaultFloat64("FLOAT_A", -3.4)
	assert.Equal(t, floatA, 3.4)

	floatB := DefaultFloat64("FLOAT_B", -3.4)
	assert.Equal(t, floatB, -3.4)

	floatZ := DefaultFloat64("FLOAT_Z", -3.4)
	assert.Equal(t, floatZ, -3.4)

	durationA := DefaultDuration("DURATION_A", 5*time.Second)
	assert.Equal(t, durationA, 3*time.Minute)

	durationB := DefaultDuration("DURATION_B", 1*time.Minute)
	assert.Equal(t, durationB, 1*time.Minute)

	durationZ := DefaultDuration("DURATION_Z", 2*time.Minute)
	assert.Equal(t, durationZ, 2*time.Minute)

	stringsA := DefaultStrings("STRINGS_A", ",", []string{"rep_string_a1", "rep_string_a2"})
	assert.Equal(t, stringsA, []string{"strings_a1", "strings_a2"})

	stringsC := DefaultStrings("STRINGS_C", ",", []string{"rep_string_a1", "rep_string_a2"})
	assert.Equal(t, stringsC, []string{"rep_string_a1", "rep_string_a2"})
}

func TestTryGetGlobal(t *testing.T) {
	stringC, err := String("STRING_C")
	assert.Equal(t, stringC, "string_c")
	assert.Equal(t, err, nil)

	stringD, err := String("STRING_D")
	assert.Equal(t, stringD, "string_d_gl")
	assert.Equal(t, err, nil)
}
