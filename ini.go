package tcfg

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Config is a single key/value row. Key may use the form SECTION::KEY to select a non-default section.
type Config struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

var (
	// DefaultSection names the section used when no [section] header has been read yet.
	DefaultSection = "default"

	// NumCommentStr is the prefix of '#' line comments.
	NumCommentStr = []byte{'#'}
	// SemCommentStr is the prefix of ';' line comments.
	SemCommentStr = []byte{';'}

	// EmptyStr is an empty byte slice used in line scanning.
	EmptyStr = []byte{}
	// EqualStr is the '=' delimiter between keys and values.
	EqualStr = []byte{'='}
	// QuoteStr is the double-quote byte used to detect quoted values.
	QuoteStr = []byte{'"'}

	// SectionStartStr and SectionEndStr delimit INI section headers.
	SectionStartStr = []byte{'['}
	SectionEndStr   = []byte{']'}
)

// IniMgr parses INI text from files or from in-memory [Config] rows.
type IniMgr struct {
}

func parseIncludeDirective(line string) (string, bool, error) {
	line = strings.TrimSpace(line)
	if !strings.HasPrefix(line, "include") {
		return "", false, nil
	}

	if len(line) > len("include") {
		nextChar := line[len("include")]
		if nextChar != ' ' && nextChar != '\t' {
			return "", false, nil
		}
	}

	rest := strings.TrimSpace(line[len("include"):])
	if rest == "" {
		return "", true, fmt.Errorf("tcfg: the include directive requires a file path")
	}

	if rest[0] == '"' {
		if len(rest) < 2 || rest[len(rest)-1] != '"' {
			return "", true, fmt.Errorf("tcfg: malformed include directive: %q", line)
		}

		return rest[1 : len(rest)-1], true, nil
	}

	if strings.ContainsAny(rest, " \t") {
		return "", true, fmt.Errorf("tcfg: include paths containing spaces must be enclosed in double quotes: %q", line)
	}

	return rest, true, nil
}

// ParseFile reads and parses the file at filePath. An empty filePath returns an empty [IniData] without error.
// Include directives are resolved relative to the including file, and circular includes return an error.
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

	return p.parseFile(filePath, nil)
}

// ParseConfig builds an [IniData] from configs. Keys are uppercased; values may be surrounded by quotes.
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

// parseFile reads filePath and parses its contents; relative include paths resolve against the file's directory.
func (p *IniMgr) parseFile(filePath string, includeStack []string) (*IniData, error) {
	filePath, err := filepath.Abs(filePath)
	if err != nil {
		return nil, err
	}

	filePath = filepath.Clean(filePath)

	for index, includePath := range includeStack {
		if includePath != filePath {
			continue
		}

		cycle := append(append([]string{}, includeStack[index:]...), filePath)

		return nil, fmt.Errorf("tcfg: circular include detected: %s", strings.Join(cycle, " -> "))
	}

	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	nextStack := append(includeStack, filePath)

	return p.parseData(filepath.Dir(filePath), data, nextStack)
}

