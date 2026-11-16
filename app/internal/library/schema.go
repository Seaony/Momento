package library

const schemaV1 = `
PRAGMA user_version = 1;
CREATE TABLE assets (
 id TEXT PRIMARY KEY NOT NULL,
 name TEXT NOT NULL DEFAULT '',
 original_file_name TEXT NOT NULL DEFAULT '',
 extension TEXT NOT NULL DEFAULT '',
 media_type TEXT NOT NULL DEFAULT '',
 size INTEGER NOT NULL DEFAULT 0,
 width INTEGER NOT NULL DEFAULT 0,
 height INTEGER NOT NULL DEFAULT 0,
 orientation INTEGER NOT NULL DEFAULT 1,
 color_profile TEXT NOT NULL DEFAULT '',
 exif_json TEXT NOT NULL DEFAULT '{}',
 note TEXT NOT NULL DEFAULT '',
 source_url TEXT NOT NULL DEFAULT '',
 relative_path TEXT NOT NULL CHECK (
  relative_path <> '' AND substr(relative_path, 1, 1) NOT IN ('/', '\')
  AND instr(relative_path, '\') = 0 AND instr(relative_path, ':') = 0
  AND relative_path <> '..' AND relative_path NOT LIKE '../%'
  AND relative_path NOT LIKE '%/../%' AND relative_path NOT LIKE '%/..'
 ),
 favorite INTEGER NOT NULL DEFAULT 0 CHECK (favorite IN (0, 1)),
 trashed INTEGER NOT NULL DEFAULT 0 CHECK (trashed IN (0, 1)),
 created_at TEXT NOT NULL,
 updated_at TEXT NOT NULL
);
CREATE INDEX assets_added ON assets(trashed, created_at DESC);
CREATE INDEX assets_name ON assets(name COLLATE NOCASE);
CREATE INDEX assets_extension ON assets(extension);
CREATE TABLE asset_colors (
 asset_id TEXT NOT NULL REFERENCES assets(id) ON DELETE CASCADE,
 hex TEXT NOT NULL,
 proportion REAL NOT NULL,
 family TEXT NOT NULL DEFAULT '',
 sort_index INTEGER NOT NULL,
 PRIMARY KEY (asset_id, sort_index)
);
CREATE INDEX asset_colors_family ON asset_colors(family, asset_id);
CREATE TABLE folders (
 id TEXT PRIMARY KEY NOT NULL,
 name TEXT NOT NULL,
 parent_id TEXT REFERENCES folders(id) ON DELETE CASCADE,
 sort_index INTEGER NOT NULL DEFAULT 0
);
CREATE INDEX folders_parent ON folders(parent_id, sort_index);
CREATE TABLE asset_folders (
 asset_id TEXT NOT NULL REFERENCES assets(id) ON DELETE CASCADE,
 folder_id TEXT NOT NULL REFERENCES folders(id) ON DELETE CASCADE,
 PRIMARY KEY (asset_id, folder_id)
);
CREATE INDEX asset_folders_folder ON asset_folders(folder_id, asset_id);
CREATE TABLE tags (
 id TEXT PRIMARY KEY NOT NULL,
 name TEXT NOT NULL,
 normalized_name TEXT NOT NULL UNIQUE,
 color_hex TEXT NOT NULL DEFAULT ''
);
CREATE TABLE asset_tags (
 asset_id TEXT NOT NULL REFERENCES assets(id) ON DELETE CASCADE,
 tag_id TEXT NOT NULL REFERENCES tags(id) ON DELETE CASCADE,
 PRIMARY KEY (asset_id, tag_id)
);
CREATE INDEX asset_tags_tag ON asset_tags(tag_id, asset_id);
`

const migrationV2 = `
ALTER TABLE assets ADD COLUMN last_used_at TEXT NOT NULL DEFAULT '';
ALTER TABLE assets ADD COLUMN rotation INTEGER NOT NULL DEFAULT 0 CHECK (rotation IN (0,90,180,270));
CREATE INDEX assets_used ON assets(trashed,last_used_at DESC);
CREATE TABLE smart_folders (
 id TEXT PRIMARY KEY NOT NULL,
 name TEXT NOT NULL,
 match_mode TEXT NOT NULL CHECK (match_mode IN ('all','any')),
 rules_json TEXT NOT NULL,
 sort_index INTEGER NOT NULL DEFAULT 0
);
CREATE TABLE quick_access (
 id TEXT PRIMARY KEY NOT NULL,
 kind TEXT NOT NULL CHECK (kind IN ('folder','tag','smart')),
 target_id TEXT NOT NULL,
 sort_index INTEGER NOT NULL DEFAULT 0,
 UNIQUE(kind,target_id)
);
CREATE TRIGGER quick_access_folder_deleted AFTER DELETE ON folders BEGIN
 DELETE FROM quick_access WHERE kind='folder' AND target_id=OLD.id;
END;
CREATE TRIGGER quick_access_tag_deleted AFTER DELETE ON tags BEGIN
 DELETE FROM quick_access WHERE kind='tag' AND target_id=OLD.id;
END;
CREATE TRIGGER quick_access_smart_deleted AFTER DELETE ON smart_folders BEGIN
 DELETE FROM quick_access WHERE kind='smart' AND target_id=OLD.id;
END;
PRAGMA user_version = 2;
`

// v3：文件夹可以选一个图标（界面图标库里的名字，空是默认文件夹图标）。
const migrationV3 = `
ALTER TABLE folders ADD COLUMN icon TEXT NOT NULL DEFAULT '';
PRAGMA user_version = 3;
`

// v4：视频和实况照片。kind 是 image、video 或 live；duration_ms 是视频（实况照片那段视频）的时长；
// motion_path 是实况照片配对视频在包内的相对路径，和 relative_path 一样只能是包内的相对路径。
const migrationV4 = `
ALTER TABLE assets ADD COLUMN kind TEXT NOT NULL DEFAULT 'image' CHECK (kind IN ('image', 'video', 'live'));
ALTER TABLE assets ADD COLUMN duration_ms INTEGER NOT NULL DEFAULT 0 CHECK (duration_ms >= 0);
ALTER TABLE assets ADD COLUMN motion_path TEXT NOT NULL DEFAULT '' CHECK (
 motion_path = '' OR (
  substr(motion_path, 1, 1) NOT IN ('/', '\') AND instr(motion_path, '\') = 0 AND instr(motion_path, ':') = 0
  AND motion_path <> '..' AND motion_path NOT LIKE '../%' AND motion_path NOT LIKE '%/../%' AND motion_path NOT LIKE '%/..'
 )
);
CREATE INDEX assets_kind ON assets(kind);
PRAGMA user_version = 4;
`

// v5：星级评分，0 是没评分，1～5 星。
const migrationV5 = `
ALTER TABLE assets ADD COLUMN rating INTEGER NOT NULL DEFAULT 0 CHECK (rating BETWEEN 0 AND 5);
PRAGMA user_version = 5;
`

const schema = schemaV1 + migrationV2 + migrationV3 + migrationV4 + migrationV5
