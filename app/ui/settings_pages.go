package ui

import (
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"time"

	"github.com/Seaony/Momento/app/internal/assets"
	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
)

var settingsPlatform = func() string { return runtime.GOOS }

func settingsError(c *ui.Context, text string) {
	p := paletteFor(c.Theme())
	revealMotion(ui.Text(c.Key("settings-error"), text).FontSize(fontSecondary).TextColor(p.Danger).Padding(space4, space16, space12, space16)).Role(ui.RoleAlert)
}

// 错误原因显示在第一行，系统给的原始错误放在第二行小字里，不遮原因。
func settingsErrorDetail(c *ui.Context, text, detail string) {
	p := paletteFor(c.Theme())
	revealMotion(ui.Column(c.Key("settings-error"))).FillWidth().Padding(space4, space16, space12, space16).Gap(space4).Role(ui.RoleAlert).Children(func() {
		ui.Text(c, text).FontSize(fontSecondary).TextColor(p.Danger)
		ui.Text(c, detail).FontSize(fontSecondary).TextColor(p.TextSecondary).Selectable()
	})
}

// 带一行说明的开关，可以禁用；说明为空时和普通开关一样。
func (a *App) switchSettingNote(c *ui.Context, key, note string, value *bool, disabled bool) {
	if note == "" {
		a.switchSetting(c, key, value, disabled)
		return
	}
	ui.Row(c.Key(key+"-note")).FillWidth().Padding(space12, space16).Gap(space12).AlignItems(ui.Center).Children(func() {
		ui.Column(c).Grow(1).MinWidth(0).Gap(space4).Children(func() {
			ui.Text(c, a.T(key))
			ui.Text(c, note).FontSize(fontSecondary).TextColor(paletteFor(c.Theme()).TextSecondary)
		})
		if ui.Switch(c.Key(key), value).Label(a.T(key)).Disabled(disabled).Changed() {
			a.SavePreferences()
		}
	})
}

func (a *App) switchSetting(c *ui.Context, key string, value *bool, disabled bool) {
	settingsRow(c, func() {
		ui.Text(c, a.T(key)).Grow(1).MinWidth(0)
		if ui.Switch(c.Key(key), value).Label(a.T(key)).Disabled(disabled).Changed() {
			a.SavePreferences()
		}
	})
}

// 带一行次要说明的开关，返回是否被切换；保存和联动由调用方处理。
func (a *App) switchSettingHint(c *ui.Context, key, hint string, value *bool) bool {
	changed := false
	// 两行文字的行和上面的分段设置一样用 space12 上下留白。
	ui.Row(c).FillWidth().Padding(space12, space16).Gap(space12).AlignItems(ui.Center).Children(func() {
		ui.Column(c).Grow(1).MinWidth(0).Gap(space4).Children(func() {
			ui.Text(c, a.T(key))
			ui.Text(c, a.T(hint)).FontSize(fontSecondary).TextColor(paletteFor(c.Theme()).TextSecondary)
		})
		changed = ui.Switch(c.Key(key), value).Label(a.T(key)).Changed()
	})
	return changed
}

func (a *App) stringSetting(c *ui.Context, key string, value *string, codes, labels []string) {
	translated := make([]string, len(labels))
	for i, label := range labels {
		translated[i] = a.T(label)
	}
	settingsRow(c.Key(key), func() {
		ui.Text(c, a.T(key)).Grow(1).MinWidth(0)
		choiceMenu(c.Key("choice"), a.T(key), max(0, slices.Index(codes, *value)), translated, func(i int) {
			*value = codes[i]
			a.SavePreferences()
		})
	})
}

func (a *App) boolSetting(c *ui.Context, key string, value *bool, firstValue bool, firstKey, secondKey string) {
	selected := 0
	if *value != firstValue {
		selected = 1
	}
	settingsRow(c.Key(key), func() {
		ui.Text(c, a.T(key)).Grow(1).MinWidth(0)
		choiceMenu(c.Key("choice"), a.T(key), selected, []string{a.T(firstKey), a.T(secondKey)}, func(i int) {
			*value = firstValue == (i == 0)
			a.SavePreferences()
		})
	})
}

