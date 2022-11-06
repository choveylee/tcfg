/**
 * @Author: lidonglin
 * @Description:
 * @File:  tcfg.go
 * @Version: 1.0.0
 * @Date: 2022/11/03 10:34
 */

package tcfg

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/choveylee/terror"
)

var (
	ValStringKeyMatchReg  = regexp.MustCompile(`\$\{(.+?)\}`)
	ValStringsKeyMatchReg = regexp.MustCompile(`\$\[(.+?)\]`)

	ValStringKeyReplaceReg = regexp.MustCompile(`\$\$\{(.+?)\}|\$\$\[(.+?)\]`)
)

const (
	DefaultStringsSeparator = ","

	DefaultBaseConfigName  = "base_config.ini"
	DefaultLocalConfigName = "local_config.ini"
)

const (
	DefaultAppName = "APP_NAME"
)

var (
	DefaultKeyPrefix = ""
)

func SetPrefix(prefix string) {
	DefaultKeyPrefix = prefix
}

// genLocalConfName gen default local config name rcrai_app_name_config.ini
func genLocalConfName() string {
	_, fileName := filepath.Split(os.Args[0])
	fileExt := filepath.Ext(os.Args[0])

	appName := strings.TrimSuffix(fileName, fileExt)
	appName = strings.ToLower(strings.ReplaceAll(appName, "-", "_"))

	localConfigName := fmt.Sprintf("%s_config.ini", appName)

	return localConfigName
}

// genConfPaths gen conf paths from config path to root path
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

// analysisConfPath analysis config file status & return available config file path
func analysisConfPath(confPaths []string, confName string) (string, error) {
	for _, confPath := range confPaths {
		retConfPath := filepath.Join(confPath, confName)

		file, err := os.Stat(retConfPath)
		if err == nil {
			if file.IsDir() == true {
				return "", terror.ErrConfIllegal(retConfPath)
			}

			return retConfPath, nil
		}

		if os.IsNotExist(err) == false {
			return "", err
		}
	}

	return "", nil
}

// analysisKey make sure key has given prefix string
func analysisKey(key string, prefix string, subPrefix string) (string, string) {
	key = strings.ToUpper(key)

	params := strings.Split(key, "::")

	if len(params) == 1 {
		if strings.HasPrefix(key, prefix) == true {
			key = strings.TrimPrefix(key, prefix)
		}

		tmpKey := key
		if strings.HasPrefix(tmpKey, subPrefix) == true {
			tmpKey = strings.TrimPrefix(tmpKey, subPrefix)
		}

		originalKey := prefix + key
		baseKey := prefix + tmpKey

		return originalKey, baseKey
	} else if len(params) > 1 {
		section := params[0]
		key := params[1]

		if strings.HasPrefix(key, prefix) == true {
			key = strings.TrimPrefix(key, prefix)
		}

		tmpKey := key
		if strings.HasPrefix(tmpKey, subPrefix) == true {
			tmpKey = strings.TrimPrefix(tmpKey, subPrefix)
		}

		originalKey := fmt.Sprintf("%s::%s%s", section, prefix, key)
		baseKey := fmt.Sprintf("%s::%s%s", section, prefix, tmpKey)

		return originalKey, baseKey
	}

	return "", ""
}

type ConfData struct {
	baseConf  *IniData
	localConf *IniData

	envConf *EnvData
}

type Configs struct {
	BaseConfigs  []*Config `json:"base_configs"`
	LocalConfigs []*Config `json:"local_configs"`
}

type Response struct {
	Error   int
	Message string
	Data    *Configs `json:"data"`
}

func loadFromFile(baseConfName, localConfName string) (*IniData, *IniData, error) {
	workPath, err := os.Getwd()
	if err != nil {
		return nil, nil, err
	}

	appPath, err := filepath.Abs(filepath.Dir(os.Args[0]))
	if err != nil {
		return nil, nil, err
	}

	extConfigPaths := make([]string, 0)

	workConfigPaths := genConfPaths(workPath)
	appConfigPaths := genConfPaths(appPath)

	extConfigPaths = append(extConfigPaths, workConfigPaths...)
	extConfigPaths = append(extConfigPaths, appConfigPaths...)

	// get base config file from cmd work path > app path
	baseConfPaths := []string{
		workPath,
		appPath,
	}
	baseConfPaths = append(baseConfPaths, extConfigPaths...)

	baseConfPath, err := analysisConfPath(baseConfPaths, baseConfName)
	if err != nil {
		return nil, nil, err
	}

	// get local config file from cmd work path > app path
	localConfPaths := []string{
		workPath,
		appPath,
	}
	localConfPaths = append(localConfPaths, extConfigPaths...)

	localConfPath, err := analysisConfPath(localConfPaths, localConfName)
	if err != nil {
		return nil, nil, err
	}

	iniMgr := &IniMgr{}

	baseConf, err := iniMgr.ParseFile(baseConfPath)
	if err != nil {
		return nil, nil, err
	}

	localConf, err := iniMgr.ParseFile(localConfPath)
	if err != nil {
		return nil, nil, err
	}

	return baseConf, localConf, nil
}

