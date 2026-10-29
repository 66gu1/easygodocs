//go:build testutil

package gorm

import (
	"slices"
	"testing"

	"github.com/66gu1/easygodocs/internal/app/entity"
	"github.com/66gu1/easygodocs/internal/infrastructure/contextx"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

// The permission check walks only the ancestors of the checked entity.
// It must give the same result as loading the whole permitted subtrees.
func TestIsPermitted_MatchesSubtreeLoading(t *testing.T) {
	t.Parallel()
	repo, gdb, _ := newEntityRepo(t)
	core := newEntityCore(t, repo, gdb)
	author := createUserForEntity(t, gdb)
	other := createUserForEntity(t, gdb)

	// root -> a -> a1 -> a11
	//      |    -> hidden (never published, created by other)
	//      -> b -> b1
	create := func(typ entity.Type, parent *uuid.UUID, userID uuid.UUID, draft bool) uuid.UUID {
		id, err := core.Create(t.Context(), entity.CreateEntityReq{
			Type: typ, Name: "e", ParentID: parent, UserID: userID, IsDraft: draft,
		})
		require.NoError(t, err)
		return id
	}
	root := create(entity.TypeDepartment, nil, author, false)
	a := create(entity.TypeDepartment, &root, author, false)
	a1 := create(entity.TypeDepartment, &a, author, false)
	a11 := create(entity.TypeArticle, &a1, author, false)
	hidden := create(entity.TypeArticle, &a, other, true)
	b := create(entity.TypeDepartment, &root, author, false)
	b1 := create(entity.TypeArticle, &b, author, false)
	all := []uuid.UUID{root, a, a1, a11, hidden, b, b1}

	directSets := [][]uuid.UUID{{root}, {a}, {a1}, {a11}, {b1}, {a1, b}, {hidden}}
	for _, userID := range []uuid.UUID{author, other} {
		ctx := contextx.SetUserID(t.Context(), userID)
		for _, direct := range directSets {
			for _, withAncestors := range []bool{false, true} {
				hType := entity.HierarchyTypeChildrenOnly
				if withAncestors {
					hType = entity.HierarchyTypeChildrenAndParents
				}
				subtrees, err := repo.GetHierarchy(ctx, direct, maxDepth, &userID, hType)
				require.NoError(t, err)

				for _, id := range all {
					want := slices.ContainsFunc(subtrees, func(item entity.ListItem) bool { return item.ID == id })
					got, err := core.IsPermitted(ctx, id, direct, withAncestors)
					require.NoError(t, err)
					require.Equal(t, want, got, "user %s, direct %v, withAncestors %v, entity %s", userID, direct, withAncestors, id)
				}
			}
		}
	}

	// sanity check of the expected semantics
	ctx := contextx.SetUserID(t.Context(), author)
	for _, tc := range []struct {
		id            uuid.UUID
		direct        []uuid.UUID
		withAncestors bool
		want          bool
	}{
		{id: a11, direct: []uuid.UUID{a}, want: true},                       // inherited by the subtree
		{id: root, direct: []uuid.UUID{a}, want: false},                     // write doesn't open ancestors
		{id: root, direct: []uuid.UUID{a}, withAncestors: true, want: true}, // read does
		{id: b, direct: []uuid.UUID{a}, withAncestors: true, want: false},   // but not siblings
		{id: hidden, direct: []uuid.UUID{root}, want: false},                // someone else's unpublished entity
	} {
		got, err := core.IsPermitted(ctx, tc.id, tc.direct, tc.withAncestors)
		require.NoError(t, err)
		require.Equal(t, tc.want, got)
	}
}
