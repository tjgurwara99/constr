# Simple utility to generate constructors in Go for types

Usage

Given a file abc.go with the following content

```Go
package test

import "io"

type Something struct {
	some   string
	writer io.Writer
}
```

Running the following command

```sh
constr -t Data abc.go
```

would change the abc.go file to the following

```Go
package test

import "io"

type Something struct {
	some   string
	writer io.Writer
}

func NewSomething(some string,
	writer io.Writer) *Something {
	return &Something{some: some,
		writer: writer}
}
```