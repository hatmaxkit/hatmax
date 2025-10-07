module github.com/adrianpk/hatmax-ref/services/todo

go 1.23

require (
	github.com/adrianpk/hatmax/pkg/lib/hm v0.0.0-00010101000000-000000000000
)

// Development mode: use local hatmax library
replace github.com/adrianpk/hatmax => ../../../../
