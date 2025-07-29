package state

// a bitmap representing a set of capabilities
type Permission uint64

// our builtin permissions
const ()

var BuiltInPerms = map[string]Permission{}

func (p Permission) Has(flag Permission) bool {
	return p&flag == flag
}
