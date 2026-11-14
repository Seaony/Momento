package ui

import (
	"log/slog"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
)

var settingsPlatform = func() string { return runtime.GOOS }

func settingsError(c *ui.Context, text string) {
	p := paletteFor(c.Theme())
	fadeMotion(ui.Text(c, text).FontSize(fontSecondary).TextColor(p.Danger).Padding(space4, space16, space12, space16)).Role(ui.RoleAlert)
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

func settingsSegments(c *ui.Context, label string, selected *int, labels ...string) ui.Element {
	theme := c.Theme()
	p := paletteFor(theme)
	parts := ui.SegmentedBase(c, selected, len(labels))
	items := make([]ui.Element, len(labels))
	parts.Track.AlignItems(ui.Stretch).Padding(space4).Gap(space4).Radius(radiusControl).Background(p.Subtle).Border(1, p.BorderSubtle).Label(label).Children(func() {
		for i, text := range labels {
			segment := parts.Segment(i).MinHeight(controlSmall).Padding(space4, space12).Radius(radiusSmall).Cursor(ui.CursorPointer).Label(text)
			styleSegment(c, segment, i == *selected)
			items[i] = segment
			segment.Children(func() { ui.Text(c, text).SingleLine() })
		}
	})
	slidingSegmentTrack(c, parts.Track, items, *selected, space4, space4)
	return parts.Track.FillEqually()
}

func (a *App) stringSetting(c *ui.Context, key string, value *string, codes, labels []string) {
	ui.Column(c.Key(key)).FillWidth().Padding(space12, space16).Gap(space8).Children(func() {
		ui.Text(c, a.T(key))
		selected := max(0, slices.Index(codes, *value))
		translated := make([]string, len(labels))
		for i, label := range labels {
			translated[i] = a.T(label)
		}
		if settingsSegments(c, a.T(key), &selected, translated...).Changed() {
			*value = codes[selected]
			a.SavePreferences()
		}
	})
}

func (a *App) boolSetting(c *ui.Context, key string, value *bool, firstValue bool, firstKey, secondKey string) {
	ui.Column(c.Key(key)).FillWidth().Padding(space12, space16).Gap(space8).Children(func() {
		ui.Text(c, a.T(key))
		selected := 0
		if *value != firstValue {
			selected = 1
		}
		if settingsSegments(c, a.T(key), &selected, a.T(firstKey), a.T(secondKey)).Changed() {
			*value = firstValue
			if selected == 1 {
				*value = !firstValue
			}
			a.SavePreferences()
		}
	})
}

func (a *App) appearanceSettings(c *ui.Context) {
	settingsSection(c, a.T("theme_title"), "", func() {
		ui.Column(c).FillWidth().Padding(space16).Children(func() { a.ThemeSwatches(c) })
		settingsSeparator(c)
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
		settingsSeparator(c)
		a.switchSetting(c, "sidebar_counts", &a.Preferences.SidebarCounts, false)
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
	})
	settingsSection(c, a.T("sidebar_show"), "", func() {
		items := []struct {
			key, label string
			fixed      bool
		}{{"all", "all", true}, {"tags", "tag_management", true}, {"trash", "trash", true}, {"favorites", "favorites", false}, {"unfiled", "uncategorized", false}, {"untagged", "untagged", false}, {"recent", "recent", false}, {"random", "random_browse", false}}
		ui.Column(c).FillWidth().Padding(space16).Gap(space12).Children(func() {
			for start := 0; start < len(items); start += 3 {
				ui.Row(c).FillWidth().Gap(space8).Children(func() {
					for i := start; i < min(start+3, len(items)); i++ {
						item := items[i]
						shown := !slices.Contains(a.Preferences.SidebarHidden, item.key)
						if settingsCheckbox(c.Key(item.key), &shown, a.T(item.label)).Grow(1).MinWidth(0).Disabled(item.fixed).Changed() {
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
}

func (a *App) importSettings(c *ui.Context) {
	p := paletteFor(c.Theme())
	settingsSection(c, "", "", func() {
		a.switchSetting(c, "select_imported", &a.Preferences.SelectImported, false)
		settingsSeparator(c)
		a.switchSetting(c, "import_keywords", &a.Preferences.ImportKeywords, false)
		settingsSeparator(c)
		a.switchSetting(c, "extension_picker", &a.Preferences.ExtensionPicker, false)
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
			settingsRow(c, func() { ui.Text(c, a.T("auto_import_empty")).TextColor(p.TextSecondary) })
		}
		for i, rule := range rules {
			if i > 0 {
				settingsSeparator(c)
			}
			ui.Column(c.Key(rule.Path)).FillWidth().Children(func() {
				settingsRow(c, func() {
					appIcon(c, "folder").Size(iconSize, iconSize).Shrink(0)
					ui.Text(c, rule.Path).SingleLine().Grow(1).MinWidth(0).Tooltip(rule.Path)
				})
				settingsRow(c, func() {
					ui.Text(c, a.T("auto_import_target")).Grow(1).MinWidth(0)
					folderName := a.T("auto_import_library_root")
					for _, f := range a.Snapshot.Folders {
						if f.ID == rule.FolderID {
							folderName = f.Name
						}
					}
					target := settingsMenuButton(c, a.T("auto_import_target")).Width(160)
					target.Children(func() {
						ui.Text(c, folderName).SingleLine().Grow(1).MinWidth(0)
						symbol := appSymbol(c, "chevrons-up-down").Size(iconSmall, iconSmall).Shrink(0)
						symbol.TextColor(symbol.AnimateColor("symbol-color", symbolColor(c.Theme(), target.Hovered()), motionColor))
					}).Menu(func(m *ui.Menu) {
						if m.Item(a.T("auto_import_library_root")).Radio(rule.FolderID == "").Chosen() {
							rules[i].FolderID = ""
							a.Preferences.AutoImport[libraryID] = rules
							a.SavePreferences()
						}
						for _, folder := range a.Snapshot.Folders {
							if m.Item(folder.Name).Radio(rule.FolderID == folder.ID).Chosen() {
								rules[i].FolderID = folder.ID
								a.Preferences.AutoImport[libraryID] = rules
								a.SavePreferences()
							}
						}
					})
					enabled := rule.Enabled
					if ui.Switch(c.Key("enabled"), &enabled).Label(a.T("auto_import_enabled", rule.Path)).Changed() {
						rules[i].Enabled = enabled
						a.Preferences.AutoImport[libraryID] = rules
						a.SavePreferences()
					}
					if settingsButton(c.Key("remove"), a.T("remove")).Height(controlSmall).Clicked() {
						a.Preferences.AutoImport[libraryID] = slices.Delete(slices.Clone(rules), i, i+1)
						a.SavePreferences()
					}
				})
				if message := a.Settings.AutoImportErrors[rule.Path]; message != "" {
					settingsError(c, message)
				}
			})
		}
		settingsSeparator(c)
		settingsRow(c, func() {
			if settingsButton(c, a.T("add_watch_folder")).Height(controlSmall).Clicked() {
				a.Settings.AutoImportAddError = ""
				a.chooseAutoImportFolder()
			}
		})
		if a.Settings.AutoImportAddError != "" {
			settingsError(c, a.Settings.AutoImportAddError)
		}
	})
}

func (a *App) chooseAutoImportFolder() {
	if a.Library == nil || a.Window == nil {
		return
	}
	libraryID := a.Library.Manifest.LibraryID
	go func() {
		paths, err := mygo.Dialog.Open(mygo.OpenDialogOptions{Parent: a.SettingsWindow, Title: a.T("add_watch_folder"), Directory: true})
		if err == nil && len(paths) > 0 {
			var info os.FileInfo
			info, err = os.Stat(paths[0])
			if err == nil && !info.IsDir() {
				err = &os.PathError{Op: "open", Path: paths[0], Err: os.ErrInvalid}
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
			rules := a.Preferences.AutoImport[libraryID]
			for _, rule := range rules {
				if rule.Path == path {
					return
				}
			}
			if a.Preferences.AutoImport == nil {
				a.Preferences.AutoImport = make(map[string][]AutoImportRule)
			}
			a.Preferences.AutoImport[libraryID] = append(rules, AutoImportRule{Path: path, Enabled: true})
			slog.Info("添加自动导入文件夹", "path", path, "library", libraryID)
			a.SavePreferences()
		})
	}()
}

func (a *App) notificationsSettings(c *ui.Context) {
	settingsSection(c, "", "", func() {
		a.switchSetting(c, "notifications_enabled", &a.Preferences.Notify, false)
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
		if a.Settings.NotificationDenied {
			settingsError(c, a.T("notifications_denied"))
		}
	})
}

func (a *App) passwordSettings(c *ui.Context) {
	settingsSection(c, "", a.T("password_lock_hint"), func() {
		settingsRow(c, func() {
			ui.Text(c, a.T("password_lock_enabled")).Grow(1)
			enabled := a.Preferences.PasswordHash != ""
			if ui.Switch(c, &enabled).Label(a.T("password_lock_enabled")).Changed() {
				mode := "disable"
				if enabled {
					mode = "enable"
				}
				a.ShowPasswordSettings(mode)
			}
		})
		if a.Preferences.PasswordHash != "" {
			settingsSeparator(c)
			settingsRow(c, func() {
				if settingsButton(c, a.T("change_password")).Height(controlSmall).Clicked() {
					a.ShowPasswordSettings("change")
				}
			})
		}
	})
}

func shortcutSearchMatches(query, name, accelerator string) bool {
	query = strings.ToLower(strings.TrimSpace(query))
	return query == "" || strings.Contains(strings.ToLower(name), query) || strings.Contains(strings.ToLower(accelerator), query)
}