// parseData parses INI content from data. It strips a UTF-8 BOM, handles [section] headers, key=value lines,
// include "path" directives, and comment blocks associated with sections or keys.
func (p *IniMgr) parseData(dir string, data []byte, includeStack []string) (*IniData, error) {
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

			includeFile, isInclude, err := parseIncludeDirective(param)
			if err != nil {
				return nil, err
			}
			if isInclude {
				if !filepath.IsAbs(includeFile) {
					includeFile = filepath.Join(dir, includeFile)
				}

				includeIniData, err := p.parseFile(includeFile, includeStack)
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

		if len(params) != 2 {
			return nil, fmt.Errorf("tcfg: invalid configuration line %q: expected KEY=VALUE syntax", string(line))
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

// IniData holds per-section key/value maps and optional comment metadata.
//
// Getters accept keys in the form SECTION::KEY; missing keys yield ok == false or zero values without an error.
// Parse errors are returned from typed accessors such as [IniData.Bool] and [IniData.Int].
type IniData struct {
	filePath string

	data map[string]map[string]string // section=> key:val

	secComment map[string]string // section : comment
	keyComment map[string]string // "section.KEY" : comment before the key line

	sync.RWMutex
}

// GetData returns a deep copy of all section maps. The caller may modify the returned maps without affecting p.
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

// GetBool returns the boolean value, whether the key exists, and a parse error if the stored string is not a valid boolean.
func (p *IniData) GetBool(key string) (bool, bool, error) {
	val, ok := p.getData(key)
	if !ok {
		return false, false, nil
	}

	ret, err := parseBool(val)

	return ret, true, err
}

// Bool returns the value from [IniData.GetBool], or false when the key is missing.
func (p *IniData) Bool(key string) (bool, error) {
	val, _, err := p.GetBool(key)

	return val, err
}

// DefaultBool returns defaultVal when the key is missing or [IniData.GetBool] reports a parse error.
func (p *IniData) DefaultBool(key string, defaultVal bool) bool {
	val, ok, err := p.GetBool(key)
	if !ok || err != nil {
		return defaultVal
	}

	return val
}

// GetInt parses the value as a base-10 integer. The second return value is false when the key is missing.
func (p *IniData) GetInt(key string) (int, bool, error) {
	val, ok := p.getData(key)
	if !ok {
		return 0, false, nil
	}

	ret, err := strconv.Atoi(val)

	return ret, true, err
}

// Int returns the value from [IniData.GetInt], or zero when the key is missing.
func (p *IniData) Int(key string) (int, error) {
	val, _, err := p.GetInt(key)

	return val, err
}

// DefaultInt returns defaultVal when the key is missing or [IniData.GetInt] fails to parse.
func (p *IniData) DefaultInt(key string, defaultVal int) int {
	val, ok, err := p.GetInt(key)
	if !ok || err != nil {
		return defaultVal
	}

	return val
}

// GetInt64 parses the value as a base-10 64-bit integer. The second return value is false when the key is missing.
func (p *IniData) GetInt64(key string) (int64, bool, error) {
	val, ok := p.getData(key)
	if !ok {
		return 0, false, nil
	}

	ret, err := strconv.ParseInt(val, 10, 64)

	return ret, true, err
}

// Int64 returns the value from [IniData.GetInt64], or zero when the key is missing.
func (p *IniData) Int64(key string) (int64, error) {
	val, _, err := p.GetInt64(key)

	return val, err
}

// DefaultInt64 returns defaultVal when the key is missing or [IniData.GetInt64] fails to parse.
func (p *IniData) DefaultInt64(key string, defaultVal int64) int64 {
	val, ok, err := p.GetInt64(key)
	if !ok || err != nil {
		return defaultVal
	}

	return val
}

// GetFloat parses the value as a 64-bit floating-point number. The second return value is false when the key is missing.
func (p *IniData) GetFloat(key string) (float64, bool, error) {
	val, ok := p.getData(key)
	if !ok {
		return 0, false, nil
	}

	ret, err := strconv.ParseFloat(val, 64)

	return ret, true, err
}

// Float returns the value from [IniData.GetFloat], or zero when the key is missing.
func (p *IniData) Float(key string) (float64, error) {
	val, _, err := p.GetFloat(key)

	return val, err
}

// DefaultFloat returns defaultVal when the key is missing or [IniData.GetFloat] fails to parse.
func (p *IniData) DefaultFloat(key string, defaultVal float64) float64 {
	val, ok, err := p.GetFloat(key)
	if !ok || err != nil {
		return defaultVal
	}

	return val
}

// GetDuration parses the value with [time.ParseDuration]. The second return value is false when the key is missing.
func (p *IniData) GetDuration(key string) (time.Duration, bool, error) {
	val, ok := p.getData(key)
	if !ok {
		return time.Duration(0), false, nil
	}

	ret, err := time.ParseDuration(val)

	return ret, true, err
}

// Duration returns the value from [IniData.GetDuration], or zero when the key is missing.
func (p *IniData) Duration(key string) (time.Duration, error) {
	val, _, err := p.GetDuration(key)

	return val, err
}

// DefaultDuration returns defaultVal when the key is missing or [IniData.GetDuration] fails to parse.
func (p *IniData) DefaultDuration(key string, defaultVal time.Duration) time.Duration {
	val, ok, err := p.GetDuration(key)
	if !ok || err != nil {
		return defaultVal
	}

	return val
}

// GetString returns the raw string value for SECTION::KEY, or for the default section when no section is given.
func (p *IniData) GetString(key string) (string, bool) {
	return p.getData(key)
}

// String returns the value from [IniData.GetString], or an empty string when the key is missing.
func (p *IniData) String(key string) string {
	val, _ := p.GetString(key)

	return val
}

// DefaultString returns defaultVal when the key is missing.
func (p *IniData) DefaultString(key string, defaultVal string) string {
	val, ok := p.GetString(key)
	if !ok {
		return defaultVal
	}

	return val
}

// GetStrings splits the string value using sep. The second return value is false when the key is missing.
func (p *IniData) GetStrings(key string, sep string) ([]string, bool) {
	vals, ok := p.GetString(key)
	if !ok {
		return nil, false
	}

	ret := strings.Split(vals, sep)

	return ret, true
}

// Strings returns the slice from [IniData.GetStrings], or nil when the key is missing.
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

// getData returns the value for an uppercased SECTION::KEY under the read lock.
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

// toString returns a JSON representation of all section data for debugging.
func (p *IniData) toString() (string, error) {
	p.RLock()
	defer p.RUnlock()

	data, err := json.Marshal(p.data)

	return string(data), err
}

// parseBool interprets val as a boolean. Supported forms include common string literals and numeric 0/1 for integer and float64 types.
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
		return false, fmt.Errorf("tcfg: invalid boolean value %q", val)
	}
	return false, fmt.Errorf("tcfg: invalid boolean value <nil>")
}
