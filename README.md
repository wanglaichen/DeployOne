# WebDown APK 管理网站

这是一个 Go 编写的 APK 包上传、浏览、下载和二维码分享网站。

## 目录结构

```text
.
├── build/              # 构建脚本和构建输出目录
├── cmd/webdown/        # 程序入口
├── config/             # 服务配置文件
├── internal/           # 后端业务代码
├── web/                # 页面模板和静态资源
└── data/               # 运行时数据，首次启动后自动创建
```

## 配置

修改 `config/config.json`：

```json
{
  "host": "0.0.0.0",
  "port": 8080,
  "public_base_url": "http://localhost:8080",
  "upload_dir": "data/uploads",
  "data_file": "data/apks.json",
  "max_upload_mb": 2048
}
```

- `host`：监听地址。
- `port`：网站端口。
- `public_base_url`：生成下载二维码时使用的服务器外部访问地址，部署到服务器后要改成真实域名或 IP。
- `upload_dir`：APK 文件保存目录。
- `data_file`：APK 信息保存文件。
- `max_upload_mb`：单个 APK 最大上传大小。
- `admin_user` / `admin_password`：管理后台账号，也可用环境变量 `WEBDOWN_ADMIN_USER`、`WEBDOWN_ADMIN_PASSWORD` 覆盖，默认均为 `admin`。

## 管理后台

首页右上角点击 **管理登录**，默认账号 `admin` / `admin`。

登录后可：

- 删除指定文件
- 清空当前分类下的全部文件
- 清空全部文件
- 刷新列表，自动移除磁盘上已被人工删除的失效记录

服务**每次启动**时也会自动检查并清理失效记录。各分类列表旁的 **刷新列表** 按钮可手动同步。

生产环境请修改管理员密码，例如：

```powershell
$env:WEBDOWN_ADMIN_USER = "admin"
$env:WEBDOWN_ADMIN_PASSWORD = "your-strong-password"
.\build\release-win2008\webdown.exe -config .\config\config.json
```

## 本地运行

```powershell
.\build\run.ps1
```

打开 `http://localhost:8080`。

也可以手动运行：

```powershell
go mod tidy
go run .\cmd\webdown -config .\config\config.json
```

## 构建

```powershell
.\build\build.ps1
```

生成完整发布目录在 `build/release`，包含：

```text
build/release/
├── webdown.exe
├── run.ps1
├── config/config.json
├── config/config.default.json
├── web/
└── data/uploads/
```

注意：再次执行 `.\build\build.ps1` 时会保留 `build/release/data` 和已有的 `build/release/config/config.json`，上传过的 APK 文件和 `data/apks.json` 列表不会被构建脚本删除。

构建后使用 exe 运行：

```powershell
.\build\run.ps1 -UseBinary
```

也可以进入发布目录运行：

```powershell
cd .\build\release
.\run.ps1
```

指定其他配置文件：

```powershell
.\build\run.ps1 -ConfigPath .\config\config.json
```

## 发布

把 `build/release` 目录拷贝到服务器即可。发布后修改 `build/release/config/config.json` 里的 `public_base_url` 为服务器真实访问地址，二维码会按该地址拼接 `/download/{包ID}`。

上传后的列表保存在 `data_file` 配置对应的文件中，默认是 `build/release/data/apks.json`；APK 文件保存在 `upload_dir` 配置对应的目录中，默认是 `build/release/data/uploads`。发布或迁移时要一起保留 `data` 目录。

## Windows Server 2008

Windows Server 2008 不能使用 Go 1.22 编译出来的程序，否则可能启动后直接崩溃并显示 `Exception 0xc0000005`。

请安装 Go 1.20.x，推荐 Go 1.20.14，然后执行：

```powershell
.\build\build-win2008.ps1
```

生成目录：

```text
build/release-win2008/
```

把 `build/release-win2008` 拷贝到 Windows Server 2008，双击 `run.bat` 运行。不要使用 Go 1.21、Go 1.22 或更高版本编译给 Windows Server 2008 使用。
