package department

import (
	"context"
	"github.com/66gu1/easygodocs/internal/app/article"
	"github.com/66gu1/easygodocs/internal/infrastructure/apperror"
	"github.com/66gu1/easygodocs/internal/infrastructure/logger"
	"github.com/google/uuid"
	"sort"
	"time"
)

var (
	nameRequiredErr = &apperror.Error{
		Message:  "department name is required",
		Code:     apperror.BadRequest,
		LogLevel: apperror.LogLevelWarn,
	}
	idRequiredErr = &apperror.Error{
		Message:  "department ID is required",
		Code:     apperror.BadRequest,
		LogLevel: apperror.LogLevelWarn,
	}
)

type Department struct {
	ID        uuid.UUID  `json:"id"`
	Name      string     `json:"name"`
	ParentID  *uuid.UUID `json:"parent_id,omitempty"`
	UpdatedAt time.Time  `json:"updated_at"`
	CreatedAt time.Time  `json:"created_at"`
	DeletedAt *time.Time `json:"deleted_at,omitempty"`
}

type CreateDepartmentReq struct {
	Name     string     `json:"name"`
	ParentID *uuid.UUID `json:"parent_id,omitempty"`
}

func (req CreateDepartmentReq) Validate() error {
	if req.Name == "" {
		return nameRequiredErr
	}

	return nil
}

type UpdateDepartmentReq struct {
	ID       uuid.UUID  `json:"id"`
	Name     string     `json:"name"`
	ParentID *uuid.UUID `json:"parent_id,omitempty"`
}

func (req UpdateDepartmentReq) Validate() error {
	if req.ID == uuid.Nil {
		return idRequiredErr
	}
	if req.Name == "" {
		return nameRequiredErr
	}

	if req.ParentID != nil {
		if *req.ParentID == uuid.Nil {
			return &apperror.Error{
				Message:  "Parent department ID must be omitted or a valid value — it cannot be an empty UUID.",
				Code:     apperror.BadRequest,
				LogLevel: apperror.LogLevelWarn,
			}
		}
		if req.ID == *req.ParentID {
			return &apperror.Error{
				Message:  "department cannot be its own parent",
				Code:     apperror.BadRequest,
				LogLevel: apperror.LogLevelWarn,
			}
		}
	}

	return nil
}

type Tree []*Node

func (t *Tree) build(ctx context.Context, deps []Department, articles []article.ArticleNode) {
	nodes := make(map[uuid.UUID]*Node, len(deps))
	for _, d := range deps {
		nodes[d.ID] = &Node{ID: d.ID, Name: d.Name, ParentID: d.ParentID, Type: NodeTypeDepartment}
	}
	for _, a := range articles {
		nodes[a.ID] = &Node{ID: a.ID, Name: a.Name, ParentID: &a.ParentID, Type: NodeTypeArticle}
	}
	var roots Tree
	for _, node := range nodes {
		if node.ParentID != nil {
			parent, ok := nodes[*node.ParentID]
			if !ok {
				logger.Warn(ctx, nil).Str("id", node.ID.String()).Str("parent_id", node.ParentID.String()).
					Msg("department.Tree.build: parent not found, skipping node")
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
	*t = roots
}

func (t *Tree) sort() {
	var sortChildren func(nodes []*Node)
	sortChildren = func(nodes []*Node) {
		sort.Slice(nodes, func(i, j int) bool {
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

type Node struct {
	ID       uuid.UUID  `json:"id"`
	Name     string     `json:"name"`
	ParentID *uuid.UUID `json:"parent_id,omitempty"`
	Type     nodeType   `json:"type"`
	Children []*Node    `json:"children,omitempty"`
}

type nodeType string

const (
	NodeTypeDepartment nodeType = "department"
	NodeTypeArticle    nodeType = "article"
)
