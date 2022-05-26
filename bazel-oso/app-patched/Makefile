.PHONY: test_go test_bazel

test:
	go test
	bazel test //:go_default_test

gazelle:
	bazel run //:gazelle -- update -go_naming_convention go_default_library $(PWD)

go_deps.bzl: go.mod go.sum
	bazel run //:gazelle -- update-repos -from_file=go.mod -to_macro=go_deps.bzl%go_deps -prune
