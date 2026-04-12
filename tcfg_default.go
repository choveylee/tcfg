package tcfg

// defaultConfData holds the configuration used by package-level accessors.
var defaultConfData = &ConfData{}

func init() {
	configName := genConfName()

	err := defaultConfData.defaultLoad(configName)
	if err != nil {
		panic(err)
	}
}

// Package-level variables are bound to defaultConfData and mirror the corresponding (*ConfData)
// methods loaded during init.
var LocalKey = defaultConfData.LocalKey

var Bool = defaultConfData.Bool
var DefaultBool = defaultConfData.DefaultBool

var Int = defaultConfData.Int
var DefaultInt = defaultConfData.DefaultInt

var Int32 = defaultConfData.Int32
var DefaultInt32 = defaultConfData.DefaultInt32

var Int64 = defaultConfData.Int64
var DefaultInt64 = defaultConfData.DefaultInt64

var Float32 = defaultConfData.Float32
var DefaultFloat32 = defaultConfData.DefaultFloat32

var Float64 = defaultConfData.Float64
var DefaultFloat64 = defaultConfData.DefaultFloat64

var Duration = defaultConfData.Duration
var DefaultDuration = defaultConfData.DefaultDuration

var String = defaultConfData.String
var DefaultString = defaultConfData.DefaultString

var Strings = defaultConfData.Strings
var DefaultStrings = defaultConfData.DefaultStrings

var DebugToString = defaultConfData.DebugToString
