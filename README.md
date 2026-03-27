# tcfg

[English](#english) · [中文](#中文)

---

## English

**tcfg** is a small Go library that loads **INI** configuration together with **environment variables**, applies an optional **key prefix** and **`APP_NAME`**-based key scoping, and expands values using **`${key}`** and **`$[key]`** placeholders (with **`$${...}` / `$$[...]`** escapes for literal dollar signs).

### Requirements

- Go **1.26.1** or newer (see `go.mod`).

### Install

```bash
go get github.com/choveylee/tcfg
```

### Quick start

After import, the package runs **`init`**, loads **`<executable_basename>_config.ini`** (executable name lowercased, `-` → `_`), and exposes package-level helpers (`String`, `Bool`, `Int`, …) backed by a default **`ConfData`** instance. If loading fails, **`init` panics**.

```go
import "github.com/choveylee/tcfg"

func main() {
    v, err := tcfg.String("MY_KEY")
    if err != nil {
        // handle: missing key, parse error, etc.
    }
    _ = v
}
```

### Configuration file lookup

`loadFromFile` looks for `<executable_basename>_config.ini` by checking the **current working directory**, then the **executable’s directory**, then **ancestor directories** up to the filesystem root for both (same order as in code: work path, app path, then their parent chains). Relative `include "other.ini"` entries resolve from the including file’s directory.

### Environment variables

- Lookups consult **environment first**, then INI.  
- Keys may use **`SECTION::KEY`**; for env this maps to **`KEY_SECTION`** (see `EnvData`).

### Key prefix and `APP_NAME`

- **`GetKeyPrefix` / `SetKeyPrefix`** set a global prefix used when resolving keys (safe for concurrent read/write with the mutex-backed implementation).  
- If **`APP_NAME`** is set, keys can be resolved with an extra **normalized app** segment (uppercase, `-` → `_`) for localized/runtime-specific names.

### Value interpolation

| Syntax | Meaning |
|--------|---------|
| `${NAME}` | Replace with the string value of key `NAME` (after env/ini resolution). Prefix `$$` skips expansion; pass-through is normalized in a final step so **`$${x}`** becomes **`${x}`** literally. |
| `$[NAME]` | Replace with **each** value of the list stored at `NAME` (split by comma by default), producing a Cartesian product when multiple `$[]` appear (see existing tests / `STRING_E` example). |
| **Nesting** | Up to **10** rounds of `${}` / `$[]` expansion per **`String`** call. |

### Using `IniMgr` / `IniData`

For **INI-only** access (no package `init`, no env+INI merge), use `IniMgr`:

```go
mgr := &tcfg.IniMgr{}
ini, err := mgr.ParseFile("app.ini")
if err != nil {
    log.Fatal(err)
}
val, ok := ini.GetString("MY_KEY")
```

`ParseConfig` can build data from in-memory `[]*tcfg.Config` rows. **`ConfData` struct fields are not exported**, so code outside this module cannot assemble a custom `ConfData` that combines env, INI, and `${}` / `$[]` expansion; use the **package-level** `String` / `Bool` / … helpers for that behavior, or fork and add an exported constructor.

### Errors

- **`tcfg.ErrNilConfData`**: calling methods on a **`nil *ConfData`**.  
- Key missing / nested resolution failures may use **`github.com/choveylee/terror`** (e.g. `ErrDataNotExist`).

### Testing

```bash
go test ./...
```

### License

*(Add your license here.)*

---

## 中文

**tcfg** 是一个轻量 Go 库，用于在加载 **INI 文件** 的同时合并 **环境变量**，支持可选 **键前缀**、基于 **`APP_NAME`** 的键作用域，并在配置值里展开 **`${key}`**、**`$[key]`** 占位符（通过 **`$${...}` / `$$[...]`** 转义得到字面量 `$` 形式）。

### 环境要求

- **Go 1.26.1** 及以上（见 `go.mod`）。

### 安装

```bash
go get github.com/choveylee/tcfg
```

### 快速开始

`import` 后，包会在 **`init`** 中加载 `<可执行文件名>_config.ini`（文件名小写，`-` 转为 `_`），并用默认的 **`ConfData`** 提供包级函数（`String`、`Bool`、`Int` 等）。若加载失败，**`init` 会直接 `panic`**。

```go
import "github.com/choveylee/tcfg"

func main() {
    v, err := tcfg.String("MY_KEY")
    if err != nil {
        // 处理缺失键、解析错误等
    }
    _ = v
}
```

### 配置文件查找顺序

`loadFromFile` 按顺序查找 `<可执行文件名>_config.ini`：先 **当前工作目录** 与 **可执行文件目录**，再分别向**文件系统根目录**回溯上级目录。INI 里的 `include "其它.ini"` 会相对于**当前被解析文件**所在目录解析相对路径。

### 环境变量

- 读配置时 **先查环境变量，再查 INI**。  
- 支持 **`SECTION::KEY`** 形式；在环境变量侧会映射为 **`KEY_SECTION`**（见 `EnvData`）。

### 键前缀与 `APP_NAME`

- **`GetKeyPrefix` / `SetKeyPrefix`**：全局键前缀（实现上带读写锁，可与读取并发安全配合）。  
- 若配置了 **`APP_NAME`**，解析键时可使用规范化后的应用名段（大写、`-` → `_`），用于多应用/本地键名组合。

### 值内插值

| 语法 | 含义 |
|------|------|
| `${NAME}` | 替换为键 `NAME` 解析后的字符串（走环境变量与 INI）。前缀多写一个 `$` 可跳过本次展开；后处理会把 **`$${x}`** 还原成字面上的 **`${x}`**。 |
| `$[NAME]` | 将 `NAME` 处的**列表值**（默认按逗号拆分）展开；多个 **`$[]`** 会做笛卡尔组合（可参考测试与示例里的 `STRING_E`）。 |
| **嵌套** | 每次 **`String`** 调用内，最多 **10** 轮 `${}` / `$[]` 展开。 |

### 直接使用 `IniMgr` / `IniData`

若不想依赖包级 `init` 或需要只读 INI，可用 **`IniMgr.ParseFile`** / **`ParseConfig`** 得到 **`IniData`**，再调用 **`GetString`、`GetInt`** 等。

```go
mgr := &tcfg.IniMgr{}
ini, err := mgr.ParseFile("app.ini")
if err != nil {
    log.Fatal(err)
}
v, ok := ini.GetString("MY_KEY")
_ = v; _ = ok
```

说明：**`ConfData` 的字段未导出**，外部模块无法自行拼装 `ConfData`；合并「环境 + INI + 插值」请使用包级 API，或在同仓库内扩展导出构造函数（若你有此需求可自行添加）。

### 错误说明

- **`tcfg.ErrNilConfData`**：在 **`nil *ConfData`** 上调用方法。  
- 缺失键、嵌套解析失败等可能来自 **`github.com/choveylee/terror`**（例如 `ErrDataNotExist`)。

### 测试

```bash
go test ./...
```

### 许可证

*（在此补充你的许可证声明。）*
