# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

**wx_channel** (微信视频号下载助手) is a Windows-only Go application that intercepts WeChat Channel video pages via HTTP proxy (SunnyNet), injects JavaScript to add download buttons, and handles video downloading/decryption. It includes a web console at `http://localhost:2025/console` and an optional Hub Server for centralized device management.

## Build & Run

```bash
# Build (requires Go 1.23+, CGO enabled, TDM-GCC on Windows)
# IMPORTANT: Use goproxy.cn for Chinese network environments
GOPROXY=https://goproxy.cn,direct CGO_LDFLAGS="-Wl,--allow-multiple-definition -static" go build -ldflags="-s -w -extldflags '-static'" -o wx_channel.exe

# Run
./wx_channel.exe           # Default port 2025
./wx_channel.exe -p 8080   # Custom port

# Windows resources (icon, version info)
go install github.com/tc-hib/go-winres@latest
go-winres make   # Generates rsrc_windows_*.syso, auto-embedded on build
```

Build flags explained:
- `CGO_LDFLAGS="-Wl,--allow-multiple-definition"` — required because `mattn/go-sqlite3` and `go-llsqlite/crawshaw` (indirect dep from gopeed/anacrolix) both embed sqlite3 C source
- `-static` and `-extldflags '-static'` — statically link MinGW runtime (libwinpthread, libgcc, libstdc++) to avoid DLL dependencies, ensuring the exe runs standalone without needing MinGW DLLs

## Testing

```bash
go test ./...                          # All tests
go test -v ./internal/config           # Single package
go test -v -run TestFunctionName ./internal/utils  # Single test
```

Tests exist for: `config`, `database`, `models`, `storage`, `utils`, `websocket`, `hub_server/middleware`, `hub_server/ws`. Some handler/router tests have build failures due to SunnyNet CGO dependencies.

## Architecture

### Request Flow

```
main.go -> cmd.Execute() -> config.Load() -> app.NewApp(cfg) -> app.Run()
                                                                  |
                                                      SunnyNet proxy (port 2025)
                                                                  |
                                                Request Interceptor Chain (first match wins):
                                                  StaticFileHandler -> APIRouter -> APIHandler ->
                                                  UploadHandler -> RecordHandler -> BatchHandler ->
                                                  CommentHandler
                                                                  |
                                                Response Interceptor Chain:
                                                  ScriptHandler (injects JS into WeChat pages)
```

All handlers implement `router.Interceptor` — return `true` to stop the chain, `false` to pass.

### Key Packages

| Package | Purpose |
|---------|---------|
| `internal/app` | App struct — wires all components, runs SunnyNet proxy, manages interceptor chains |
| `internal/handlers` | HTTP interceptors: `script.go` (JS injection, most complex ~1500 lines), `upload.go` (chunked download), `batch.go`, `api.go` |
| `internal/router` | REST API routes (`api_routes.go`), SunnyNet-to-http adapter (`adapter.go`) |
| `internal/services` | Business logic: `gopeed_service.go` (download engine), `queue_service.go`, `download_record_service.go` |
| `internal/database` | SQLite via GORM (WAL mode). Repository pattern: `browse_repository.go`, `download_repository.go`, etc. |
| `internal/websocket` | WebSocket hub for real-time console updates. Load-balanced client selection. |
| `internal/config` | Viper-based config. Priority: DB settings > env vars (`WX_CHANNEL_*`) > config.yaml > defaults |
| `internal/assets` | `go:embed` for JS injection scripts (`inject/*.js`) and SSL cert |
| `pkg/decrypt` | Video decryption (AES/RSA) |
| `hub_server/` | Separate Hub Server app: Vue 3 frontend + Go backend for multi-device management |

### JavaScript Injection System

ScriptHandler intercepts WeChat JS responses and injects code that fires events via an event bus:
- `PCFlowLoaded` — video list loaded
- `FeedProfileLoaded` — single video details
- `BeforeDownloadMedia` / `MediaDownloaded` — video fetch lifecycle

Injection scripts are in `internal/assets/inject/`. The event bus (`eventbus.js` + `mitt`) is the bridge between injected JS and Go handlers.

### Dependency Injection

All components are constructor-injected in `app.NewApp()`. Services are singletons shared across handlers. No DI framework — manual wiring.

### Configuration

Dynamic config with runtime reloading via DB settings table. Always use `config.Get()` (not the startup instance) to get current values. Key config files: `config.yaml.example`, `config.yaml.full`.

### Ports

- **2025**: Main proxy + web console (`/console`)
- **2025+1**: WebSocket server
- **9090**: Prometheus metrics (if `metrics_enabled: true`)
- **8080**: Hub Server frontend (separate app)

## Version

Version is hardcoded in `internal/version/version.go`. Windows metadata in `winres/winres.json`. Update both when releasing.

## Platform Constraints

- **Windows-only**: SunnyNet library requires Windows + process injection into `WeChatAppEx.exe`
- **CGO required**: Multiple deps need CGO (sqlite3, SunnyNet)
- WeChat JS patterns break on WeChat updates — check `dev-docs/FIX_HISTORY.md` for past fixes
