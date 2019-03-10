# Take aways

A simple Mailist application designed to teach the basics of Go programming language and also to introduce the Echo web framework.

## Go development setup:

### Windows

TBA

### Linux

TBA

### macOS

TBA

## Go Basics

TBA

### Go environment

- `go env`
- Directory structures
- Go documentation

### Go Packages

```bash
go get -u github.com/labstack/echo/...
```

#### Go Essencials:

- Variables and Types
- Exported vs Unexported names
- Syntax and Control Flow
- Arrays, Slices and Maps
- Functions
- Pointers
- Structs and Interfaces
- Concurrency
- Interface
- Concurrency
  - CSP and message sending
  - Shared Memory with Locks

```go
var Name string
Name = "Jake Peralta"

var Name string = "Jake Peralta"

Name := "Jake Peralta"
```

#### Echo Essencials:

- Serving Staics
- Routes
  - Groups
  - Nested Routes
  - Route specific middlewares
- Text, HTML and JSON responses
- Middlewares
- Error Handling