func (a *App) appearanceSettings(c *ui.Context) {
	settingsSection(c, a.T("theme_title"), "", func() {
		ui.Column(c).FillWidth().Padding(space16).Children(func() { a.ThemeSwatches(c) })
	})
	settingsSection(c, "", "", func() {
		settingsRow(c, func() {
			ui.Text(c, a.T("ui_zoom")).Grow(1)
			label := "100%"
			codes := []float32{.8, .9, 1, 1.1, 1.25, 1.5}
			labels := []string{"80%", "90%", "100%", "110%", "125%", "150%"}
			for i, code := range codes {
				if a.Preferences.UIZoom == code {
					label = labels[i]
				}
			}
			zoom := settingsMenuButton(c.Key("ui-zoom"), a.T("ui_zoom"))
			zoom.Children(func() {
				ui.Text(c, label).FontFeatures("tnum").FontSize(fontNumber).SingleLine()
				symbol := appSymbol(c, "chevrons-up-down").Size(iconSmall, iconSmall).Shrink(0)
				symbol.TextColor(symbol.AnimateColor("symbol-color", symbolColor(c.Theme(), zoom.Hovered()), motionColor))
			}).Menu(func(menu *ui.Menu) {
				for i, choice := range labels {
					if menu.Item(choice).Radio(a.Preferences.UIZoom == codes[i]).Chosen() {
						a.Preferences.UIZoom = codes[i]
						a.SavePreferences()
					}
				}
			})
		})
		settingsSeparator(c)
		settingsRow(c, func() { ui.Text(c, a.T("language")).Grow(1); a.languageMenu(c.Key("language")) })
	})
}

func (a *App) sidebarSettings(c *ui.Context) {
	settingsSection(c, "", "", func() {
		a.stringSetting(c, "folder_double_click", &a.Preferences.FolderDoubleClick, []string{"rename", "expand"}, []string{"rename", "folder_expand"})
		settingsSeparator(c)
		enabled := a.Preferences.SmartFolders
		if a.switchSettingHint(c, "smart_folders_enabled", "smart_folders_enabled_hint", &enabled) {
			a.SetSmartFoldersEnabled(enabled)
		}
		settingsSeparator(c)
		a.switchSetting(c, "sidebar_counts", &a.Preferences.SidebarCounts, false)
	})
	settingsSection(c, a.T("sidebar_show"), "", func() {
		// 「全部」「标签管理」「回收站」永远显示，不列出来。
		items := []struct{ key, label string }{{"favorites", "favorites"}, {"recent", "recently_used"}, {"unfiled", "uncategorized"}, {"untagged", "untagged"}, {"random", "random_browse"}}
		ui.Column(c).FillWidth().Padding(space16).Gap(space12).Children(func() {
			for start := 0; start < len(items); start += 3 {
				ui.Row(c).FillWidth().Gap(space8).Children(func() {
					for i := start; i < min(start+3, len(items)); i++ {
						item := items[i]
						shown := !slices.Contains(a.Preferences.SidebarHidden, item.key)
						if settingsCheckbox(c.Key(item.key), &shown, a.T(item.label)).Grow(1).MinWidth(0).Changed() {
							a.setSidebarShown(item.key, shown)
						}
					}
					for i := min(start+3, len(items)); i < start+3; i++ {
						ui.Box(c).Grow(1).MinWidth(0)
					}
				})
			}
		})
		settingsSeparator(c)
		ui.Row(c).FillWidth().Padding(space16).Gap(space8).Children(func() {
			// 智能文件夹由上面的开关决定，这里不再重复；空位占住第三列，和上面三列对齐。
			for _, item := range []struct{ key, label string }{{"quick", "quick_access"}, {"folders", "folders"}} {
				shown := !slices.Contains(a.Preferences.SidebarHidden, item.key)
				if settingsCheckbox(c.Key(item.key), &shown, a.T(item.label)).Grow(1).MinWidth(0).Changed() {
					a.setSidebarShown(item.key, shown)
				}
			}
			ui.Box(c).Grow(1).MinWidth(0)
		})
	})
}

