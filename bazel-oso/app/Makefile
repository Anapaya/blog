.PHONY: test_go test_bazel

test_go:
	go test

test_bazel:
	bazel test //:go_default_test

test_local:
	bazel test --override_repository=com_github_osohq_go_oso=$(PWD)/third_party/com_github_osohq_go_oso //:go_default_test

test_export_polar_header:
	bazel test --override_repository=com_github_osohq_go_oso=$(PWD)/third_party/export_polar_header //:go_default_test

test_export_libpolar:
	bazel test --override_repository=com_github_osohq_go_oso=$(PWD)/third_party/export_libpolar //:go_default_test

gazelle:
	bazel run //:gazelle -- update -go_naming_convention go_default_library $(PWD)

go_deps.bzl: go.mod go.sum
	bazel run //:gazelle -- update-repos -from_file=go.mod -to_macro=go_deps.bzl%go_deps -prune
