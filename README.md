# tcfg

[![Go Reference](https://pkg.go.dev/badge/github.com/choveylee/tcfg.svg)](https://pkg.go.dev/github.com/choveylee/tcfg)

Go library that loads **INI** files together with **environment variables**, applies an optional **key prefix** and **`APP_NAME`**-based scoping, and expands values using **`${key}`** and **`$[key]`** placeholders (with **`$${...}` / `$$[...]`** escapes for literal `$`).

## Features

- **Environment first**, then **INI**, for the same logical key
- **`SECTION::KEY`** in INI; environment lookups use **`KEY_SECTION`**
- Optional global key prefix (**`GetKeyPrefix` / `SetKeyPrefix`**) and per-app keys when **`APP_NAME`** is set
- **`${name}`** interpolation and **`$[name]`** list expansion (Cartesian product when multiple **`$[]`** appear)
- **`IniMgr` / `IniData`** for file or in-memory INI without relying on package **`init`**

## Requirements

- [Go](https://go.dev/dl/) **1.25.0** or later (see [`go.mod`](go.mod))

## Installation

```bash
go get github.com/choveylee/tcfg
```

## Dependencies

- [`github.com/choveylee/terror`](https://github.com/choveylee/terror) — errors such as `ErrDataNotExist` for missing keys in some paths

## Usage

### Package-level API

Importing the package runs **`init`**, which resolves **`<executable_basename>_config.ini`** (basename lowercased, `-` → `_`) and binds package-level helpers (`tcfg.String`, `tcfg.Bool`, …) to a default **`ConfData`**. **`init` panics** if loading fails with an error (for example I/O failure or a parse error on an existing config file).

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

**`ConfData` fields are not exported**; use these helpers (or fork and expose a constructor) for env + INI + `${}` / `$[]` behavior.

### INI-only (`IniMgr`)

For INI parsing only—no package **`init`**, no automatic env merge:

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

Use **`ParseConfig`** to build data from `[]*tcfg.Config`. The parser supports sections, **`include "path"`** (relative paths are resolved from the including file’s directory), UTF-8 BOM, and line comments (`#`, `;`).

## Configuration file discovery

The default loader searches for **`<executable_basename>_config.ini`** in this order:

1. Current working directory  
2. Directory containing the executable  
3. Ancestor directories of (1), then of (2), up toward the filesystem root  

The first path that refers to a **regular file** wins. If **no** file is found, loading completes with empty INI data (see source).

## Environment variables

- Resolution order: **environment**, then **INI**.  
- Keys may use **`SECTION::KEY`**; for environment variables this maps to **`KEY_SECTION`** (see **`EnvData`**).

## Key prefix and `APP_NAME`

- **`GetKeyPrefix`** / **`SetKeyPrefix`** set a global prefix for key resolution (safe for concurrent use).  
- If **`APP_NAME`** is set, keys can include an extra normalized segment (uppercase, `-` → `_`), e.g. via **`LocalKey`**.

## Value interpolation

| Syntax | Behavior |
|--------|----------|
| **`${NAME}`** | Replaced with the resolved string value of **`NAME`**. A doubled **`$$`** skips expansion; a final pass turns **`$${x}`** into literal **`${x}`**. |
| **`$[NAME]`** | Uses the list at **`NAME`** (default separator: comma); multiple **`$[...]`** yield a Cartesian product over those lists. |
| **Nesting** | Up to **10** rounds of **`${}`** / **`$[]`** expansion per **`String`** call. |

Precompiled patterns (package variables): **`ValStringKeyMatchReg`**, **`ValStringsKeyMatchReg`**, **`ValStringKeyReplaceReg`**.

## Errors

- **`tcfg.ErrNilConfData`** — returned when a method is called on a **`nil *ConfData`** receiver.  
- Missing keys and some nested-resolution failures may use **`github.com/choveylee/terror`** (e.g. **`ErrDataNotExist`**).

## License

Add your license here.