func (a *App) setSidebarShown(key string, shown bool) {
	a.Preferences.SidebarHidden = slices.DeleteFunc(a.Preferences.SidebarHidden, func(item string) bool { return item == key })
	if !shown {
		a.Preferences.SidebarHidden = append(a.Preferences.SidebarHidden, key)
	}
	a.SavePreferences()
}

func (a *App) controlsSettings(c *ui.Context) {
	settingsSection(c, "", "", func() {
		a.stringSetting(c, "double_click", &a.Preferences.DoubleClick, []string{"preview", "external"}, []string{"momento_preview", "default_app"})
		if settingsPlatform() == "darwin" {
			settingsSeparator(c)
			a.stringSetting(c, "space_key", &a.Preferences.SpaceKey, []string{"preview", "system"}, []string{"momento_preview", "system_preview"})
		}
		settingsSeparator(c)
		a.stringSetting(c, "preview_wheel", &a.Preferences.PreviewWheel, []string{"navigate", "zoom"}, []string{"wheel_navigate", "wheel_zoom"})
	})
}

func (a *App) previewSettings(c *ui.Context) {
	settingsSection(c, "", a.T("preview_smart_hint"), func() {
		a.stringSetting(c, "preview_fit", &a.Preferences.PreviewFit, []string{"smart", "fit"}, []string{"preview_smart", "preview_fit_window"})
	})
	settingsSection(c, "", "", func() {
		a.boolSetting(c, "preview_details", &a.Preferences.PreviewPixelated, false, "preview_smooth", "preview_pixelated")
		settingsSeparator(c)
		a.boolSetting(c, "preview_transparency", &a.Preferences.PreviewChecker, true, "preview_checker", "preview_solid")
	})
	settingsSection(c, "", "", func() {
		a.switchSetting(c, "hover_play_videos", &a.Preferences.HoverPlayVideos, false)
	})
}

