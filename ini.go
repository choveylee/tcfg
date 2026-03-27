/**
 * @Author: lidonglin
 * @Description:
 * @File:  ini.go
 * @Version: 1.0.0
 * @Date: 2022/11/03 10:35
 */

package tcfg

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Config is one key/value pair; Key may use "SECTION::KEY" to target a non-default section.
type Config struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

var (
	// DefaultSection is the INI section used when no "[section]" header applies.
	DefaultSection = "default"

	// NumCommentStr marks line comments when a line starts with '#'.
	NumCommentStr = []byte{'#'}
	// SemCommentStr marks line comments when a line starts with ';'.
	SemCommentStr = []byte{';'}

	EmptyStr = []byte{}
	EqualStr = []byte{'='}
	QuoteStr = []byte{'"'}

	SectionStartStr = []byte{'['}
	SectionEndStr   = []byte{']'}
)

// IniMgr parses INI text from disk or from in-memory Config rows.
type IniMgr struct {
}

// ParseFile reads an INI file; an empty filePath returns an empty IniData without error.
func (p *IniMgr) ParseFile(filePath string) (*IniData, error) {
	if filePath == "" {
		iniData := &IniData{
			filePath: filePath,

			data: make(map[string]map[string]string),

			secComment: make(map[string]string),
			keyComment: make(map[string]string),

			RWMutex: sync.RWMutex{},
		}

		return iniData, nil
	}

	return p.parseFile(filePath)
}

// ParseConfig builds IniData from programmatic key/value pairs (keys uppercased; values may be quoted).
func (p *IniMgr) ParseConfig(configs []*Config) (*IniData, error) {
	iniData := &IniData{
		filePath: "config",

		data: make(map[string]map[string]string),

		secComment: make(map[string]string),
		keyComment: make(map[string]string),

		RWMutex: sync.RWMutex{},
	}

	iniData.Lock()
	defer iniData.Unlock()

	for _, config := range configs {
		section := DefaultSection

		key := strings.ToUpper(config.Key)
		val := strings.TrimSpace(config.Value)

		params := strings.Split(key, "::")
		if len(params) > 1 {
			section = strings.TrimSpace(params[0])
			key = strings.TrimSpace(params[1])
		}

		if _, ok := iniData.data[section]; !ok {
			iniData.data[section] = make(map[string]string)
		}

		if strings.HasPrefix(val, string(QuoteStr)) {
			val = strings.Trim(val, string(QuoteStr))
		}

		iniData.data[section][key] = val
	}

	return iniData, nil
}

// parseFile reads bytes from disk and delegates to parseData with the file's directory for relative includes.
func (p *IniMgr) parseFile(filePath string) (*IniData, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	return p.parseData(filepath.Dir(filePath), data)
}

// parseData scans INI lines: UTF-8 BOM strip, sections, key=value, include "path", and line continuations for comments.
func (p *IniMgr) parseData(dir string, data []byte) (*IniData, error) {
	iniData := &IniData{
		data: make(map[string]map[string]string),

		secComment: make(map[string]string),
		keyComment: make(map[string]string),

		RWMutex: sync.RWMutex{},
	}

	iniData.Lock()
	defer iniData.Unlock()

	buf := bufio.NewReader(bytes.NewBuffer(data))

	// check the BOM (UTF-8: EF BB BF)
	head, err := buf.Peek(3)
	if err == nil && len(head) >= 3 && head[0] == 239 && head[1] == 187 && head[2] == 191 {
		for i := 1; i <= 3; i++ {
			_, _ = buf.ReadByte()
		}
	}

	var commentData bytes.Buffer
	section := DefaultSection

	for {
		isEof := false

		line, err := buf.ReadBytes('\n')
		if err == io.EOF {
			isEof = true
		}

		if _, ok := err.(*os.PathError); ok {
			return nil, err
		}

		line = bytes.TrimSpace(line)

		if bytes.Equal(line, EmptyStr) {
			if isEof {
				break
			}

			continue
		}

		var commentType []byte

		if bytes.HasPrefix(line, NumCommentStr) {
			commentType = NumCommentStr
		} else if bytes.HasPrefix(line, SemCommentStr) {
			commentType = SemCommentStr
		}

		// attach commentData
		if commentType != nil {
			line = bytes.TrimLeft(line, string(commentType))

			// Need append to a new line if multi-line comments.
			if commentData.Len() > 0 {
				commentData.WriteByte('\n')
			}

			commentData.Write(line)

			continue
		}

		if bytes.HasPrefix(line, SectionStartStr) && bytes.HasSuffix(line, SectionEndStr) {
			section = strings.ToUpper(string(line[1 : len(line)-1]))

			if commentData.Len() > 0 {
				iniData.secComment[section] = commentData.String()
				commentData.Reset()
			}

			_, ok := iniData.data[section]
			if !ok {
				iniData.data[section] = make(map[string]string)
			}

			continue
		}

		if _, ok := iniData.data[section]; !ok {
			iniData.data[section] = make(map[string]string)
		}

		params := bytes.SplitN(line, EqualStr, 2)

		key := strings.ToUpper(string(bytes.TrimSpace(params[0]))) // key name case-insensitive

		// handle include "other.ini"
		if len(params) == 1 {
			param := string(bytes.TrimSpace(params[0]))

			if strings.HasPrefix(param, "include") {
				includeFiles := strings.Fields(param)

				if includeFiles[0] == "include" && len(includeFiles) == 2 {
					includeFile := strings.Trim(includeFiles[1], "\"")

					if !filepath.IsAbs(includeFile) {
						includeFile = filepath.Join(dir, includeFile)
					}

					includeIniData, err := p.parseFile(includeFile)
					if err != nil {
						return nil, err
					}

					for section, vals := range includeIniData.data {
						_, ok := iniData.data[section]
						if !ok {
							iniData.data[section] = make(map[string]string)
						}

						for key, val := range vals {
							iniData.data[section][key] = val
						}
					}

					for section, comment := range includeIniData.secComment {
						iniData.secComment[section] = comment
					}

					for key, comment := range includeIniData.keyComment {
						iniData.keyComment[key] = comment
					}

					continue
				}
			}
		}

		if len(params) != 2 {
			return nil, errors.New("read the content error: \"" + string(line) + "\", should key = val")
		}

		val := bytes.TrimSpace(params[1])

		if bytes.HasPrefix(val, QuoteStr) {
			val = bytes.Trim(val, string(QuoteStr))
		}

		// replace //n to ##n
		retVal := strings.ReplaceAll(string(val), "\\\\n", "$$n")
		retVal = strings.ReplaceAll(retVal, "\\n", "\n")
		retVal = strings.ReplaceAll(retVal, "$$n", "\\n")

		iniData.data[section][key] = retVal

		if commentData.Len() > 0 {
			iniData.keyComment[section+"."+key] = commentData.String()
			commentData.Reset()
		}

		if isEof {
			break
		}
	}

	return iniData, nil
}

