//go:build testutil

package gorm

import (
	"testing"
	"time"

	"github.com/66gu1/easygodocs/internal/app/entity"
	"github.com/66gu1/easygodocs/internal/infrastructure/db"
	"github.com/google/uuid"
	"github.com/pressly/goose/v3"
	"github.com/stretchr/testify/require"
)

// versionBeforeDrafts is the last migration before entity_drafts was added.
const versionBeforeDrafts = 20250819082427

// Before entity_drafts a published entity could be turned back into a draft by overwriting its row
// and setting current_version to NULL. The migration must move such drafts to entity_drafts
// and restore the published version.
func TestMigration_AddEntityDrafts_MovesOldDrafts(t *testing.T) {
	t.Parallel()
	repo, gdb, _ := newEntityRepo(t)
	sqlDB, err := gdb.DB()
	require.NoError(t, err)

	author := createUserForEntity(t, gdb)
	editor := createUserForEntity(t, gdb)
	now := time.Now().UTC().Truncate(time.Second)

	unpublished := uuid.New()
	require.NoError(t, repo.CreateDraft(t.Context(), entity.CreateEntityReq{
		Type: entity.TypeDepartment, Name: "new", Content: "new content", UserID: author,
	}, unpublished))
	republished := uuid.New()
	require.NoError(t, repo.Create(t.Context(), entity.CreateEntityReq{
		Type: entity.TypeDepartment, Name: "v1", Content: "published content", UserID: author,
	}, republished, now))

	require.NoError(t, goose.DownTo(sqlDB, db.MigrationsDir(), versionBeforeDrafts))
	// what the old UpdateDraft did
	require.NoError(t, gdb.Exec(
		`UPDATE entities SET name = 'draft', content = 'draft content', updated_by = ?, updated_at = ?, current_version = NULL WHERE id = ?`,
		editor, now.Add(time.Hour), republished,
	).Error)
	require.NoError(t, goose.Up(sqlDB, db.MigrationsDir()))

	// the published version is visible again, the draft belongs to its editor
	got, err := repo.Get(t.Context(), republished)
	require.NoError(t, err)
	require.Equal(t, "v1", got.Name)
	require.Equal(t, "published content", got.Content)
	require.Equal(t, &[]int{1}[0], got.CurrentVersion)
	require.Equal(t, author, got.UpdatedBy)

	draft, err := repo.GetDraft(t.Context(), republished, editor)
	require.NoError(t, err)
	require.Equal(t, "draft", draft.Name)
	require.Equal(t, "draft content", draft.Content)
	require.Equal(t, &[]int{1}[0], draft.BaseVersion)

	// a never published entity stays as it was
	got, err = repo.Get(t.Context(), unpublished)
	require.NoError(t, err)
	require.Equal(t, "new content", got.Content)
	require.Nil(t, got.CurrentVersion)
	_, err = repo.GetDraft(t.Context(), unpublished, author)
	require.ErrorIs(t, err, entity.ErrDraftNotFound())
}