func (a *App) importSettings(c *ui.Context) {
	p := paletteFor(c.Theme())
	settingsSection(c, "", "", func() {
		a.switchSetting(c, "select_imported", &a.Preferences.SelectImported, false)
		settingsSeparator(c)
		if a.switchSettingHint(c, "import_keywords", "import_keywords_hint", &a.Preferences.ImportKeywords) {
			a.SavePreferences()
		}
	})
	footer := a.T("auto_import_hint")
	if settingsPlatform() == "windows" {
		footer = a.T("auto_import_hint_windows")
	}
	settingsSection(c, a.T("auto_import"), footer, func() {
		if a.Library == nil {
			settingsRow(c, func() { ui.Text(c, a.T("open_library_first")).TextColor(p.TextSecondary) })
			settingsRow(c, func() { settingsButton(c, a.T("add_watch_folder")).Disabled(true) })
			return
		}
		libraryID := a.Library.Manifest.LibraryID
		rules := a.Preferences.AutoImport[libraryID]
		if len(rules) == 0 {
			fadeMotion(ui.Column(c.Key("auto-import-empty"))).FillWidth().Padding(space16, 0).Gap(space8).AlignItems(ui.Center).Children(func() {
				emptyArt(c.Key("art"), "no-watched", emptyArtSmall)
				ui.Text(c, a.T("auto_import_empty")).FontSize(fontSecondary).TextColor(p.TextTertiary)
			})
		}
		for i, rule := range rules {
			if i > 0 {
				settingsSeparator(c)
			}
			// 加进来的监视文件夹在列表里展开出来，移除时收起。
			revealMotion(ui.Column(c.Key(rule.Path))).FillWidth().Children(func() {
				settingsRow(c, func() {
					appIcon(c, "folder").Size(iconSize, iconSize).Shrink(0)
					ui.Text(c, rule.Path).SingleLine().Grow(1).MinWidth(0).Tooltip(rule.Path)
					enabled := rule.Enabled
					if ui.Switch(c.Key("enabled"), &enabled).Label(a.T("auto_import_enabled", rule.Path)).Changed() {
						rules[i].Enabled = enabled
						a.Preferences.AutoImport[libraryID] = rules
						a.SavePreferences()
					}
				})
				settingsRow(c, func() {
					ui.Text(c, a.T("auto_import_target")).Grow(1).MinWidth(0)
					// 菜单里和浏览器扩展选择器一样显示完整层级，同名子文件夹分得清。
					choices := a.importFolderChoices()
					folderName := choices[0].Label
					for _, choice := range choices {
						if choice.ID == rule.FolderID {
							folderName = choice.Label
						}
					}
					target := settingsMenuButton(c, a.T("auto_import_target")).Width(200)
					target.Children(func() {
						ui.Text(c, folderName).SingleLine().Grow(1).MinWidth(0)
						symbol := appSymbol(c, "chevrons-up-down").Size(iconSmall, iconSmall).Shrink(0)
						symbol.TextColor(symbol.AnimateColor("symbol-color", symbolColor(c.Theme(), target.Hovered()), motionColor))
					}).Menu(func(m *ui.Menu) {
						for _, choice := range choices {
							if m.Item(choice.Label).Radio(rule.FolderID == choice.ID).Chosen() {
								rules[i].FolderID = choice.ID
								a.Preferences.AutoImport[libraryID] = rules
								a.SavePreferences()
							}
						}
					})
					if settingsButton(c.Key("remove"), a.T("remove")).Height(controlSmall).Clicked() {
						a.Preferences.AutoImport[libraryID] = slices.Delete(slices.Clone(rules), i, i+1)
						a.SavePreferences()
					}
				})
				if message := a.Settings.AutoImportErrors[rule.Path]; message != "" {
					if a.Settings.AutoImportMissing[rule.Path] {
						ui.Row(c.Key("settings-error")).FillWidth().Padding(space4, space16, space12, space16).Gap(space12).AlignItems(ui.Center).Children(func() {
							ui.Text(c, a.T("auto_import_missing")).FontSize(fontSecondary).TextColor(p.Danger).Grow(1).MinWidth(0).Role(ui.RoleAlert)
							if settingsButton(c.Key("choose-again"), a.T("auto_import_choose_again")).Height(controlSmall).Clicked() {
								a.Settings.AutoImportAddError = ""
								a.chooseAutoImportFolder(rule.Path)
							}
						})
					} else {
						settingsError(c, message)
					}
				}
			})
		}
		settingsSeparator(c)
		settingsRow(c, func() {
			if settingsButton(c, a.T("add_watch_folder")).Height(controlSmall).Clicked() {
				a.Settings.AutoImportAddError = ""
				a.chooseAutoImportFolder("")
			}
		})
		if a.Settings.AutoImportAddError != "" {
			settingsError(c, a.Settings.AutoImportAddError)
		}
	})
}

