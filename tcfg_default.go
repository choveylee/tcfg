/**
 * @Author: lidonglin
 * @Description:
 * @File:  tcfg_default.go
 * @Version: 1.0.0
 * @Date: 2022/11/03 10:34
 */

package tcfg

var defaultConfData = &ConfData{}

// init load config from app_name_config.ini
func init() {
	configName := genConfName()

	err := defaultConfData.defaultLoad(configName)
	if err != nil {
		panic(err)
	}
}

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
