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

1. 更新 Xcode build settings 里的 `MARKETING_VERSION` 和 `CURRENT_PROJECT_VERSION`。
2. 用 Release 配置构建、签名、notarize 并打包 DMG。
3. 准备 appcast 输入目录。

```bash
mkdir -p dist/appcast
cp dist/Momento.dmg "dist/appcast/Momento-${MARKETING_VERSION}.dmg"
```

4. 生成或更新 appcast。

```bash
SPARKLE_BIN="$(find "$HOME/Library/Developer/Xcode/DerivedData" -path '*/SourcePackages/artifacts/sparkle/Sparkle/bin' -type d | head -n 1)"
"$SPARKLE_BIN/generate_appcast" \
  --download-url-prefix "https://github.com/Seaony/Momento/releases/download/v${MARKETING_VERSION}/" \
  dist/appcast
```

5. 在 GitHub Releases 创建 `v${MARKETING_VERSION}`，上传同名 DMG。
6. 把 `dist/appcast/appcast.xml` 发布到 GitHub Pages 对应的 `https://seaony.github.io/Momento/appcast.xml`。
7. 启动旧版本 Momento，使用 `Momento > Check for Updates...` 验证能发现新版本。

## 注意事项

- 不要修改已发布的 DMG 后继续使用旧 appcast。包内容一变，Sparkle 签名就必须重新生成。
- 不要把 Sparkle private key 文件提交到仓库或 Release asset。
- 如果 GitHub Release asset 改名，必须重新生成 appcast，确保 enclosure URL 和 Release asset 完全一致。
