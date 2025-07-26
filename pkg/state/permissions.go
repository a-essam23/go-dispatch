package state

// a bitmap representing a set of capabilities
type Permission uint64

const (
	PermCanRead  Permission = 1 << iota
	PermCanWrite            // 2
)

var BuiltInPerms = map[string]Permission{
	"read":  PermCanRead,
	"write": PermCanWrite,
}

func (p Permission) Has(flag Permission) bool {
	return p&flag == flag
}