func (p *ConfData) defaultLoad(baseConfName, localConfName string) error {
	// 1. init env
	envConf := &EnvData{}

	p.envConf = envConf

	// 2. load from file
	baseConf, localConf, err := loadFromFile(baseConfName, localConfName)
	if err == nil {
		p.baseConf = baseConf
		p.localConf = localConf

		return nil
	}

	return err
}

// analysisValue support replace ${Key} & $[Key]
func (p *ConfData) analysisValue(val string) (string, bool, error) {
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
		if ok == false {
			return val, isMatch, terror.ErrDataNotExist(key)
		} else {
			if err != nil {
				return val, isMatch, err
			}

			retVal += realVal
			startIndex = tmpEndIndex
		}
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
		if ok == true {
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

	if isMatch == true {
		return retVal, isMatch, nil
	}

	// 3 replace $${key} & &&[key]
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

// LocalKey convert key to local key
func (p *ConfData) LocalKey(key string) string {
	appName, ok, err := p.string(DefaultAppName)
	if err == nil && ok == true {
		appName = strings.ToUpper(strings.Replace(appName, "-", "_", -1))

		if strings.HasPrefix(key, DefaultKeyPrefix) == true {
			key = strings.TrimPrefix(key, DefaultKeyPrefix)
		}

		key = fmt.Sprintf("%s%s_%s", DefaultKeyPrefix, appName, key)
	}

	return key
}

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

func (p *ConfData) DefaultBool(key string, defaultVal bool) bool {
	val, err := p.Bool(key)
	if err != nil {
		return defaultVal
	}

	return val
}

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

func (p *ConfData) DefaultInt(key string, defaultVal int) int {
	val, err := p.Int(key)
	if err != nil {
		return defaultVal
	}

	return val
}

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

func (p *ConfData) DefaultInt32(key string, defaultVal int32) int32 {
	val, err := p.Int32(key)
	if err != nil {
		return defaultVal
	}

	return val
}

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

func (p *ConfData) DefaultInt64(key string, defaultVal int64) int64 {
	val, err := p.Int64(key)
	if err != nil {
		return defaultVal
	}

	return val
}

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

func (p *ConfData) DefaultFloat32(key string, defaultVal float32) float32 {
	val, err := p.Float32(key)
	if err != nil {
		return defaultVal
	}

	return val
}

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

func (p *ConfData) DefaultFloat64(key string, defaultVal float64) float64 {
	val, err := p.Float64(key)
	if err != nil {
		return defaultVal
	}

	return val
}

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

func (p *ConfData) DefaultDuration(key string, defaultVal time.Duration) time.Duration {
	val, err := p.Duration(key)
	if err != nil {
		return defaultVal
	}

	return val
}

func (p *ConfData) String(key string) (string, error) {
	val, ok, err := p.stringEx(key)
	if ok == true {
		if err == nil {
			// nested level max 10
			for i := 0; i < 10; i++ {
				retVal, isMatch, err := p.analysisValue(val)
				if err != nil {
					return val, err
				}

				if isMatch == false {
					return retVal, nil
				}

				val = retVal
			}

			return val, terror.ErrDataNotExist(key)
		}

		return val, err
	}

	return "", terror.ErrDataNotExist(key)
}

// stringEx
func (p *ConfData) stringEx(key string) (string, bool, error) {
	appName, ok, err := p.string(DefaultAppName)
	if ok == false || err != nil {
		appName = ""
	} else {
		appName = strings.ToUpper(strings.Replace(appName, "-", "_", -1)) + "_"
	}

	originalKey, baseKey := analysisKey(key, DefaultKeyPrefix, appName)

	val, ok, err := p.string(originalKey)
	if ok == true {
		return val, ok, err
	}

	if originalKey != baseKey {
		val, ok, err = p.string(baseKey)
	}

	return val, ok, err
}

func (p *ConfData) string(key string) (string, bool, error) {
	val, ok := p.envConf.GetString(key)
	if ok == true {
		return val, ok, nil
	}

	val, ok = p.localConf.GetString(key)
	if ok == true {
		return val, ok, nil
	}

	val, ok = p.baseConf.GetString(key)
	if ok == true {
		return val, ok, nil
	}

	return val, ok, nil
}

func (p *ConfData) DefaultString(key string, defaultVal string) string {
	val, err := p.String(key)
	if err != nil {
		return defaultVal
	}

	return val
}

func (p *ConfData) Strings(key string, sep string) ([]string, error) {
	val, err := p.String(key)
	if err != nil {
		return nil, err
	}

	vals := strings.Split(val, sep)

	return vals, nil
}

func (p *ConfData) DefaultStrings(key string, sep string, defaultVals []string) []string {
	val, err := p.Strings(key, sep)
	if err != nil {
		return defaultVals
	}

	return val
}

func (p *ConfData) DebugToString() string {
	baseConfData, _ := p.baseConf.toString()

	localConfData, _ := p.localConf.toString()

	return fmt.Sprintf("base config data: %s.\nlocal config data: %s",
		baseConfData, localConfData)
}
