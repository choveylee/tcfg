// Package tcfg loads INI files together with environment variables, resolves keys using
// an optional key prefix and APP_NAME-based scoping, and expands values with ${key} and
// $[key] placeholders (including $${...} / $$[...] escapes for literal $ characters).
//
// IniMgr/IniData handle file and in-memory INI parsing (sections, includes, UTF-8 BOM).
// EnvData reads os environment variables (SECTION::KEY maps to KEY_SECTION).
// The default ConfData instance is loaded in init from <exe_basename>_config.ini; package-level
// functions (String, Bool, …) delegate to that instance.

/**
 * @Author: lidonglin
 * @Description:
 * @File:  tcfg.go
 * @Version: 1.0.0
 * @Date: 2022/11/03 10:34
 */

package tcfg

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/choveylee/terror"
)

// Regular expressions used to find ${key}, $[key], and escaped $${...} / $$[...] spans in config values.
var (
	ValStringKeyMatchReg  = regexp.MustCompile(`\$\{(.+?)\}`)
	ValStringsKeyMatchReg = regexp.MustCompile(`\$\[(.+?)\]`)

	ValStringKeyReplaceReg = regexp.MustCompile(`\$\$\{(.+?)\}|\$\$\[(.+?)\]`)
)

// DefaultStringsSeparator is the default delimiter when splitting list values for $[key] expansion.
const (
	DefaultStringsSeparator = ","
)

// DefaultAppName is the configuration key used as the application name for scoped/local keys.
const (
	DefaultAppName = "APP_NAME"
)

// ErrNilConfData is returned when a *ConfData method is invoked with a nil receiver.
var ErrNilConfData = errors.New("tcfg: method called on nil *ConfData receiver")

var (
	keyPrefixMutex sync.RWMutex

	defaultKeyPrefix string
)

// GetKeyPrefix returns the key prefix configured by SetKeyPrefix. It is safe for concurrent use with SetKeyPrefix.
func GetKeyPrefix() string {
	keyPrefixMutex.RLock()
	defer keyPrefixMutex.RUnlock()

	return defaultKeyPrefix
}

// SetKeyPrefix sets the global key prefix applied when resolving keys (safe for concurrent use with GetKeyPrefix).
func SetKeyPrefix(keyPrefix string) {
	keyPrefixMutex.Lock()

	defaultKeyPrefix = keyPrefix

	keyPrefixMutex.Unlock()
}

// genConfName returns <executable_basename>_config.ini (lowercased, hyphen becomes underscore).
func genConfName() string {
	_, fileName := filepath.Split(os.Args[0])
	fileExt := filepath.Ext(os.Args[0])

	appName := strings.TrimSuffix(fileName, fileExt)
	appName = strings.ToLower(strings.ReplaceAll(appName, "-", "_"))

	configName := fmt.Sprintf("%s_config.ini", appName)

	return configName
}

// genConfPaths returns parent directories from configPath up to the filesystem root (excluding configPath itself).
func genConfPaths(configPath string) []string {
	configPaths := make([]string, 0)

	for {
		tmpConfigPath := filepath.Dir(configPath)
		if tmpConfigPath == configPath {
			break
		}

		configPath = tmpConfigPath
		configPaths = append(configPaths, configPath)
	}

	return configPaths
}

// analysisConfPath searches confPaths in order for confName; returns the first regular file path or ("", nil) if not found.
func analysisConfPath(confPaths []string, confName string) (string, error) {
	for _, confPath := range confPaths {
		retConfPath := filepath.Join(confPath, confName)

		file, err := os.Stat(retConfPath)
		if err == nil {
			if file.IsDir() {
				return "", terror.ErrConfIllegal(retConfPath)
			}

			return retConfPath, nil
		}

		if !os.IsNotExist(err) {
			return "", err
		}
	}

	return "", nil
}

// analysisKey builds uppercased lookup keys with optional prefix and APP_NAME sub-prefix (section::key supported).
// It returns (originalKey, baseKey) for fallback when scoped keys differ from unscoped.
func analysisKey(key string, prefix string, subPrefix string) (string, string) {
	key = strings.ToUpper(key)

	params := strings.Split(key, "::")

	if len(params) == 1 {
		if strings.HasPrefix(key, prefix) {
			key = strings.TrimPrefix(key, prefix)
		}

		tmpKey := key
		if strings.HasPrefix(tmpKey, subPrefix) {
			tmpKey = strings.TrimPrefix(tmpKey, subPrefix)
		}

		originalKey := prefix + key
		baseKey := prefix + tmpKey

		return originalKey, baseKey
	} else if len(params) > 1 {
		section := params[0]
		secKey := params[1]

		if strings.HasPrefix(secKey, prefix) {
			secKey = strings.TrimPrefix(secKey, prefix)
		}

		tmpKey := secKey
		if strings.HasPrefix(tmpKey, subPrefix) {
			tmpKey = strings.TrimPrefix(tmpKey, subPrefix)
		}

		originalKey := fmt.Sprintf("%s::%s%s", section, prefix, secKey)
		baseKey := fmt.Sprintf("%s::%s%s", section, prefix, tmpKey)

		return originalKey, baseKey
	}

	return "", ""
}

