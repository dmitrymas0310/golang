package migrations

import "embed"

// Files позволяет интеграционным тестам применять те же SQL-файлы, что и Goose.
//
//go:embed *.sql
var Files embed.FS
