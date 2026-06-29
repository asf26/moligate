# 助手版本发布与更新配置

本文档说明「系统设置 -> QQ 群 -> 助手版本管理」的用途，以及助手打包、签名、上传和后台字段填写方式。

## 后台配置位置

后台进入：

```text
系统设置 -> QQ 群 -> 助手版本管理
```

字段含义：

- `助手版本`：当前发布给客户端的版本号，例如 `0.1.1`。建议使用 SemVer 格式。
- `强制更新`：客户端是否必须更新。
- `助手发布说明`：版本更新说明，会返回给客户端。
- `macOS 下载地址`：macOS 自动更新包的公网 URL。
- `macOS 签名`：macOS 更新包对应 `.sig` 文件的文本内容。
- `Windows 下载地址`：Windows x64 自动更新包或安装包的公网 URL。
- `Windows 签名`：Windows 更新包对应 `.sig` 文件的文本内容。
- `发布时间戳`：Unix 秒级时间戳，例如 `1710000000`。

签名字段填的是 `.sig` 文件内容，不是 `.sig` 文件地址。

## 对外接口

普通版本检查接口：

```http
GET /api/assistant/version
```

返回示例：

```json
{
  "version": "0.1.1",
  "force_update": false,
  "release_notes": "修复钱包汇率和版本检查",
  "mac_download_url": "https://cdn.example.com/assistant/0.1.1/魔力门助手_0.1.1.app.tar.gz",
  "mac_signature": "mac .sig 文件内容",
  "win_download_url": "https://cdn.example.com/assistant/0.1.1/魔力门助手_0.1.1_x64-setup.exe",
  "win_signature": "windows .sig 文件内容",
  "published_at": 1710000000
}
```

Tauri 自动更新接口：

```http
GET /api/assistant/version?target=darwin&arch=aarch64&current_version=0.1.0
GET /api/assistant/version?target=darwin&arch=x86_64&current_version=0.1.0
GET /api/assistant/version?target=windows&arch=x86_64&current_version=0.1.0
```

有新版本时返回：

```json
{
  "version": "0.2.0",
  "url": "https://cdn.example.com/assistant/0.2.0/魔力门助手.app.tar.gz",
  "signature": "对应 .sig 文件内容",
  "notes": "自动更新",
  "pub_date": "2024-03-09T16:00:00Z"
}
```

没有新版本、版本号为空、下载地址为空或签名为空时返回 `204 No Content`。

## 签名是什么

Tauri 更新器需要用签名确认更新包来自可信来源。签名分两部分：

- 公钥：写进助手项目的 `tauri.conf.json`，客户端用它校验更新包。
- 私钥：打包时使用，用来给更新包生成 `.sig`，必须安全保存，不能提交到 Git，也不能放进本项目后台配置。

如果私钥丢失，已经安装的旧客户端将无法校验你后续发布的新更新包，需要重新发一个内置新公钥的安装包。

## 首次生成签名密钥

在助手项目中执行，密钥只需要生成一次：

```bash
bunx tauri signer generate -w ~/.tauri/moligate-assistant.key
```

如果设置了密码，要记住这个密码。命令会输出公钥，把公钥配置到助手项目的 Tauri 配置里，示例：

```json
{
  "bundle": {
    "createUpdaterArtifacts": true
  },
  "plugins": {
    "updater": {
      "pubkey": "这里填生成出来的公钥",
      "endpoints": [
        "https://你的域名/api/assistant/version?target={{target}}&arch={{arch}}&current_version={{current_version}}"
      ]
    }
  }
}
```

`createUpdaterArtifacts` 必须开启，否则构建时不会生成自动更新需要的包和 `.sig` 文件。

## 打包时必须带签名私钥

每次正式打包前，先在当前终端设置签名私钥环境变量：