// ConfData holds parsed INI data and environment overrides; env is checked before INI for each key.
type ConfData struct {
	iniData *IniData

	envData *EnvData
}

// Configs is a JSON-friendly list of flat key/value entries (used by Response).
type Configs struct {
	Configs []*Config `json:"configs"`
}

// Response groups an error code, message, and optional Configs payload.
type Response struct {
	Error   int
	Message string

	Data *Configs `json:"data"`
}

// loadFromFile searches the working directory and executable directory (and parents) for configName and parses it.
func loadFromFile(configName string) (*IniData, error) {
	workPath, err := os.Getwd()
	if err != nil {
		return nil, err
	}

	appPath, err := filepath.Abs(filepath.Dir(os.Args[0]))
	if err != nil {
		return nil, err
	}

	extConfigPaths := make([]string, 0)

	workConfigPaths := genConfPaths(workPath)
	appConfigPaths := genConfPaths(appPath)

	extConfigPaths = append(extConfigPaths, workConfigPaths...)
	extConfigPaths = append(extConfigPaths, appConfigPaths...)

	// get config file from cmd work path > app path
	configPaths := []string{
		workPath,
		appPath,
	}

	configPaths = append(configPaths, extConfigPaths...)

	configPath, err := analysisConfPath(configPaths, configName)
	if err != nil {
		return nil, err
	}

	iniMgr := &IniMgr{}

	iniData, err := iniMgr.ParseFile(configPath)
	if err != nil {
		return nil, err
	}

	return iniData, nil
}

// defaultLoad initializes env storage and merges INI loaded from configName into p.
func (p *ConfData) defaultLoad(configName string) error {
	if p == nil {
		return ErrNilConfData
	}

	// init env
	envData := &EnvData{}

	p.envData = envData

	// load from file
	iniData, err := loadFromFile(configName)
	if err == nil {
		p.iniData = iniData

		return nil
	}

	return err
}

// analysisValue expands ${key}, Cartesian-expands $[key] list placeholders, then unescapes $${...}/$$[...].
// The bool reports whether any ${} or $[] substitution ran (step 3 may still run when false).
func (p *ConfData) analysisValue(val string) (string, bool, error) {
	if p == nil {
		return val, false, ErrNilConfData
	}

	isMatch := false

	retVal := ""

	// 1. replace all ${key}
	startIndex := 0

	retMatches := ValStringKeyMatchReg.FindAllStringIndex(val, -1)

	for _, retMatch := range retMatches {
		tmpStartIndex := retMatch[0]
		tmpEndIndex := retMatch[1]

		retVal += val[startIndex:tmpStartIndex]

		if tmpStartIndex > 0 {
			if val[tmpStartIndex-1] == '$' {

				retVal += val[tmpStartIndex:tmpEndIndex]

				startIndex = tmpEndIndex

				continue
			}
		}

		isMatch = true

		key := strings.TrimSpace(val[tmpStartIndex+2 : tmpEndIndex-1])

		realVal, ok, err := p.stringEx(key)
		if !ok {
			return val, isMatch, terror.ErrDataNotExist(key)
		}
		if err != nil {
			return val, isMatch, err
		}

		retVal += realVal
		startIndex = tmpEndIndex
	}

	retVal += val[startIndex:]

	// 2. replace all $[key]
	val = retVal
	retVal = ""

	retMatches = ValStringsKeyMatchReg.FindAllStringIndex(val, -1)

	matchKeysMap := make(map[string][]string)

	// 2.1 search all $[key]
	for _, retMatch := range retMatches {
		tmpStartIndex := retMatch[0]
		tmpEndIndex := retMatch[1]

		if tmpStartIndex > 0 {
			if val[tmpStartIndex-1] == '$' {
				continue
			}
		}

		isMatch = true

		matchKey := val[tmpStartIndex:tmpEndIndex]

		_, ok := matchKeysMap[matchKey]
		if ok {
			continue
		}

		key := strings.TrimSpace(val[tmpStartIndex+2 : tmpEndIndex-1])

		retVals, err := p.Strings(key, DefaultStringsSeparator)
		if err != nil {
			return val, isMatch, err
		}

		matchKeysMap[matchKey] = retVals
	}

	rets := []string{val}

	for matchKey, matchVals := range matchKeysMap {
		tmpRets := rets
		rets = make([]string, 0)

		for _, tmpRet := range tmpRets {
			for _, matchVal := range matchVals {
				tmpIndex := 0
				realRet := ""

				for {
					index := strings.Index(tmpRet[tmpIndex:], matchKey)
					if index == -1 {
						realRet += tmpRet[tmpIndex:]
						break
					}

					if (tmpIndex+index) > 0 && tmpRet[tmpIndex+index-1] == '$' {
						realRet += tmpRet[tmpIndex : tmpIndex+index]
						realRet += matchKey

						tmpIndex = tmpIndex + index + len(matchKey)

						continue
					}

					realRet += tmpRet[tmpIndex : tmpIndex+index]
					realRet += matchVal

					tmpIndex = tmpIndex + index + len(matchKey)
				}

				rets = append(rets, realRet)
			}
		}
	}
	retVal = strings.Join(rets, DefaultStringsSeparator)

	if isMatch {
		return retVal, isMatch, nil
	}

	// Step 3: turn escaped $${...} / $$[...] into literal ${...} / $[...] (strip one leading $; see ValStringKeyReplaceReg).
	startIndex = 0

	val = retVal
	retVal = ""

	retMatches = ValStringKeyReplaceReg.FindAllStringIndex(val, -1)

	for _, retMatch := range retMatches {
		tmpStartIndex := retMatch[0]
		tmpEndIndex := retMatch[1]

		retVal += val[startIndex:tmpStartIndex]
		retVal += val[tmpStartIndex+1 : tmpEndIndex]

		startIndex = tmpEndIndex
	}

	retVal += val[startIndex:]

	return retVal, isMatch, nil
}

