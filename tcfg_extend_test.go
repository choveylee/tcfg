package tcfg

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/choveylee/terror"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAnalysisKey(t *testing.T) {
	t.Parallel()

	t.Run("single without sub prefix", func(t *testing.T) {
		o, b := analysisKey("foo", "P_", "")
		assert.Equal(t, "P_FOO", o)
		assert.Equal(t, "P_FOO", b)
	})

	t.Run("single with prefix strip", func(t *testing.T) {
		o, b := analysisKey("P_FOO", "P_", "S_")
		assert.Equal(t, "P_FOO", o)
		assert.Equal(t, "P_FOO", b)
	})

	t.Run("single with sub prefix strip", func(t *testing.T) {
		o, b := analysisKey("P_S_X", "P_", "S_")
		assert.Equal(t, "P_S_X", o)
		assert.Equal(t, "P_X", b)
	})

	t.Run("section", func(t *testing.T) {
		o, b := analysisKey("dev::KEY", "P_", "S_")
		assert.Equal(t, "DEV::P_KEY", o)
		assert.Equal(t, "DEV::P_KEY", b)
	})

	t.Run("empty key uses only prefix", func(t *testing.T) {
		o, b := analysisKey("", "P_", "")
		assert.Equal(t, "P_", o)
		assert.Equal(t, "P_", b)
	})
}

func TestGenConfPaths(t *testing.T) {
	t.Parallel()

	p := filepath.Join(string(filepath.Separator), "a", "b", "c")
	paths := genConfPaths(p)
	require.GreaterOrEqual(t, len(paths), 2)
	assert.Equal(t, filepath.Dir(p), paths[0])
	// Walk ends at volume root (e.g. "/" on Unix, "C:\" on Windows).
	assert.True(t, filepath.IsAbs(paths[len(paths)-1]))
}

func TestAnalysisConfPath(t *testing.T) {
	t.Parallel()

	tmp := t.TempDir()
	cfgPath := filepath.Join(tmp, "ok.ini")
	require.NoError(t, os.WriteFile(cfgPath, []byte("K=V\n"), 0o644))

	got, err := analysisConfPath([]string{tmp}, "ok.ini")
	require.NoError(t, err)
	assert.Equal(t, cfgPath, got)

	got, err = analysisConfPath([]string{tmp}, "missing.ini")
	require.NoError(t, err)
	assert.Equal(t, "", got)

	dirAsName := filepath.Join(tmp, "bad.ini")
	require.NoError(t, os.Mkdir(dirAsName, 0o755))
	_, err = analysisConfPath([]string{tmp}, "bad.ini")
	require.Error(t, err)
	assert.Contains(t, err.Error(), filepath.Base(dirAsName))
	assert.Equal(t, terror.ErrConfIllegal(dirAsName).Error(), err.Error())
}

func TestIniMgrParseFileEmptyPath(t *testing.T) {
	t.Parallel()
	mgr := &IniMgr{}
	d, err := mgr.ParseFile("")
	require.NoError(t, err)
	require.NotNil(t, d)
	v, ok := d.GetString("ANY")
	assert.False(t, ok)
	assert.Equal(t, "", v)
}

func TestIniMgrParseConfigSectionKey(t *testing.T) {
	t.Parallel()
	mgr := &IniMgr{}
	d, err := mgr.ParseConfig([]*Config{
		{Key: "ROOT", Value: "v0"},
		{Key: "  dev  ::  subk  ", Value: `"quoted"`},
	})
	require.NoError(t, err)

	v, ok := d.GetString("ROOT")
	require.True(t, ok)
	assert.Equal(t, "v0", v)

	v, ok = d.GetString("DEV::SUBK")
	require.True(t, ok)
	assert.Equal(t, "quoted", v)
}

func TestParseBool(t *testing.T) {
	t.Parallel()

	v, err := parseBool(true)
	require.NoError(t, err)
	assert.True(t, v)

	v, err = parseBool("true")
	require.NoError(t, err)
	assert.True(t, v)

	v, err = parseBool("false")
	require.NoError(t, err)
	assert.False(t, v)

	v, err = parseBool(int(1))
	require.NoError(t, err)
	assert.True(t, v)

	v, err = parseBool(int64(0))
	require.NoError(t, err)
	assert.False(t, v)

	_, err = parseBool(int(2))
	require.Error(t, err)

	_, err = parseBool(nil)
	require.Error(t, err)
}

func TestGetSetKeyPrefix(t *testing.T) {
	prev := GetKeyPrefix()
	t.Cleanup(func() { SetKeyPrefix(prev) })

	SetKeyPrefix("T_")
	assert.Equal(t, "T_", GetKeyPrefix())

	SetKeyPrefix("")
	assert.Equal(t, "", GetKeyPrefix())
}

func TestConfDataStringInterpolation(t *testing.T) {
	t.Parallel()

	mgr := &IniMgr{}
	ini, err := mgr.ParseConfig([]*Config{
		{Key: "A", Value: "1"},
		{Key: "B", Value: "2"},
		{Key: "REF", Value: `${A}+${B}`},
	})
	require.NoError(t, err)

	p := &ConfData{
		iniData: ini,
		envData: &EnvData{},
	}

	out, err := p.String("REF")
	require.NoError(t, err)
	assert.Equal(t, "1+2", out)
}

func TestConfDataStringDollarEscape(t *testing.T) {
	t.Parallel()

	mgr := &IniMgr{}
	ini, err := mgr.ParseConfig([]*Config{
		{Key: "ESC", Value: `pre$${NOTVAR}post`},
	})
	require.NoError(t, err)

	p := &ConfData{
		iniData: ini,
		envData: &EnvData{},
	}

	out, err := p.String("ESC")
	require.NoError(t, err)
	assert.Equal(t, `pre${NOTVAR}post`, out)
}

func TestConfDataInt32Float32(t *testing.T) {
	t.Parallel()

	mgr := &IniMgr{}
	ini, err := mgr.ParseConfig([]*Config{
		{Key: "I32", Value: "42"},
		{Key: "F32", Value: "1.25"},
	})
	require.NoError(t, err)

	p := &ConfData{
		iniData: ini,
		envData: &EnvData{},
	}

	i, err := p.Int32("I32")
	require.NoError(t, err)
	assert.Equal(t, int32(42), i)

	f, err := p.Float32("F32")
	require.NoError(t, err)
	assert.InDelta(t, float32(1.25), f, 1e-5)

	assert.Equal(t, int32(9), p.DefaultInt32("NOPE", 9))
	assert.Equal(t, float32(2.5), p.DefaultFloat32("NOPE", 2.5))
}

func TestIniDataToStringJSON(t *testing.T) {
	t.Parallel()
	mgr := &IniMgr{}
	d, err := mgr.ParseConfig([]*Config{{Key: "K", Value: "v"}})
	require.NoError(t, err)
	s, err := d.toString()
	require.NoError(t, err)
	assert.Contains(t, s, `"K":"v"`)
}
