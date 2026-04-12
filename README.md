# tcfg

[![Go Reference](https://pkg.go.dev/badge/github.com/choveylee/tcfg.svg)](https://pkg.go.dev/github.com/choveylee/tcfg)

Small Go library for **INI** files plus **environment variables**, optional **key prefix** and **`APP_NAME`** scoping, and value expansion with **`${key}`** and **`$[key]`** (with **`$${...}` / `$$[...]`** escapes for literal `$`).

## Features

- Merge **env** (first) and **INI** values for the same logical key
- Resolve **`SECTION::KEY`** (INI); env uses **`KEY_SECTION`**
- Optional global key prefix and per-app scoping via **`APP_NAME`**
- **`${name}`** string interpolation and **`$[name]`** list expansion (Cartesian product when multiple appear)
- **`IniMgr` / `IniData`** for file or in-memory INI without the package `init` path

## Requirements

- Go **1.26.1** or later (see [`go.mod`](go.mod))

## Installation

```bash
go get github.com/choveylee/tcfg
```

## Usage

### Package-level API

Importing the package runs **`init`**, which loads **`<executable_basename>_config.ini`** (basename lowercased, `-` → `_`) and wires package-level helpers (`tcfg.String`, `tcfg.Bool`, …) to a default **`ConfData`**. If that load path fails in a way that returns an error (for example I/O or parse errors on a found file), **`init` panics**.

```go
import "github.com/choveylee/tcfg"

func example() {
    v, err := tcfg.String("MY_KEY")
    if err != nil {
        // missing key, parse error, nested expansion failure, etc.
        return
    }
    _ = v
}
```

Use this when you want env + INI + `${}` / `$[]` behavior without constructing **`ConfData`** yourself (its fields are not exported).

### INI-only (`IniMgr`)

For parsing INI only—no package `init`, no automatic env merge—use **`IniMgr`**:

```go
mgr := &tcfg.IniMgr{}
ini, err := mgr.ParseFile("app.ini")
if err != nil {
    log.Fatal(err)
}
if v, ok := ini.GetString("MY_KEY"); ok {
    _ = v
}
```

Build data from slices with **`ParseConfig`**. INI parsing supports sections, **`include "path"`** (relative paths resolved from the including file’s directory), UTF-8 BOM, and line comments (`#`, `;`).

## Configuration file discovery

The default loader searches for **`<executable_basename>_config.ini`** in this order:

1. Current working directory  
2. Directory of the executable  
3. Ancestor directories of (1), then of (2), walking toward the filesystem root  

The first regular file match wins. If no file is found, loading still succeeds with empty INI data (see implementation).

## Environment variables

- Lookups try **environment first**, then INI.  
- Keys may use **`SECTION::KEY`**. In the environment layer this maps to **`KEY_SECTION`** (see **`EnvData`**).

## Key prefix and `APP_NAME`

- **`GetKeyPrefix`** / **`SetKeyPrefix`** set a global prefix applied during key resolution (safe for concurrent use with the internal mutex).  
- If **`APP_NAME`** is set, keys can include an extra normalized segment (uppercase, `-` → `_`) for app-specific names (**`LocalKey`** and related resolution).

## Value interpolation

| Syntax | Behavior |
|--------|----------|
| **`${NAME}`** | Replaced with the resolved string value of **`NAME`**. A leading **`$$`** skips that match; a later pass turns **`$${x}`** into literal **`${x}`**. |
| **`$[NAME]`** | Splits the value at **`NAME`** by the default separator (comma) and expands each token; multiple **`$[...]`** placeholders produce a Cartesian product over those lists. |
| **Nesting** | Up to **10** rounds of **`${}`** / **`$[]`** expansion per **`String`** call. |

Interpolation regexes are compiled lazily on first use; accessors are **`ValStringKeyMatchReg()`**, **`ValStringsKeyMatchReg()`**, and **`ValStringKeyReplaceReg()`**.

## Errors

- **`tcfg.ErrNilConfData`**: method called on a **`nil *ConfData`**.  
- Missing keys and some nested-resolution failures may surface errors from **`github.com/choveylee/terror`** (for example **`ErrDataNotExist`**).

## License

Add your license here.
