package apperr_test

import (
	"testing"

	"github.com/66gu1/easygodocs/internal/infrastructure/apperr"
	"github.com/stretchr/testify/require"
)

func TestAppError_WithMethodsDoNotMutateOriginal(t *testing.T) {
	t.Parallel()

	base := apperr.New("message", "test/code", apperr.ClassBadRequest, apperr.LogLevelWarn).
		WithViolation(apperr.Violation{Field: "a", Rule: apperr.RuleRequired})

	withDetail := base.WithDetail("detail")
	withMessage := base.WithUserMessage("other message")
	withViolation := base.WithViolation(apperr.Violation{Field: "b", Rule: apperr.RuleTooLong})

	require.Equal(t, "message", base.Error())
	require.Equal(t, "message", base.Message)
	require.Len(t, base.Violations, 1)

	require.Equal(t, "detail", withDetail.Error())
	require.Equal(t, "other message", withMessage.Message)
	require.Len(t, withViolation.Violations, 2)
}
