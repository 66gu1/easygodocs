package user

type role string

const (
	RoleAdmin role = "admin"
	RoleRead  role = "read"
	RoleWrite role = "write"
)

func (role role) IsValid() bool {
	switch role {
	case RoleAdmin, RoleRead, RoleWrite:
		return true
	default:
		return false
	}
}

func (role role) String() string {
	return string(role)
}
