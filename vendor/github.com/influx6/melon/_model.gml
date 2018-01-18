import (
  "context"
  {{range attrs "imports"}}
  {{quote .}}
  {{end}}
)

// {{sel "Name" | capitalize}}UniqueHash defines a unique hash for {{sel "Name"}} which can
// be used to reference a given instance within a context.ValueBag or a google context.Context
// value store.
const {{sel "Name" | capitalize}}UniqueHash = {{ (joinVariadic ":" (sel "Name") (sel "Type")) | sha1 | quote }}

// {{sel "Name"}}Reader defines reader for {{sel "Type"}} type.
type {{sel "Name"}}Reader interface {
  Read{{sel "Name"}}() ({{sel "Type"}}, error)
}

// {{sel "Name"}}ReadCloser defines reader and closer for {{sel "Type"}} type.
type {{sel "Name"}}ReadCloser interface {
  Closer
  {{sel "Name"}}Reader
}

// {{sel "Name"}}StreamReader defines reader {{sel "Type"}} type.
type {{sel "Name"}}StreamReader interface {
  Read(int) ([]{{sel "Type"}}, error)
}

// {{sel "Name"}}StreamReadCloser defines reader and closer for {{sel "Type"}} type.
type {{sel "Name"}}StreamReadCloser interface {
  Closer
  {{sel "Name"}}StreamReader
}

// {{sel "Name"}}Writer defines writer for {{sel "Type"}} type.
type {{sel "Name"}}Writer interface {
  Write{{sel "Name"}}({{sel "Type"}}) error
}

// {{sel "Name"}}WriteCloser defines writer and closer for {{sel "Type"}} type.
type {{sel "Name"}}WriteCloser interface {
  Closer
  {{sel "Name"}}Writer
}

// {{sel "Name"}}StreamWrite defines writer for {{sel "Type"}} type.
type {{sel "Name"}}StreamWriter interface {
  Write([]{{sel "Type"}}) (int, error)
}

// {{sel "Name"}}StreamWriteCloser defines writer and closer for {{sel "Type"}} type.
type {{sel "Name"}}StreamWriteCloser interface {
  Closer
  {{sel "Name"}}StreamWriter
}

// {{sel "Name"}}ReadWriteCloser composes reader types with closer for {{sel "Type"}}.
// with associated close method.
type {{sel "Name"}}ReadWriteCloser interface {
  Closer
  {{sel "Name"}}Reader
  {{sel "Name"}}Writer
}

// {{sel "Name"}}StreamReadWriteCloser composes stream types with closer for {{sel "Type"}}.
type {{sel "Name"}}Stream interface {
  Closer
  {{sel "Name"}}StreamReader
  {{sel "Name"}}StreamWriter
}
