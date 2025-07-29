package dto

import (
	"context"
	"fmt"
	user "github.com/66gu1/easygodocs/internal/infrastructure/auth"
	"github.com/66gu1/easygodocs/internal/infrastructure/logger"
	"github.com/google/uuid"
	"sort"
	"time"
)

type EntityType string

const (
	EntityTypeDepartment EntityType = "department"
	EntityTypeArticle    EntityType = "article"
)

type Entity struct {
	Type EntityType `json:"type"`
	ID   uuid.UUID  `json:"id"`
}

type Hierarchy struct {
	Entity Entity  `json:"entity"`
	Parent *Entity `json:"parent,omitempty"`
}

type HierarchyWithName struct {
	Hierarchy
	EntityName string `json:"entity_name"`
}

type DeleteRequest struct {
	Entity    Entity    `json:"entity"`
	DeletedAt time.Time `json:"deleted_at"`
}

type CheckPermissionRequest struct {
	Entity Entity    `json:"entity"`
	Role   user.Role `json:"role"`
}

type Tree []*Node

type Node struct {
	HierarchyWithName
	Children []*Node `json:"children,omitempty"`
}

func BuildTree(ctx context.Context, entities []HierarchyWithName) Tree {
	nodes := make(map[Entity]*Node, len(entities))
	for _, e := range entities {
		nodes[e.Entity] = &Node{HierarchyWithName: e}
	}

	var roots Tree
	for _, node := range nodes {
		if node.Parent != nil {
			parent, ok := nodes[*node.Parent]
			if !ok {
				logger.Error(ctx, fmt.Errorf("parent not found")).Interface("node", node).
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

	return roots
}

func (t *Tree) sort() {
	var sortChildren func(nodes []*Node)
	sortChildren = func(nodes []*Node) {
		sort.Slice(nodes, func(i, j int) bool {
			return nodes[i].EntityName < nodes[j].EntityName
		})
		for _, node := range nodes {
			if node.Children != nil {
				sortChildren(node.Children)
			}
		}
	}
	sortChildren(*t)
}
