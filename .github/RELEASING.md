# 发版

版本号只写在 `app/mygo.json` 的 `version`。发布说明写在 `app/CHANGELOG.md` 的 `## <版本号>` 一节，它会成为 GitHub Release 的正文和应用内更新窗口里的说明。

安装包和更新文件都发在本仓库的 GitHub Releases。应用按 `app/mygo.json` 里的 `updates.github` 和 `updates.publicKey` 检查并校验更新。

## 在 Mac 上本地发版（1.0.0 就是这样发的）

需要：

- 钥匙串里有 `Developer ID Application` 证书（`security find-identity -v -p codesigning` 能看到）。
- 公证凭据已存成钥匙串配置 `notary`（`xcrun notarytool store-credentials notary`）。
- 更新签名私钥在 `~/Library/Application Support/mygo/update-keys/mygo-update.key`，由 `go tool mygo keygen` 生成。私钥不进仓库，另存一份到密码管理器；丢了它，装好的应用就再也收不到更新。
- `brew install makensis`，用来打 Windows 安装程序。
- `gh` 已登录，有本仓库的写权限。

在 `app/` 里执行。签名身份和公证只在这次构建时临时写进 `mygo.json`，构建完还原，这样别人克隆源码构建时不会因为没有证书而失败：

```sh
cp mygo.json /tmp/mygo.json.bak
jq '.macos.signingIdentity = "Developer ID Application: SHEN WANG (JJJKGPY7T7)" | .macos.notarize = {keychainProfile: "notary"}' /tmp/mygo.json.bak > mygo.json
MYGO_UPDATER_PRIVATE_KEY="$(cat "$HOME/Library/Application Support/mygo/update-keys/mygo-update.key")" \
  go tool mygo build -platform darwin/universal,windows/amd64,windows/arm64,linux/amd64,linux/arm64 -upload
cp /tmp/mygo.json.bak mygo.json
```

这一步会签名并公证 macOS 的 DMG（公证一次，大约一分钟），交叉编译 Windows 和 Linux，签好三个平台的更新包，然后上传到 `v<版本号>` 的草稿 Release。

发布前检查：

- `xcrun stapler validate` 和 `spctl -a -t open --context context:primary-signature -vv` 检查 DMG，应该显示 `Notarized Developer ID`。
- 从 DMG 里拷出 App 启动一次，看主窗口和日志。
- 草稿里应该有 DMG、两个 Windows 安装程序、DEB、RPM、各平台 tar.gz、`update-*.json`、`install.sh`、`install.ps1` 和 `SHA256SUMS`。

检查都过了就直接公开：`gh release edit v<版本号> --repo Seaony/Momento --draft=false --latest`，再确认 `https://github.com/Seaony/Momento/releases/latest/download/update-darwin-universal.json` 能下载。

## GitHub Actions

[release.yml](workflows/release.yml) 在推送版本标签或手动触发时运行。仓库密钥配齐之前，它只保存未签名的构建产物，不上传 Release。要让它签名并上传草稿，需要配置：`MYGO_UPDATER_PRIVATE_KEY`；`MACOS_CERTIFICATE`、`MACOS_CERTIFICATE_PASSWORD`、`APPLE_ID`、`APPLE_TEAM_ID`、`APPLE_APP_PASSWORD`；`WINDOWS_CERTIFICATE`、`WINDOWS_CERTIFICATE_PASSWORD`。

## 还没在真机上验证的

Windows 和 Linux 的安装、标题栏、窗口按钮、导入导出、往文件管理器拖文件，以及三个平台的开机启动，都还没在真机上检查过。交叉编译成功或离线截图正常，不能代替这些检查。
