# Building oso with bazel (patched)

This package contains the showcase of building oso as a third-party
dependency with bazel.

To prepare the patch, we followed the stages in [app](../app/). Check it out to
learn more.

The requiered patch is located in
[patches/com_github_osohq_go_oso.patch](./patches/com_github_osohq_go_oso.patch)
and registered in [./go_deps.bzl](./go_deps.bzl).

Note that we need to set `-p5` because the local repositories are deeply nested
in our git repository, and thus the file paths have a fairly long prefix.
Usually, you will need to only provide `-p1` to strip the `a/` and `b/` prefixes
from the file paths.


