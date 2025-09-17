package apperr

type Rule string

const (
	RuleRequired      Rule = "required"
	RuleTooLong       Rule = "too_long"
	RuleCycle         Rule = "cycle"
	RuleMaxHierarchy  Rule = "max_hierarchy_depth"
	RuleInvalidFormat Rule = "invalid_format"
	RuleTooShort      Rule = "too_short"
	RuleDuplicate     Rule = "duplicate"
	RuleMismatch      Rule = "mismatch"
	RuleForbidden     Rule = "forbidden"
	RuleInvalidState  Rule = "invalid_state"
	RuleNotFound      Rule = "not_found"
)
