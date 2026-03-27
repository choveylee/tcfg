/**
 * @Author: lidonglin
 * @Description:
 * @File:  env.go
 * @Version: 1.0.0
 * @Date: 2022/11/03 10:35
 */

package tcfg

import (
	"os"
	"strconv"
	"strings"
	"time"
)

// EnvData is a zero-value-friendly helper backed by os.LookupEnv.
type EnvData struct {
}

// GetBool parses a boolean from the environment; ok is false when the variable is unset.
func (p *EnvData) GetBool(key string) (bool, bool, error) {
	val, ok := p.getData(key)
	if !ok {
		return false, false, nil
	}

	ret, err := parseBool(val)

	return ret, true, err
}

// Bool returns GetBool's value or false when unset.
func (p *EnvData) Bool(key string) (bool, error) {
	val, _, err := p.GetBool(key)

	return val, err
}

// DefaultBool returns defaultVal when the variable is missing or invalid.
func (p *EnvData) DefaultBool(key string, defaultVal bool) bool {
	val, ok, err := p.GetBool(key)
	if !ok || err != nil {
		return defaultVal
	}

	return val
}

// GetInt parses a decimal int from the environment; ok is false when unset.
func (p *EnvData) GetInt(key string) (int, bool, error) {
	val, ok := p.getData(key)
	if !ok {
		return 0, false, nil
	}

	ret, err := strconv.Atoi(val)

	return ret, true, err
}

// Int is like GetInt but returns 0 when unset.
func (p *EnvData) Int(key string) (int, error) {
	val, _, err := p.GetInt(key)

	return val, err
}

// DefaultInt returns defaultVal when unset or invalid.
func (p *EnvData) DefaultInt(key string, defaultVal int) int {
	val, ok, err := p.GetInt(key)
	if !ok || err != nil {
		return defaultVal
	}

	return val
}

// GetInt64 parses a base-10 int64; ok is false when unset.
func (p *EnvData) GetInt64(key string) (int64, bool, error) {
	val, ok := p.getData(key)
	if !ok {
		return 0, false, nil
	}

	ret, err := strconv.ParseInt(val, 10, 64)

	return ret, true, err
}

// Int64 is like GetInt64 but returns 0 when unset.
func (p *EnvData) Int64(key string) (int64, error) {
	val, _, err := p.GetInt64(key)

	return val, err
}

// DefaultInt64 returns defaultVal when unset or invalid.
func (p *EnvData) DefaultInt64(key string, defaultVal int64) int64 {
	val, ok, err := p.GetInt64(key)
	if !ok || err != nil {
		return defaultVal
	}

	return val
}

// GetFloat parses float64; ok is false when unset.
func (p *EnvData) GetFloat(key string) (float64, bool, error) {
	val, ok := p.getData(key)
	if !ok {
		return 0, false, nil
	}

	ret, err := strconv.ParseFloat(val, 64)

	return ret, true, err
}

// Float is like GetFloat but returns 0 when unset.
func (p *EnvData) Float(key string) (float64, error) {
	val, _, err := p.GetFloat(key)

	return val, err
}

// DefaultFloat returns defaultVal when unset or invalid.
func (p *EnvData) DefaultFloat(key string, defaultVal float64) float64 {
	val, ok, err := p.GetFloat(key)
	if !ok || err != nil {
		return defaultVal
	}

	return val
}

// GetDuration parses time.ParseDuration; ok is false when unset.
func (p *EnvData) GetDuration(key string) (time.Duration, bool, error) {
	val, ok := p.getData(key)
	if !ok {
		return time.Duration(0), false, nil
	}

	ret, err := time.ParseDuration(val)

	return ret, true, err
}

// Duration is like GetDuration but returns 0 when unset.
func (p *EnvData) Duration(key string) (time.Duration, error) {
	val, _, err := p.GetDuration(key)

	return val, err
}

// DefaultDuration returns defaultVal when unset or invalid.
func (p *EnvData) DefaultDuration(key string, defaultVal time.Duration) time.Duration {
	val, ok, err := p.GetDuration(key)
	if !ok || err != nil {
		return defaultVal
	}

	return val
}

// GetString returns the raw environment value (after SECTION::KEY rewrite to KEY_SECTION).
func (p *EnvData) GetString(key string) (string, bool) {
	return p.getData(key)
}

// String returns GetString's value or "" when unset.
func (p *EnvData) String(key string) string {
	val, _ := p.GetString(key)

	return val
}

// DefaultString returns defaultVal when unset.
func (p *EnvData) DefaultString(key string, defaultVal string) string {
	val, ok := p.GetString(key)
	if !ok {
		return defaultVal
	}

	return val
}

// GetStrings splits the env value by sep; ok is false when unset.
func (p *EnvData) GetStrings(key string, sep string) ([]string, bool) {
	vals, ok := p.GetString(key)
	if !ok {
		return nil, false
	}

	ret := strings.Split(vals, sep)

	return ret, true
}

// Strings returns GetStrings' slice or nil when unset.
func (p *EnvData) Strings(key string, sep string) []string {
	vals, _ := p.GetStrings(key, sep)

	return vals
}

// DefaultStrings returns defaultVals when unset.
func (p *EnvData) DefaultStrings(key string, sep string, defaultVals []string) []string {
	vals, ok := p.GetStrings(key, sep)
	if !ok {
		return defaultVals
	}

	return vals
}

// getData maps SECTION::NAME to the env name NAME_SECTION (single-colon section form).
func (p *EnvData) getData(key string) (string, bool) {
	params := strings.Split(key, "::")

	if len(params) == 2 {
		key = params[1] + "_" + params[0]
	}

	return os.LookupEnv(key)
}