// IniData stores section/name/value maps with optional comment metadata.
// Get* accessors use SECTION::KEY; missing keys return ok=false (or zero) without error; parse errors surface from Bool/Int/etc.
type IniData struct {
	filePath string

	data map[string]map[string]string // section=> key:val

	secComment map[string]string // section : comment
	keyComment map[string]string // "section.KEY" : comment before the key line

	sync.RWMutex
}

// GetData returns a deep copy of all section data; callers may modify the returned maps safely.
func (p *IniData) GetData() map[string]map[string]string {
	p.RLock()
	defer p.RUnlock()

	if p.data == nil {
		return nil
	}

	iniData := make(map[string]map[string]string, len(p.data))

	for section, vals := range p.data {
		sectionData := make(map[string]string, len(vals))

		for key, val := range vals {
			sectionData[key] = val
		}

		iniData[section] = sectionData
	}

	return iniData
}

// GetBool returns the bool value, whether the key existed, and a parse error if the string was invalid.
func (p *IniData) GetBool(key string) (bool, bool, error) {
	val, ok := p.getData(key)
	if !ok {
		return false, false, nil
	}

	ret, err := parseBool(val)

	return ret, true, err
}

// Bool returns GetBool's value or false when the key is missing.
func (p *IniData) Bool(key string) (bool, error) {
	val, _, err := p.GetBool(key)

	return val, err
}

// DefaultBool returns defaultVal when the key is missing or parsing fails.
func (p *IniData) DefaultBool(key string, defaultVal bool) bool {
	val, ok, err := p.GetBool(key)
	if !ok || err != nil {
		return defaultVal
	}

	return val
}

// GetInt parses a decimal integer; ok is false when the key is missing.
func (p *IniData) GetInt(key string) (int, bool, error) {
	val, ok := p.getData(key)
	if !ok {
		return 0, false, nil
	}

	ret, err := strconv.Atoi(val)

	return ret, true, err
}

// Int is like GetInt but returns 0 when the key is missing.
func (p *IniData) Int(key string) (int, error) {
	val, _, err := p.GetInt(key)

	return val, err
}

// DefaultInt returns defaultVal when the key is missing or parsing fails.
func (p *IniData) DefaultInt(key string, defaultVal int) int {
	val, ok, err := p.GetInt(key)
	if !ok || err != nil {
		return defaultVal
	}

	return val
}

// GetInt64 parses a base-10 64-bit integer; ok is false when the key is missing.
func (p *IniData) GetInt64(key string) (int64, bool, error) {
	val, ok := p.getData(key)
	if !ok {
		return 0, false, nil
	}

	ret, err := strconv.ParseInt(val, 10, 64)

	return ret, true, err
}

// Int64 is like GetInt64 but returns 0 when the key is missing.
func (p *IniData) Int64(key string) (int64, error) {
	val, _, err := p.GetInt64(key)

	return val, err
}

