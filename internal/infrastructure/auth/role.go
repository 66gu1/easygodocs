package auth

type Role string

const (
	RoleAdmin Role = "admin"
	RoleRead  Role = "read"
	RoleWrite Role = "write"
)

func (role Role) IsValid() bool {
	switch role {
	case RoleAdmin, RoleRead, RoleWrite:
		return true
	default:
		return false
	}
}

func (role Role) String() string {
	return string(role)
}
