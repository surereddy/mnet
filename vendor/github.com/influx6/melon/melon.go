package melon

// Seeker defines a interface which exposes a Seek method to move
// a reader forward.
type Seeker interface {
	Seek(int) error
}

// Closer defines an interface which exposes a Close method
type Closer interface {
	Close() error
}