```bash
export TAURI_SIGNING_PRIVATE_KEY="$(cat ~/.tauri/moligate-assistant.key)"
export TAURI_SIGNING_PRIVATE_KEY_PASSWORD="你的私钥密码，没有密码则留空"
```

然后执行助手项目自己的构建命令，例如：

```bash
bunx tauri build
```

如果助手项目封装了脚本，比如 `bun run build:tauri`，使用项目脚本也可以，关键是构建进程必须能读取到 `TAURI_SIGNING_PRIVATE_KEY`。

## 打包后文件在哪里

Tauri 默认会把产物放在助手项目的：

```text
src-tauri/target/release/bundle/
```

常见产物位置：

- macOS：
  - `src-tauri/target/release/bundle/macos/魔力门助手.app`
  - `src-tauri/target/release/bundle/macos/魔力门助手.app.tar.gz`
  - `src-tauri/target/release/bundle/macos/魔力门助手.app.tar.gz.sig`
- Windows NSIS：
  - `src-tauri/target/release/bundle/nsis/魔力门助手_0.1.1_x64-setup.exe`
  - `src-tauri/target/release/bundle/nsis/魔力门助手_0.1.1_x64-setup.exe.sig`
- Windows MSI：
  - `src-tauri/target/release/bundle/msi/魔力门助手_0.1.1_x64.msi`
  - `src-tauri/target/release/bundle/msi/魔力门助手_0.1.1_x64.msi.sig`

实际文件名会受助手项目的 `productName`、版本号和 bundle 配置影响，以构建目录里的真实文件为准。

## 文件放哪里

安装包和更新包需要放到客户端能访问的公网位置，比如 CDN、对象存储、服务器静态目录或 GitHub Releases。

推荐目录结构：

```text
assistant/
  0.1.1/
    魔力门助手_0.1.1_universal.dmg
    魔力门助手.app.tar.gz
    魔力门助手.app.tar.gz.sig
    魔力门助手_0.1.1_x64-setup.exe
    魔力门助手_0.1.1_x64-setup.exe.sig
```

后台字段填写方式：

- `macOS 下载地址`：填 macOS 自动更新包 URL，通常是 `.app.tar.gz`。
- `macOS 签名`：打开 `.app.tar.gz.sig`，把里面的文本完整复制进去。
- `Windows 下载地址`：填 Windows 更新包 URL。当前后台按 Windows x64 返回这一个地址，通常填 `.exe`；如果助手项目配置成 v1 兼容更新包，则填对应 `.zip`。
- `Windows 签名`：打开 Windows 更新包对应的 `.sig`，把里面的文本完整复制进去。

`.sig` 文件可以和安装包一起上传到 CDN 旁边留档，但后台接口不会让客户端下载 `.sig` 文件；接口会直接把后台保存的签名内容放到 `signature` 字段。

## 发布检查清单

1. 更新助手项目版本号。
2. 确认 `tauri.conf.json` 里已配置 updater `pubkey` 和 endpoint。
3. 打包终端设置 `TAURI_SIGNING_PRIVATE_KEY`。
4. 执行 Tauri 构建命令。
5. 在 `src-tauri/target/release/bundle/` 找到更新包和对应 `.sig`。
6. 把更新包上传到 CDN 或静态文件服务器。
7. 在后台「助手版本管理」填写版本号、下载地址、签名内容、发布说明和发布时间戳。
8. 访问 `/api/assistant/version` 检查普通版本接口。
9. 访问带 `target`、`arch`、`current_version` 的 URL 检查 Tauri 更新接口。

## 重要安全要求

- 私钥文件，例如 `~/.tauri/moligate-assistant.key`，不能提交到任何代码仓库。
- 私钥密码不能写进仓库、文档示例、后台系统设置或前端代码。
- 后台只保存公开下载 URL 和 `.sig` 文本内容。
- 如果换了签名公私钥，旧客户端无法用旧公钥校验新私钥签出来的更新包，需要先发布一个内置新公钥的过渡版本。
