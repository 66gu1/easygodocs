package db

import (
	"fmt"
	"github.com/66gu1/easygodocs/internal/infrastructure/apperror"
)

type tableName string

const (
	ArticleTableName    tableName = "articles"
	DepartmentTableName tableName = "departments"
)

type ArticleParentType string

const (
	ArticleParentTypeArticle    ArticleParentType = "article"
	ArticleParentTypeDepartment ArticleParentType = "department"
)

const (
	validateParentConditionsDepartment = ""
	validateParentConditionsArticle    = "AND parent_type = '%s'"
)

const (
	statusNotFound = "not_found"
	statusCycle    = "cycle"
	statusOK       = "ok"
)

var (
	parentNodFoundErr = &apperror.Error{
		Message:  "parent department not found",
		Code:     apperror.BadRequest,
		LogLevel: apperror.LogLevelWarn,
	}
	parentCycleErr = &apperror.Error{
		Message:  "cannot assign department as its own descendant (cycle detected)",
		Code:     apperror.BadRequest,
		LogLevel: apperror.LogLevelWarn,
	}
)

func GetRecursiveFetcherQuery(tableName tableName) string {
	return fmt.Sprintf(`
WITH RECURSIVE
    base AS (SELECT d.id, d.parent_id, d.name
             FROM %[1]s d
             WHERE d.deleted_at IS NULL
               AND d.id = ANY(?)),
    children AS (SELECT *
                 FROM base
                 UNION ALL
                 SELECT d.id, d.parent_id, d.name
                 FROM %[1]s d
                          JOIN children c ON d.parent_id = c.id
                 WHERE d.deleted_at IS NULL),
    parents AS (SELECT *
                FROM base
                UNION ALL
                SELECT d.id, d.parent_id, d.name
                FROM %[1]s d
                         JOIN parents p ON d.id = p.parent_id
                WHERE d.deleted_at IS NULL)
SELECT id, parent_id, name
FROM (SELECT *
      FROM children
      UNION
      SELECT *
      FROM parents) AS all_deps;
`, tableName)
}

func GetRecursiveDeleteQuery(tableName tableName) string {
	return fmt.Sprintf(`
	WITH RECURSIVE sub AS (
		SELECT id FROM %[1]s WHERE id = ?
	UNION ALL
	SELECT d.id
	FROM %[1]s d
	INNER JOIN sub sd ON d.parent_id = sd.id
	)
	UPDATE sub SET deleted_at = ?
	WHERE id IN (SELECT id FROM subdepartments);
	`, tableName)
}

func GetRecursiveValidateParentQuery(tableName tableName) (string, error) {
	condition, err := getValidateParentConditionByTableName(tableName)
	if err != nil {
		return "", fmt.Errorf("GetRecursiveValidateParentQuery: %w", err)
	}
	return fmt.Sprintf(`
    WITH RECURSIVE tree AS (
    SELECT * FROM %[1]s WHERE id = ? AND deleted_at IS NULL
    UNION ALL
    SELECT d.* FROM %[1]s d
    INNER JOIN tree dt ON dt.parent_id = d.id
    WHERE d.deleted_at IS NULL %[2]s
)
SELECT
    CASE
        WHEN NOT EXISTS (SELECT 1 FROM tree) THEN 'not_found'
        WHEN EXISTS (SELECT 1 FROM tree WHERE id = ?) THEN 'cycle'
        ELSE 'ok'
    END AS status;
`, tableName, condition), nil
}

func getValidateParentConditionByTableName(name tableName) (string, error) {
	switch name {
	case DepartmentTableName:
		return validateParentConditionsDepartment, nil
	case ArticleTableName:
		return validateParentConditionsArticle, nil
	default:
		return "", fmt.Errorf("GetValidateParentConditionByTableName: unexpected table name %s", name)
	}
}

func GetValidateParentErrorByStatus(status string) error {
	switch status {
	case statusNotFound:
		return parentNodFoundErr
	case statusCycle:
		return parentCycleErr
	case statusOK:
		return nil
	default:
		return fmt.Errorf("gormRepo.ValidateParent: unexpected status %s", status)
	}
}
