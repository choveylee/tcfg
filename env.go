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

type EnvData struct {
}

func (p *EnvData) GetBool(key string) (bool, bool, error) {
	val, ok := p.getData(key)
	if ok == false {
		return false, false, nil
	}

	ret, err := parseBool(val)

	return ret, true, err
}

func (p *EnvData) Bool(key string) (bool, error) {
	val, _, err := p.GetBool(key)

	return val, err
}

func (p *EnvData) DefaultBool(key string, defaultVal bool) bool {
	val, ok, err := p.GetBool(key)
	if ok == false || err != nil {
		return defaultVal
	}

	return val
}

func (p *EnvData) GetInt(key string) (int, bool, error) {
	val, ok := p.getData(key)
	if ok == false {
		return 0, false, nil
	}

	ret, err := strconv.Atoi(val)

	return ret, true, err
}

func (p *EnvData) Int(key string) (int, error) {
	val, _, err := p.GetInt(key)

	return val, err
}

func (p *EnvData) DefaultInt(key string, defaultVal int) int {
	val, ok, err := p.GetInt(key)
	if ok == false || err != nil {
		return defaultVal
	}

	return val
}

func (p *EnvData) GetInt64(key string) (int64, bool, error) {
	val, ok := p.getData(key)
	if ok == false {
		return 0, false, nil
	}

	ret, err := strconv.ParseInt(val, 10, 64)

	return ret, true, err
}

func (p *EnvData) Int64(key string) (int64, error) {
	val, _, err := p.GetInt64(key)

	return val, err
}

func (p *EnvData) DefaultInt64(key string, defaultVal int64) int64 {
	val, ok, err := p.GetInt64(key)
	if ok == false || err != nil {
		return defaultVal
	}

	return val
}

func (p *EnvData) GetFloat(key string) (float64, bool, error) {
	val, ok := p.getData(key)
	if ok == false {
		return 0, false, nil
	}

	ret, err := strconv.ParseFloat(val, 64)

	return ret, true, err
}

func (p *EnvData) Float(key string) (float64, error) {
	val, _, err := p.GetFloat(key)

	return val, err
}

func (p *EnvData) DefaultFloat(key string, defaultVal float64) float64 {
	val, ok, err := p.GetFloat(key)
	if ok == false || err != nil {
		return defaultVal
	}

	return val
}

func (p *EnvData) GetDuration(key string) (time.Duration, bool, error) {
	val, ok := p.getData(key)
	if ok == false {
		return time.Duration(0), false, nil
	}

	ret, err := time.ParseDuration(val)

	return ret, true, err
}

func (p *EnvData) Duration(key string) (time.Duration, error) {
	val, _, err := p.GetDuration(key)

	return val, err
}

func (p *EnvData) DefaultDuration(key string, defaultVal time.Duration) time.Duration {
	val, ok, err := p.GetDuration(key)
	if ok == false || err != nil {
		return defaultVal
	}

	return val
}

func (p *EnvData) GetString(key string) (string, bool) {
	return p.getData(key)
}

func (p *EnvData) String(key string) string {
	val, _ := p.GetString(key)

	return val
}

func (p *EnvData) DefaultString(key string, defaultVal string) string {
	val, ok := p.GetString(key)
	if ok == false {
		return defaultVal
	}

	return val
}

func (p *EnvData) GetStrings(key string, sep string) ([]string, bool) {
	vals, ok := p.GetString(key)
	if ok == false {
		return nil, false
	}

	ret := strings.Split(vals, sep)

	return ret, true
}

func (p *EnvData) Strings(key string, sep string) []string {
	vals, _ := p.GetStrings(key, sep)

	return vals
}

func (p *EnvData) DefaultStrings(key string, sep string, defaultVals []string) []string {
	vals, ok := p.GetStrings(key, sep)
	if ok == false {
		return defaultVals
	}

	return vals
}

func (p *EnvData) getData(key string) (string, bool) {
	params := strings.Split(key, "::")

	if len(params) == 2 {
		key = params[1] + "_" + params[0]
	}

	return os.LookupEnv(key)
}
