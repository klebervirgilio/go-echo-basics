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

- GOPATH
- Directory structures
- Go documentation

### Go Dependecy Management

```bash
go get -u github.com/labstack/echo/...
```

#### Go Essencials:

- Keywords
- Syntax and Control Flow
- Exported vs Unexported names
- Functions
- Array/Slices and Interation
- Maps
- Struct
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
