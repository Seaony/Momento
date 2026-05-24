# Momento 更新发布流程

Momento 使用 Sparkle 2 做 App 内更新，GitHub Releases 只负责托管发布包和 appcast 里指向的下载文件。

## 当前配置

- Sparkle feed: `https://seaony.github.io/Momento/appcast.xml`
- 发布仓库: `Seaony/Momento`
- 更新签名 public key 已写入 `Momento/Info.plist` 的 `SUPublicEDKey`
- private key 由 Sparkle `generate_keys` 保存到当前 macOS 登录 Keychain，不提交到仓库
- App Sandbox 已通过 `Momento/Momento.entitlements` 给 Sparkle installer XPC service 放行

## 首次机器准备

如果换机器发布，需要把 Sparkle private key 导入到登录 Keychain。

```bash
SPARKLE_BIN="$(find "$HOME/Library/Developer/Xcode/DerivedData" -path '*/SourcePackages/artifacts/sparkle/Sparkle/bin' -type d | head -n 1)"
"$SPARKLE_BIN/generate_keys" -f /path/to/private-key-file
```

如果只是确认当前机器上的 public key：

```bash
SPARKLE_BIN="$(find "$HOME/Library/Developer/Xcode/DerivedData" -path '*/SourcePackages/artifacts/sparkle/Sparkle/bin' -type d | head -n 1)"
"$SPARKLE_BIN/generate_keys" -p
```

## 发布新版本

推荐使用仓库里的发布准备脚本。脚本只准备本地发布产物，不会自动 commit、tag、push，也不会创建 GitHub Release。

```bash
scripts/prepare-release.sh 1.0.1 2
```

脚本会完成：

- 更新 Xcode build settings 里的 `MARKETING_VERSION` 和 `CURRENT_PROJECT_VERSION`
- 用 Release 配置构建 App
- 生成 `dist/Momento-<version>.dmg`
- 使用 Sparkle `sign_update` 生成更新包签名
- 更新根目录 `appcast.xml`
- 校验 DMG、签名、App 内版本、`appcast.xml` 和 `git diff --check`

脚本通过后，再执行：

```bash
git add Momento.xcodeproj/project.pbxproj appcast.xml
git commit -m "chore: release 1.0.1"
git tag 1.0.1
git push origin master
git push origin 1.0.1
```

然后在 GitHub Releases 创建 `1.0.1`，上传：

```text
dist/Momento-1.0.1.dmg
```

GitHub Pages 刷新后，确认 feed 可访问：

```bash
curl -fsSL https://seaony.github.io/Momento/appcast.xml
```

最后启动旧版本 Momento，使用 `Momento > Check for Updates...` 或等待自动检查，验证能发现新版本。

如果需要把 Release tag 改成带 `v` 的格式，可以临时传环境变量：

```bash
MOMENTO_RELEASE_TAG=v1.0.1 scripts/prepare-release.sh 1.0.1 2
```

## 注意事项

- 不要修改已发布的 DMG 后继续使用旧 appcast。包内容一变，Sparkle 签名就必须重新生成。
- 不要把 Sparkle private key 文件提交到仓库或 Release asset。
- 如果 GitHub Release asset 改名，必须重新生成 appcast，确保 enclosure URL 和 Release asset 完全一致。
