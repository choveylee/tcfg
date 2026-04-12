// Package tcfg loads INI configuration together with environment variables, resolves keys
// using an optional prefix and APP_NAME-based scoping, and expands values that contain
// ${key} and $[key] placeholders (including $${...} and $$[...] escapes for literal dollar signs).
//
// # Default import behavior
//
// Importing this package runs init, which constructs a default [ConfData], loads a file named
// <executable_basename>_config.ini (basename lowercased; '-' replaced with '_'), and binds
// package-level accessors such as [String] and [Bool] to that instance. If loading fails with
// an error, init panics.
//
// # Resolution order
//
// For each key, environment variables are consulted before INI values. Keys may use the form
// SECTION::KEY in INI; in the environment layer this maps to KEY_SECTION (see [EnvData]).
//
// # INI without default loading
//
// Use [IniMgr] and [IniData] to parse INI files or in-memory [Config] rows without the default
// [ConfData] path or automatic environment merging.
package tcfg
