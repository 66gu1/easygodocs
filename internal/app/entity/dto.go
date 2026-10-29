package entity

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/66gu1/easygodocs/internal/infrastructure/apperr"
	"github.com/66gu1/easygodocs/internal/infrastructure/logger"
	"github.com/google/uuid"
)

const (
	FieldVersion apperr.Field = "version"
	FieldNode    apperr.Field = "node"
)

type Type string

const (
	TypeArticle    Type = "article"
	TypeDepartment Type = "department"
)

func (t Type) CheckIsValid() error {
	switch t {
	case TypeArticle, TypeDepartment:
		return nil
	default:
		return ErrInvalidType()
	}
}

func (t Type) ValidateParentTypeCompatibility(parentType Type) error {
	if t == TypeDepartment && parentType == TypeArticle {
		return ErrIncompatibleParentType()
	}

	return nil
}

type Entity struct {
	ID             uuid.UUID  `json:"id"`
	Type           Type       `json:"type"`
	Name           string     `json:"name"`
	Content        string     `json:"content"`
	ParentID       *uuid.UUID `json:"parent_id,omitempty"`
	CreatedBy      uuid.UUID  `json:"created_by"`
	UpdatedBy      uuid.UUID  `json:"updated_by"`
	CurrentVersion *int       `json:"current_version,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
}

type ListItem struct {
	ID       uuid.UUID  `json:"id"`
	Type     Type       `json:"type"`
	Name     string     `json:"name"`
	ParentID *uuid.UUID `json:"parent_id,omitempty"`
	// CurrentVersion is nil for an entity that has never been published.
	CurrentVersion *int `json:"current_version,omitempty"`
	Depth          int  `json:"-"`
}

func (i ListItem) IsPublished() bool {
	return i.CurrentVersion != nil
}

// Draft holds a user's unpublished changes to an entity.
// Readers keep seeing the published version until the draft is published.
type Draft struct {
	EntityID uuid.UUID `json:"entity_id"`
	UserID   uuid.UUID `json:"user_id"`
	Name     string    `json:"name"`
	Content  string    `json:"content"`
	// BaseVersion is the published version the draft was started from.
	BaseVersion *int      `json:"base_version,omitempty"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type SaveDraftReq struct {
	EntityID uuid.UUID `json:"entity_id"`
	UserID   uuid.UUID `json:"user_id"`
	Name     string    `json:"name"`
	Content  string    `json:"content"`
}

type CreateEntityReq struct {
	Type     Type       `json:"type"`
	Name     string     `json:"name"`
	Content  string     `json:"content"`
	ParentID *uuid.UUID `json:"parent_id,omitempty"`
	IsDraft  bool       `json:"is_draft"`
	UserID   uuid.UUID  `json:"user_id"`
}

type UpdateEntityReq struct {
	ID            uuid.UUID  `json:"id"`
	Name          string     `json:"name"`
	Content       string     `json:"content"`
	ParentID      *uuid.UUID `json:"parent_id,omitempty"`
	UserID        uuid.UUID  `json:"user_id"`
	ParentChanged bool       `json:"parent_changed"`
	EntityType    Type       `json:"entity_type"`
	// ExpectedParentID is the parent the caller saw when checking permissions.
	// The update fails with a conflict if the entity was moved in the meantime.
	ExpectedParentID *uuid.UUID `json:"expected_parent_id,omitempty"`
}

type Tree []*Node

type Node struct {
	ListItem
	Children []*Node `json:"children,omitempty"`
}

func BuildTree(ctx context.Context, entities []ListItem) Tree {
	nodes := make(map[uuid.UUID]*Node, len(entities))
	for _, e := range entities {
		nodes[e.ID] = &Node{ListItem: e}
	}

	roots := Tree{}
	for _, node := range nodes {
		if node.ParentID != nil {
			parent, ok := nodes[*node.ParentID]
			if !ok {
				logger.Error(ctx, fmt.Errorf("parent not found")).
					Interface(FieldNode.String(), node).
					Msg("entity.BuildTree: parent not found, skipping node")
				continue
			}
			parent.Children = append(parent.Children, node)
		} else {
			roots = append(roots, node)
		}
	}
	if roots != nil {
		roots.sort()
	}

	return roots
}

func (t *Tree) sort() {
	var sortChildren func(nodes []*Node)
	sortChildren = func(nodes []*Node) {
		sort.Slice(nodes, func(i, j int) bool {
			if nodes[i].Name == nodes[j].Name {
				return nodes[i].ID.String() < nodes[j].ID.String()
			}
			return nodes[i].Name < nodes[j].Name
		})
		for _, node := range nodes {
			if node.Children != nil {
				sortChildren(node.Children)
			}
		}
	}
	sortChildren(*t)
}