// LocalKey builds keyPrefix + upper(APP_NAME) + "_" + key when APP_NAME is set; strips keyPrefix if already present.
func (p *ConfData) LocalKey(key string) string {
	appName, ok, err := p.string(DefaultAppName)
	if err == nil && ok {
		keyPrefix := GetKeyPrefix()

		if strings.HasPrefix(key, keyPrefix) {
			key = strings.TrimPrefix(key, keyPrefix)
		}

		appName = strings.ToUpper(strings.Replace(appName, "-", "_", -1))

		key = fmt.Sprintf("%s%s_%s", keyPrefix, appName, key)
	}

	return key
}

// Bool reads a string value and parses it as a boolean.
func (p *ConfData) Bool(key string) (bool, error) {
	val, err := p.String(key)
	if err != nil {
		return false, err
	}

	ret, err := parseBool(val)
	if err != nil {
		return false, err
	}

	return ret, nil
}

// DefaultBool is like Bool but returns defaultVal if the key is missing or parsing fails.
func (p *ConfData) DefaultBool(key string, defaultVal bool) bool {
	val, err := p.Bool(key)
	if err != nil {
		return defaultVal
	}

	return val
}

// Int reads and parses the value as a base-10 integer.
func (p *ConfData) Int(key string) (int, error) {
	val, err := p.String(key)
	if err != nil {
		return 0, err
	}

	ret, err := strconv.Atoi(val)
	if err != nil {
		return 0, err
	}

	return ret, nil
}

// DefaultInt is like Int but returns defaultVal on error or missing key.
func (p *ConfData) DefaultInt(key string, defaultVal int) int {
	val, err := p.Int(key)
	if err != nil {
		return defaultVal
	}

	return val
}

// Int32 reads and parses the value as a 32-bit integer.
func (p *ConfData) Int32(key string) (int32, error) {
	val, err := p.String(key)
	if err != nil {
		return 0, err
	}

	ret, err := strconv.ParseInt(val, 10, 32)
	if err != nil {
		return 0, err
	}

	return int32(ret), nil
}

// DefaultInt32 is like Int32 but returns defaultVal on error or missing key.
func (p *ConfData) DefaultInt32(key string, defaultVal int32) int32 {
	val, err := p.Int32(key)
	if err != nil {
		return defaultVal
	}

	return val
}

// Int64 reads and parses the value as a 64-bit integer.
func (p *ConfData) Int64(key string) (int64, error) {
	val, err := p.String(key)
	if err != nil {
		return 0, err
	}

	ret, err := strconv.ParseInt(val, 10, 64)
	if err != nil {
		return 0, err
	}

	return ret, nil
}

// DefaultInt64 is like Int64 but returns defaultVal on error or missing key.
func (p *ConfData) DefaultInt64(key string, defaultVal int64) int64 {
	val, err := p.Int64(key)
	if err != nil {
		return defaultVal
	}

	return val
}

