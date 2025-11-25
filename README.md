#  Article Service

## Overview

The application has a structure based on Go convention:
- https://github.com/golang-standards/project-layout

---

## Concurrency Patterns

The application uses:
- **Worker Pool** – for parallel processing of pages with articles
- **Fan-in** – to combine results from multiple goroutines into a single output channel
- **Rate limit** – to avoid possible errors(429 Too Many Requests etc.) from external api
- **Pipeline** – to gradually process our articles

---

## Tests

Covered the most important core logic.

- Pattern: **TableDrivenTests**
- Library: [`testify`](https://github.com/stretchr/testify)

Run from the root project directory to see code coverage:

```bash
$ go test ./... -coverprofile=coverage.out
$ go tool cover -html=coverage.out
```

---

## Application Initialization Steps

1. Create application
2. Init app(logs, pars args etc.)
3. Run all parallel processes
4. On `SIGURG` signal, context cancel or successful result gracefully shut down the application

---

## Using

The application accepts **one command-line argument**:

| Flag   | Type    | Required | Description              |
|--------|---------|:--------:|--------------------------|
| `-l=8` | int     |   YES    | The size of top articles |

### Examples

```bash
# build
go build -o ./bin/top-articles ./cmd/articlesservice

# run (basic)
./bin/top-articles -l=10