// 选好文件夹后先数一下里面现有的图片，弹确认；确认前不加规则。replace 不为空时是给这条规则重新选文件夹。
func (a *App) chooseAutoImportFolder(replace string) {
	if a.Library == nil || a.Window == nil {
		return
	}
	libraryID := a.Library.Manifest.LibraryID
	go func() {
		count := 0
		paths, err := mygo.Dialog.Open(mygo.OpenDialogOptions{Parent: a.SettingsWindow, Title: a.T("add_watch_folder"), Directory: true})
		if err == nil && len(paths) > 0 {
			var info os.FileInfo
			info, err = os.Stat(paths[0])
			if err == nil && !info.IsDir() {
				err = &os.PathError{Op: "open", Path: paths[0], Err: os.ErrInvalid}
			}
			if err == nil {
				count, err = countAutoImportFiles(paths[0])
			}
		}
		a.Window.Update(func() {
			if err != nil {
				a.Settings.AutoImportAddError = a.Texts.Error(err)
				a.ShowError(err)
				a.InvalidateWindows()
				return
			}
			if len(paths) == 0 || a.Library == nil || a.Library.Manifest.LibraryID != libraryID {
				return
			}
			path := filepath.Clean(paths[0])
			for _, rule := range a.Preferences.AutoImport[libraryID] {
				if rule.Path == path {
					return
				}
			}
			a.Settings.AutoImportConfirm = &autoImportConfirm{Path: path, Replace: replace, Count: count, LibraryID: libraryID}
			a.InvalidateWindows()
		})
	}()
}

type autoImportConfirm struct {
	Path, Replace, LibraryID string
	Count                    int
}

// 数文件夹里现在有多少个能导入的文件，规则和自动导入扫描一致：递归、跳过隐藏项和 .momento 图库包、不跟随链接。
func countAutoImportFiles(root string) (int, error) {
	count := 0
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if path == root {
			return nil
		}
		if strings.HasPrefix(entry.Name(), ".") || entry.IsDir() && strings.EqualFold(filepath.Ext(path), ".momento") {
			if entry.IsDir() {
				return fs.SkipDir
			}
			return nil
		}
		if entry.Type().IsRegular() && assets.Supported(path) {
			count++
		}
		return nil
	})
	return count, err
}

func (a *App) autoImportConfirmDialog(c *ui.Context) {
	state := a.Settings.AutoImportConfirm
	if state == nil {
		return
	}
	open, confirmed := true, false
	suffix := ""
	if settingsPlatform() == "windows" {
		suffix = "_windows"
	}
	name := filepath.Base(state.Path)
	text, action := a.T("auto_import_confirm"+suffix, name, state.Count), a.T("auto_import_confirm_add")
	if state.Count == 0 {
		text, action = a.T("auto_import_confirm_empty"+suffix, name), a.T("auto_import_confirm_add_empty")
	}
	dialogPanel(c.Key("auto-import-confirm"), &open, panelDialog, "folder", a.T("auto_import_confirm_title"), func() {
		ui.Text(c, text).FillWidth().TextColor(paletteFor(c.Theme()).TextSecondary)
	}, func() {
		// 添加后会把原文件移到废纸篓，默认焦点放在取消上，和其他会移走文件的确认一样，回车不会直接批准。
		if dialogButton(c, a.T("cancel"), false).AutoFocus().Clicked() {
			open = false
		}
		if dialogPrimaryButton(c, action, false).Clicked() {
			confirmed, open = true, false
		}
	})
	if confirmed {
		a.commitAutoImportFolder(state)
	}
	if !open {
		a.Settings.AutoImportConfirm = nil
	}
}

func (a *App) commitAutoImportFolder(state *autoImportConfirm) {
	if a.Library == nil || a.Library.Manifest.LibraryID != state.LibraryID {
		return
	}
	rules := a.Preferences.AutoImport[state.LibraryID]
	if i := slices.IndexFunc(rules, func(rule AutoImportRule) bool { return state.Replace != "" && rule.Path == state.Replace }); i >= 0 {
		// 只换路径，目标文件夹和开关保持原样。
		rules = slices.Clone(rules)
		rules[i].Path = state.Path
		delete(a.Settings.AutoImportErrors, state.Replace)
		delete(a.Settings.AutoImportMissing, state.Replace)
		a.Preferences.AutoImport[state.LibraryID] = rules
		slog.Info("重新选择自动导入文件夹", "from", state.Replace, "path", state.Path, "library", state.LibraryID)
		a.SavePreferences()
		return
	}
	if a.Preferences.AutoImport == nil {
		a.Preferences.AutoImport = make(map[string][]AutoImportRule)
	}
	a.Preferences.AutoImport[state.LibraryID] = append(rules, AutoImportRule{Path: state.Path, Enabled: true, Library: a.Library.Root})
	slog.Info("添加自动导入文件夹", "path", state.Path, "library", state.LibraryID, "existing", state.Count)
	a.SavePreferences()
}

