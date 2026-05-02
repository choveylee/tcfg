# tcfg

[![Go Reference](https://pkg.go.dev/badge/github.com/choveylee/tcfg.svg)](https://pkg.go.dev/github.com/choveylee/tcfg)

`tcfg` is a Go library for loading configuration from **INI** files and **environment variables**. It supports optional **key prefixes**, **`APP_NAME`**-based scoping, and value expansion through **`${key}`** and **`$[key]`** placeholders. Literal dollar signs may be expressed with **`$${...}`** and **`$$[...]`**.

## Features

- Environment values take precedence over INI values for the same logical key
- INI keys may use **`SECTION::KEY`**; environment lookups use **`KEY_SECTION`**
- Global key prefixes are supported through **`GetKeyPrefix`** and **`SetKeyPrefix`**
- Per-application key scoping is available when **`APP_NAME`** is set
- **`${name}`** interpolation and **`$[name]`** list expansion are supported
- **`IniMgr`** and **`IniData`** may be used directly without relying on package **`init`**

## Requirements

- [Go](https://go.dev/dl/) **1.25.0** or later (see [`go.mod`](go.mod))

## Installation

```bash
go get github.com/choveylee/tcfg
```

## Dependencies

- [`github.com/choveylee/terror`](https://github.com/choveylee/terror) — provides shared error values such as `ErrDataNotExist`

## Usage

### Package-level API

Importing the package initializes the default **`ConfData`** instance, resolves a configuration file named **`<executable_basename>_config.ini`** (with the basename lowercased and `-` replaced by `_`), and binds package-level helpers such as `tcfg.String` and `tcfg.Bool` to that instance. The search checks the current working directory first and then the directory containing the current executable (via **`os.Executable`**). If initialization fails, package initialization panics.

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

The fields of **`ConfData`** are not exported. Use the package-level helpers, or work with **`IniMgr`** and **`IniData`** directly when explicit control over loading behavior is required.

### INI-only (`IniMgr`)

Use the following approach when only INI parsing is required and automatic environment merging is not desired:

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

Use **`ParseConfig`** to build data from `[]*tcfg.Config`. The parser supports sections, **`include "path"`** directives (resolved relative to the including file, with circular include chains reported as errors), UTF-8 BOM, and line comments beginning with `#` or `;`.

## Configuration file discovery

The default loader searches for **`<executable_basename>_config.ini`** in the following order:

1. Current working directory  
2. Directory containing the current executable (via **`os.Executable`**)  
3. Ancestor directories of (1), then of (2), up toward the filesystem root  

The first path that resolves to a **regular file** is used. If no configuration file is found, loading proceeds with empty INI data.

## Environment variables

- Resolution order is **environment first**, followed by **INI**.  
- Keys may use **`SECTION::KEY`**; for environment variables this maps to **`KEY_SECTION`** (see **`EnvData`**).

## Key prefix and `APP_NAME`

- **`GetKeyPrefix`** and **`SetKeyPrefix`** configure a global prefix for key resolution and are safe for concurrent use.  
- If **`APP_NAME`** is set, keys may include an additional normalized segment (uppercase, with `-` replaced by `_`), for example through **`LocalKey`**.

## Value interpolation

| Syntax | Behavior |
|--------|----------|
| **`${NAME}`** | Replaced with the resolved string value of **`NAME`**. A doubled **`$$`** suppresses expansion, and a final unescape pass turns **`$${x}`** into literal **`${x}`**. |
| **`$[NAME]`** | Resolves **`NAME`** as a list (default separator: comma). Multiple **`$[...]`** placeholders produce a Cartesian product of all referenced lists. |
| **Nesting** | Up to **10** rounds of **`${}`** and **`$[]`** expansion are performed for each **`String`** call. |

The package exposes the following precompiled patterns: **`ValStringKeyMatchReg`**, **`ValStringsKeyMatchReg`**, and **`ValStringKeyReplaceReg`**.

## Errors

- **`tcfg.ErrNilConfData`** is returned when an operation is invoked on a **`nil *ConfData`** receiver.  
- Missing keys and certain nested-resolution failures may use error values from **`github.com/choveylee/terror`**, such as **`ErrDataNotExist`**.

## License

This repository does not currently declare a license.
