module github.com/laurentsimon/jupyter-lineage/cli/proxy

go 1.22

replace github.com/laurentsimon/jupyter-lineage/pkg v0.0.0 => ../../pkg

require github.com/laurentsimon/jupyter-lineage/pkg v0.0.0

require github.com/elazarl/goproxy v0.0.0-20231117061959-7cc037d33fb5 // indirect
