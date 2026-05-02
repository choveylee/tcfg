// Package tcfg provides facilities for loading configuration from INI files and environment variables.
// It supports optional key prefixes, APP_NAME-based scoping, and value expansion through
// ${key} and $[key] placeholders, including $${...} and $$[...] escapes for literal dollar signs.
//
// # Default package initialization
//
// Importing this package initializes the default [ConfData] instance, derives a configuration file name
// in the form <executable_basename>_config.ini, and binds package-level helpers such as [String] and [Bool]
// to that instance. The default loader searches the current working directory first and then the
// directory of the current executable. If initialization encounters an error, package initialization panics.
//
// # Resolution order
//
// For each logical key, environment variables take precedence over INI values. INI keys may use the
// form SECTION::KEY; in the environment layer this maps to KEY_SECTION (see [EnvData]).
//
// # INI parsing without default loading
//
// Use [IniMgr] and [IniData] to parse INI files or in-memory [Config] entries without relying on
// package initialization or the default [ConfData] instance.
package tcfg
