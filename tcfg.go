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

// ValStringKeyMatchReg matches ${name} placeholders in interpolated values.
//
// ValStringsKeyMatchReg matches $[name] list placeholders.
//
// ValStringKeyReplaceReg matches escaped $${name} and $$[name] spans for the unescape pass.
var (
	ValStringKeyMatchReg  = regexp.MustCompile(`\$\{(.+?)\}`)
	ValStringsKeyMatchReg = regexp.MustCompile(`\$\[(.+?)\]`)

	ValStringKeyReplaceReg = regexp.MustCompile(`\$\$\{(.+?)\}|\$\$\[(.+?)\]`)
)

// DefaultStringsSeparator is the separator used when splitting list values during $[key] expansion.
const (
	DefaultStringsSeparator = ","
)

// DefaultAppName is the configuration key that holds the application name for APP_NAME-scoped resolution.
const (
	DefaultAppName = "APP_NAME"
)

// ErrNilConfData is returned when a method is called on a nil *ConfData receiver.
var ErrNilConfData = errors.New("tcfg: method called on nil *ConfData receiver")

var (
	keyPrefixMutex sync.RWMutex

	defaultKeyPrefix string
)

// GetKeyPrefix returns the global key prefix set by [SetKeyPrefix]. It is safe for concurrent use with [SetKeyPrefix].
func GetKeyPrefix() string {
	keyPrefixMutex.RLock()
	defer keyPrefixMutex.RUnlock()

	return defaultKeyPrefix
}

// SetKeyPrefix sets the global key prefix used when resolving keys. It is safe for concurrent use with [GetKeyPrefix].
func SetKeyPrefix(keyPrefix string) {
	keyPrefixMutex.Lock()

	defaultKeyPrefix = keyPrefix

	keyPrefixMutex.Unlock()
}

// genConfName returns the default configuration file name derived from os.Args[0]
// (<basename>_config.ini with the basename lowercased and '-' replaced by '_').
func genConfName() string {
	_, fileName := filepath.Split(os.Args[0])
	fileExt := filepath.Ext(os.Args[0])

	appName := strings.TrimSuffix(fileName, fileExt)
	appName = strings.ToLower(strings.ReplaceAll(appName, "-", "_"))

	configName := fmt.Sprintf("%s_config.ini", appName)

	return configName
}

// genConfPaths returns ancestor directory paths from the parent of configPath up to the filesystem root.
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

// analysisConfPath returns the path to the first regular file named confName in confPaths,
// or ("", nil) if none exists. If a matching path names a directory, it returns an error.
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

// analysisKey builds uppercased lookup keys with optional prefix and APP_NAME sub-prefix.
// Keys may use SECTION::KEY. It returns (originalKey, baseKey) for fallback when scoped and base forms differ.
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

// ConfData combines environment values with parsed INI data. For each key the environment is consulted first.
type ConfData struct {
	iniData *IniData

	envData *EnvData
}

// Configs is a JSON-serializable list of key/value pairs, typically used with [Response].
type Configs struct {
	Configs []*Config `json:"configs"`
}

// Response carries an error code, message, and optional structured configuration data.
type Response struct {
	Error   int
	Message string

	Data *Configs `json:"data"`
}

// loadFromFile locates configName under the working directory and executable directory (and their ancestors) and parses it.
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

// defaultLoad attaches a new [EnvData] to p and merges INI content loaded from configName.
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

// analysisValue performs ${key} substitution, Cartesian expansion of $[key] list placeholders,
// and unescaping of $${...} and $$[...] spans. The bool is true if any ${} or $[] substitution ran
// in the first two phases (the unescape pass may still run when it is false).
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

// LocalKey returns a key qualified with [GetKeyPrefix] and [DefaultAppName] when APP_NAME is set,
// after stripping a leading key prefix if present.
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

// Bool returns the boolean value associated with key. The underlying string is parsed after [ConfData.String] resolution.
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

// DefaultBool returns defaultVal if [ConfData.Bool] would fail or the key is missing.
func (p *ConfData) DefaultBool(key string, defaultVal bool) bool {
	val, err := p.Bool(key)
	if err != nil {
		return defaultVal
	}

	return val
}

// Int returns the decimal integer associated with key after [ConfData.String] resolution.
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

// DefaultInt returns defaultVal if [ConfData.Int] would fail or the key is missing.
func (p *ConfData) DefaultInt(key string, defaultVal int) int {
	val, err := p.Int(key)
	if err != nil {
		return defaultVal
	}

	return val
}

// Int32 returns the signed 32-bit integer associated with key after [ConfData.String] resolution.
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

// DefaultInt32 returns defaultVal if [ConfData.Int32] would fail or the key is missing.
func (p *ConfData) DefaultInt32(key string, defaultVal int32) int32 {
	val, err := p.Int32(key)
	if err != nil {
		return defaultVal
	}

	return val
}

// Int64 returns the signed 64-bit integer associated with key after [ConfData.String] resolution.
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

// DefaultInt64 returns defaultVal if [ConfData.Int64] would fail or the key is missing.
func (p *ConfData) DefaultInt64(key string, defaultVal int64) int64 {
	val, err := p.Int64(key)
	if err != nil {
		return defaultVal
	}

	return val
}

// Float32 returns the 32-bit floating-point value associated with key after [ConfData.String] resolution.
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

// DefaultFloat32 returns defaultVal if [ConfData.Float32] would fail or the key is missing.
func (p *ConfData) DefaultFloat32(key string, defaultVal float32) float32 {
	val, err := p.Float32(key)
	if err != nil {
		return defaultVal
	}

	return val
}

// Float64 returns the 64-bit floating-point value associated with key after [ConfData.String] resolution.
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

// DefaultFloat64 returns defaultVal if [ConfData.Float64] would fail or the key is missing.
func (p *ConfData) DefaultFloat64(key string, defaultVal float64) float64 {
	val, err := p.Float64(key)
	if err != nil {
		return defaultVal
	}

	return val
}

// Duration returns the duration associated with key, parsed with [time.ParseDuration] after [ConfData.String] resolution.
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

// DefaultDuration returns defaultVal if [ConfData.Duration] would fail or the key is missing.
func (p *ConfData) DefaultDuration(key string, defaultVal time.Duration) time.Duration {
	val, err := p.Duration(key)
	if err != nil {
		return defaultVal
	}

	return val
}

// String returns the fully expanded value for key: environment and INI resolution, then up to ten rounds
// of ${} and $[] interpolation.
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

// stringEx resolves key using APP_NAME-scoped and base key forms, consulting the environment before INI.
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

// string returns the raw value for key from the environment if present, otherwise from INI.
// A nil receiver returns [ErrNilConfData].
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

// DefaultString returns defaultVal if [ConfData.String] would fail or the key is missing.
func (p *ConfData) DefaultString(key string, defaultVal string) string {
	val, err := p.String(key)
	if err != nil {
		return defaultVal
	}

	return val
}

// Strings splits the expanded [ConfData.String] value for key using sep. A single empty field yields an empty slice.
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

// DefaultStrings returns defaultVals if [ConfData.Strings] would fail or the key is missing.
func (p *ConfData) DefaultStrings(key string, sep string, defaultVals []string) []string {
	val, err := p.Strings(key, sep)
	if err != nil {
		return defaultVals
	}

	return val
}

// DebugToString returns a human-readable summary of parsed INI data, or a placeholder if p or INI data is nil.
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
