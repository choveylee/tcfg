// Package tcfg loads INI files together with environment variables, resolves keys using
// an optional key prefix and APP_NAME-based scoping, and expands values with ${key} and
// $[key] placeholders (including $${...} / $$[...] escapes for literal $ characters).
//
// IniMgr/IniData handle file and in-memory INI parsing (sections, includes, UTF-8 BOM).
// EnvData reads os environment variables (SECTION::KEY maps to KEY_SECTION).
// The default ConfData instance is loaded in init from <exe_basename>_config.ini; package-level
// functions (String, Bool, …) delegate to that instance.
package tcfg