// DefaultInt64 returns defaultVal when the key is missing or parsing fails.
func (p *IniData) DefaultInt64(key string, defaultVal int64) int64 {
	val, ok, err := p.GetInt64(key)
	if !ok || err != nil {
		return defaultVal
	}

	return val
}

// GetFloat parses a float64; ok is false when the key is missing.
func (p *IniData) GetFloat(key string) (float64, bool, error) {
	val, ok := p.getData(key)
	if !ok {
		return 0, false, nil
	}

	ret, err := strconv.ParseFloat(val, 64)

	return ret, true, err
}

// Float is like GetFloat but returns 0 when the key is missing.
func (p *IniData) Float(key string) (float64, error) {
	val, _, err := p.GetFloat(key)

	return val, err
}

// DefaultFloat returns defaultVal when the key is missing or parsing fails.
func (p *IniData) DefaultFloat(key string, defaultVal float64) float64 {
	val, ok, err := p.GetFloat(key)
	if !ok || err != nil {
		return defaultVal
	}

	return val
}

// GetDuration parses a duration string; ok is false when the key is missing.
func (p *IniData) GetDuration(key string) (time.Duration, bool, error) {
	val, ok := p.getData(key)
	if !ok {
		return time.Duration(0), false, nil
	}

	ret, err := time.ParseDuration(val)

	return ret, true, err
}

// Duration is like GetDuration but returns 0 when the key is missing.
func (p *IniData) Duration(key string) (time.Duration, error) {
	val, _, err := p.GetDuration(key)

	return val, err
}

// DefaultDuration returns defaultVal when the key is missing or parsing fails.
func (p *IniData) DefaultDuration(key string, defaultVal time.Duration) time.Duration {
	val, ok, err := p.GetDuration(key)
	if !ok || err != nil {
		return defaultVal
	}

	return val
}

// GetString returns the raw string for SECTION::KEY or the default section.
func (p *IniData) GetString(key string) (string, bool) {
	return p.getData(key)
}

// String returns GetString's value or "" when missing (no error distinction).
func (p *IniData) String(key string) string {
	val, _ := p.GetString(key)

	return val
}

// DefaultString returns defaultVal when the key is absent.
func (p *IniData) DefaultString(key string, defaultVal string) string {
	val, ok := p.GetString(key)
	if !ok {
		return defaultVal
	}

	return val
}

// GetStrings splits the string value by sep; ok is false when the key is missing.
func (p *IniData) GetStrings(key string, sep string) ([]string, bool) {
	vals, ok := p.GetString(key)
	if !ok {
		return nil, false
	}

	ret := strings.Split(vals, sep)

	return ret, true
}

// Strings returns GetStrings' slice or nil when the key is missing.
func (p *IniData) Strings(key string, sep string) []string {
	vals, _ := p.GetStrings(key, sep)

	return vals
}

// DefaultStrings returns defaultVals when the key is missing.
func (p *IniData) DefaultStrings(key string, sep string, defaultVals []string) []string {
	vals, ok := p.GetStrings(key, sep)
	if !ok {
		return defaultVals
	}

	return vals
}

// getData resolves an uppercased SECTION::KEY into the nested map (thread-safe read lock).
func (p *IniData) getData(key string) (string, bool) {
	if key == "" {
		return "", false
	}

	p.RLock()
	defer p.RUnlock()

	params := strings.Split(strings.ToUpper(key), "::")

	tmpSection := DefaultSection
	tmpKey := params[0]

	if len(params) >= 2 {
		tmpSection = params[0]
		tmpKey = params[1]
	}

	vals, ok := p.data[tmpSection]
	if !ok {
		return "", false
	}

	val, ok := vals[tmpKey]

	return val, ok
}

// toString JSON-encodes all section maps for debugging.
func (p *IniData) toString() (string, error) {
	p.RLock()
	defer p.RUnlock()

	data, err := json.Marshal(p.data)

	return string(data), err
}

// parseBool converts common string/int/bool representations into a bool (used by IniData and ConfData paths).
func parseBool(val interface{}) (value bool, err error) {
	if val != nil {
		switch v := val.(type) {
		case bool:
			return v, nil
		case string:
			switch v {
			case "1", "t", "T", "true", "TRUE", "True", "YES", "yes", "Yes", "Y", "y", "ON", "on", "On":
				return true, nil
			case "0", "f", "F", "false", "FALSE", "False", "NO", "no", "No", "N", "n", "OFF", "off", "Off":
				return false, nil
			}
		case int, int8, int32, int64:
			strV := fmt.Sprintf("%d", v)
			if strV == "1" {
				return true, nil
			} else if strV == "0" {
				return false, nil
			}
		case float64:
			if v == 1.0 {
				return true, nil
			} else if v == 0.0 {
				return false, nil
			}
		}
		return false, fmt.Errorf("parsing %q: invalid syntax", val)
	}
	return false, fmt.Errorf("parsing <nil>: invalid syntax")
}
