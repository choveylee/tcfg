/**
 * @Author: lidonglin
 * @Description:
 * @File:  tcfg_default.go
 * @Version: 1.0.0
 * @Date: 2022/11/03 10:34
 */

package tcfg

var defaultConfData = &ConfData{}

// init load global config from base_config.ini & local config from app_name_config.ini
func init() {
	globalConfName := DefaultBaseConfigName
	localConfName := genLocalConfName()

	err := defaultConfData.defaultLoad(globalConfName, localConfName)
	if err != nil {
		panic(err)
	}
}

var LocalKey = defaultConfData.LocalKey

var Bool = defaultConfData.Bool
var DefaultBool = defaultConfData.DefaultBool

var Int = defaultConfData.Int
var DefaultInt = defaultConfData.DefaultInt

var Int64 = defaultConfData.Int64
var DefaultInt64 = defaultConfData.DefaultInt64

var Float64 = defaultConfData.Float64
var DefaultFloat64 = defaultConfData.DefaultFloat64

var Duration = defaultConfData.Duration
var DefaultDuration = defaultConfData.DefaultDuration

var String = defaultConfData.String
var DefaultString = defaultConfData.DefaultString

var Strings = defaultConfData.Strings
var DefaultStrings = defaultConfData.DefaultStrings

var DebugToString = defaultConfData.DebugToString