// Float32 reads and parses the value as a 32-bit float.
func (p *ConfData) Float32(key string) (float32, error) {
	val, err := p.String(key)
	if err != nil {
		return 0, err
	}

	ret, err := strconv.ParseFloat(val, 32)
	if err != nil {
		return 0, err
	}

	return float32(ret), nil
}

// DefaultFloat32 is like Float32 but returns defaultVal on error or missing key.
func (p *ConfData) DefaultFloat32(key string, defaultVal float32) float32 {
	val, err := p.Float32(key)
	if err != nil {
		return defaultVal
	}

	return val
}

// Float64 reads and parses the value as a 64-bit float.
func (p *ConfData) Float64(key string) (float64, error) {
	val, err := p.String(key)
	if err != nil {
		return 0, err
	}

	ret, err := strconv.ParseFloat(val, 64)
	if err != nil {
		return 0, err
	}

	return ret, nil
}

// DefaultFloat64 is like Float64 but returns defaultVal on error or missing key.
func (p *ConfData) DefaultFloat64(key string, defaultVal float64) float64 {
	val, err := p.Float64(key)
	if err != nil {
		return defaultVal
	}

	return val
}

// Duration reads and parses the value with time.ParseDuration.
func (p *ConfData) Duration(key string) (time.Duration, error) {
	val, err := p.String(key)
	if err != nil {
		return 0, err
	}

	ret, err := time.ParseDuration(val)
	if err != nil {
		return 0, err
	}

	return ret, nil
}

// DefaultDuration is like Duration but returns defaultVal on error or missing key.
func (p *ConfData) DefaultDuration(key string, defaultVal time.Duration) time.Duration {
	val, err := p.Duration(key)
	if err != nil {
		return defaultVal
	}

	return val
}

// String returns the fully expanded string value (env, INI, then ${} / $[] interpolation, up to 10 nesting levels).
func (p *ConfData) String(key string) (string, error) {
	val, ok, err := p.stringEx(key)
	if ok {
		if err == nil {
			// nested level max 10
			for i := 0; i < 10; i++ {
				retVal, isMatch, err := p.analysisValue(val)
				if err != nil {
					return val, err
				}

				if !isMatch {
					return retVal, nil
				}

				val = retVal
			}

			return val, terror.ErrDataNotExist(key)
		}

		return val, err
	}

	if err != nil {
		return "", err
	}

	return "", terror.ErrDataNotExist(key)
}

// stringEx resolves key using APP_NAME-scoped and base forms; checks env then INI.
func (p *ConfData) stringEx(key string) (string, bool, error) {
	appName, ok, err := p.string(DefaultAppName)
	if !ok || err != nil {
		appName = ""
	} else {
		appName = strings.ToUpper(strings.Replace(appName, "-", "_", -1)) + "_"
	}

	keyPrefix := GetKeyPrefix()

	originalKey, baseKey := analysisKey(key, keyPrefix, appName)

	val, ok, err := p.string(originalKey)
	if ok {
		return val, ok, err
	}

	if originalKey != baseKey {
		val, ok, err = p.string(baseKey)
	}

	return val, ok, err
}

// string reads a raw value by key from env first, then INI. A nil receiver yields ErrNilConfData.
func (p *ConfData) string(key string) (string, bool, error) {
	if p == nil {
		return "", false, ErrNilConfData
	}

	if p.envData != nil {
		val, ok := p.envData.GetString(key)
		if ok {
			return val, ok, nil
		}
	}

	if p.iniData != nil {
		val, ok := p.iniData.GetString(key)
		if ok {
			return val, ok, nil
		}
	}

	return "", false, nil
}

// DefaultString is like String but returns defaultVal on error or missing key.
func (p *ConfData) DefaultString(key string, defaultVal string) string {
	val, err := p.String(key)
	if err != nil {
		return defaultVal
	}

	return val
}

// Strings splits String(key) by sep (empty single field becomes an empty slice).
func (p *ConfData) Strings(key string, sep string) ([]string, error) {
	val, err := p.String(key)
	if err != nil {
		return nil, err
	}

	vals := strings.Split(val, sep)

	if len(vals) == 1 && vals[0] == "" {
		return []string{}, nil
	}

	return vals, nil
}

// DefaultStrings is like Strings but returns defaultVals on error or missing key.
func (p *ConfData) DefaultStrings(key string, sep string, defaultVals []string) []string {
	val, err := p.Strings(key, sep)
	if err != nil {
		return defaultVals
	}

	return val
}

// DebugToString returns a short debug dump of INI section data as JSON, or a placeholder when nil.
func (p *ConfData) DebugToString() string {
	if p == nil {
		return "ini config data: <nil>."
	}

	if p.iniData == nil {
		return "ini config data: <nil>."
	}

	strIniData, _ := p.iniData.toString()

	return fmt.Sprintf("ini config data: %s.",
		strIniData)
}
