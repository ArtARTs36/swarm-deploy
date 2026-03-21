package events

// UserAuthenticated is emitted when a user passes web authentication.
type UserAuthenticated struct {
	Username string
}

func (u *UserAuthenticated) Type() Type {
	return TypeUserAuthenticated
}