func (a *App) notificationsSettings(c *ui.Context) {
	settingsSection(c, "", "", func() {
		a.switchSettingNote(c, "notifications_enabled", a.T("notifications_foreground_hint"), &a.Preferences.Notify, false)
		if a.Settings.NotificationDenied {
			settingsError(c, a.T("notifications_denied"))
		}
		settingsSeparator(c)
		ui.Column(c).FillWidth().Padding(space12, space16).Gap(space12).Children(func() {
			for _, event := range []struct{ key, label string }{{"extension", "notify_extension"}, {"auto-import", "notify_auto_import"}, {"duplicate", "notify_duplicate"}} {
				checked := slices.Contains(a.Preferences.NotifyEvents, event.key)
				if settingsCheckbox(c.Key(event.key), &checked, a.T(event.label)).Disabled(!a.Preferences.Notify).Changed() {
					a.Preferences.NotifyEvents = slices.DeleteFunc(a.Preferences.NotifyEvents, func(key string) bool { return key == event.key })
					if checked {
						a.Preferences.NotifyEvents = append(a.Preferences.NotifyEvents, event.key)
					}
					a.SavePreferences()
				}
			}
		})
		settingsSeparator(c)
		a.switchSetting(c, "notify_sound", &a.Preferences.NotifySound, !a.Preferences.Notify)
	})
}

func (a *App) passwordSettings(c *ui.Context) {
	p := paletteFor(c.Theme())
	settingsSection(c, "", a.T("password_lock_hint"), func() {
		settingsRow(c, func() {
			ui.Text(c, a.T("password_lock_enabled")).Grow(1)
			// 操作结果显示几秒就收起，页面重新进入时也清掉。
			if a.Settings.PasswordNotice != "" {
				if remaining := passwordNoticeDuration - time.Since(a.Settings.PasswordNoticeAt); remaining > 0 {
					fadeMotion(ui.Text(c.Key("password-notice"), a.Settings.PasswordNotice).FontSize(fontSecondary).TextColor(p.Success)).Role(ui.RoleStatus)
					c.After(remaining)
				} else {
					a.Settings.PasswordNotice = ""
				}
			}
			enabled := a.Preferences.PasswordHash != ""
			if ui.Switch(c, &enabled).Label(a.T("password_lock_enabled")).Changed() {
				mode := "disable"
				if enabled {
					mode = "enable"
				}
				a.ShowPasswordSettings(mode)
			}
		})
		// 打开密码锁后「修改密码」一行展开出来，关掉时收起。
		if a.Preferences.PasswordHash != "" {
			revealMotion(ui.Column(c.Key("change-password"))).FillWidth().Children(func() {
				settingsSeparator(c)
				settingsRow(c, func() {
					if settingsButton(c, a.T("change_password")).Height(controlSmall).Clicked() {
						a.ShowPasswordSettings("change")
					}
				})
				settingsSeparator(c)
				settingsRow(c, func() {
					if settingsButton(c.Key("lock-now"), a.T("lock_now")).Height(controlSmall).Clicked() {
						a.LockNow()
					}
					if shortcut := displayShortcut(a.Shortcut("lock")); shortcut != "" {
						ui.Text(c, shortcut).FontSize(fontSecondary).TextColor(p.TextTertiary).SingleLine()
					}
				})
			})
		}
	})
}

const passwordNoticeDuration = 4 * time.Second

func shortcutSearchMatches(query, name, accelerator string) bool {
	query = strings.ToLower(strings.TrimSpace(query))
	return query == "" || strings.Contains(strings.ToLower(name), query) || strings.Contains(strings.ToLower(accelerator), query)
}
