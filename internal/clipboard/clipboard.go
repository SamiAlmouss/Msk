package clipboard

// Interface allows CLI tests to use an in-memory clipboard.
type Interface interface {
	Read() (string, error)
	Write(string) error
}
type System struct{}
