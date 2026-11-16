package library

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"
	"strings"
	"time"

	"github.com/Seaony/Momento/app/internal/i18n"
	"github.com/Seaony/Momento/app/internal/model"
	"github.com/google/uuid"
)

// 批注框太小在图上点不中，也多半是误拖；比例上至少占原图 0.5%。
const minAnnotationSize = .005

func loadAnnotations(ctx context.Context, tx *sql.Tx, assets []model.Asset, byID map[string]int32) error {
	rows, err := tx.QueryContext(ctx, "SELECT id,asset_id,x,y,w,h,text,created_at FROM asset_annotations ORDER BY asset_id,created_at,id")
	if err != nil {
		return err
	}
	for rows.Next() {
		var note model.Annotation
		var assetID, created string
		if err = rows.Scan(&note.ID, &assetID, &note.X, &note.Y, &note.W, &note.H, &note.Text, &created); err != nil {
			return errors.Join(err, rows.Close())
		}
		if note.CreatedAt, err = time.Parse(time.RFC3339Nano, created); err != nil {
			return errors.Join(err, rows.Close())
		}
		if i, ok := byID[assetID]; ok {
			assets[i].Annotations = append(assets[i].Annotations, note)
		}
	}
	return errors.Join(rows.Err(), rows.Close())
}

// validAnnotationRect 检查框在图里、不是一条线；越界一点点（拖到图外）由界面先收进来，这里只拦真正不对的。
func validAnnotationRect(note model.Annotation) bool {
	return note.X >= 0 && note.Y >= 0 && note.W >= minAnnotationSize && note.H >= minAnnotationSize &&
		note.X+note.W <= 1+1e-6 && note.Y+note.H <= 1+1e-6
}

// AddAnnotation 给素材加一块批注，返回带 ID 和时间的批注；素材的修改时间一起更新。
func (l *Library) AddAnnotation(ctx context.Context, assetID string, note model.Annotation) (model.Annotation, error) {
	if !validAnnotationRect(note) {
		return model.Annotation{}, i18n.NewError("error_annotation_rect")
	}
	note.ID = uuid.NewString()
	note.Text = strings.TrimSpace(note.Text)
	note.CreatedAt = time.Now().UTC()
	now := note.CreatedAt.Format(time.RFC3339Nano)
	err := l.transaction(ctx, func(tx *sql.Tx) error {
		if err := changed(tx.ExecContext(ctx, "UPDATE assets SET updated_at=? WHERE id=?", now, assetID)); err != nil {
			return err
		}
		_, err := tx.ExecContext(ctx, "INSERT INTO asset_annotations(id,asset_id,x,y,w,h,text,created_at) VALUES(?,?,?,?,?,?,?,?)",
			note.ID, assetID, note.X, note.Y, note.W, note.H, note.Text, now)
		return err
	})
	if err != nil {
		return model.Annotation{}, err
	}
	slog.Info("添加区域批注", "asset", assetID, "annotation", note.ID)
	return note, nil
}

// UpdateAnnotation 改批注文字。
func (l *Library) UpdateAnnotation(ctx context.Context, id, text string) error {
	return l.transaction(ctx, func(tx *sql.Tx) error {
		if err := touchAnnotationAsset(ctx, tx, id); err != nil {
			return err
		}
		return changed(tx.ExecContext(ctx, "UPDATE asset_annotations SET text=? WHERE id=?", strings.TrimSpace(text), id))
	})
}

// DeleteAnnotation 删掉一块批注。
func (l *Library) DeleteAnnotation(ctx context.Context, id string) error {
	err := l.transaction(ctx, func(tx *sql.Tx) error {
		if err := touchAnnotationAsset(ctx, tx, id); err != nil {
			return err
		}
		return changed(tx.ExecContext(ctx, "DELETE FROM asset_annotations WHERE id=?", id))
	})
	if err == nil {
		slog.Info("删除区域批注", "annotation", id)
	}
	return err
}

// touchAnnotationAsset 更新批注所在素材的修改时间；批注不存在时报 sql.ErrNoRows。
func touchAnnotationAsset(ctx context.Context, tx *sql.Tx, id string) error {
	return changed(tx.ExecContext(ctx, "UPDATE assets SET updated_at=? WHERE id=(SELECT asset_id FROM asset_annotations WHERE id=?)", time.Now().UTC().Format(time.RFC3339Nano), id))
}
