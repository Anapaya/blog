# Building oso with bazel

This package contains the failing showcase of building oso as a third-party
dependency with bazel.

## Problem

Running the regular Go tests works as expected:

```txt
$ go test
ok      github.com/Anapaya/blog/bazel-oso/app   0.002s

```

However, when running the tests with bazel, we fail to build:

```txt
$ bazel test //...
(14:07:41) INFO: Invocation ID: 65ae868e-149f-4fe4-9bc0-8bf58af5681d
(14:07:41) INFO: Current date is 2022-05-26
(14:07:41) INFO: Analyzed target //:go_default_test (0 packages loaded, 0 targets configured).
(14:07:41) INFO: Found 1 test target...
(14:07:41) ERROR: .../external/com_github_osohq_go_oso/internal/ffi/BUILD.bazel:3:11: GoCompilePkg external/com_github_osohq_go_oso/internal/ffi/ffi.a failed: (Exit 1): builder failed: error executing command bazel-out/k8-opt-exec-2B5CBBC6/bin/external/go_sdk/builder compilepkg -sdk external/go_sdk -installsuffix linux_amd64 -src external/com_github_osohq_go_oso/internal/ffi/ffi.go -embedroot '' -embedroot ... (remaining 35 arguments skipped)

Use --sandbox_debug to see verbose messages from the sandbox
com_github_anapaya_blog_bazel_oso_app/external/com_github_osohq_go_oso/internal/ffi/ffi.go:6:11: fatal error: native/polar.h: No such file or directory
compilation terminated.
compilepkg: error running subcommand external/go_sdk/pkg/tool/linux_amd64/cgo: exit status 2
Target //:go_default_test failed to build
Use --verbose_failures to see the command lines of failed build steps.
(14:07:41) INFO: Elapsed time: 0.275s, Critical Path: 0.07s
(14:07:41) INFO: 2 processes: 2 internal.
(14:07:41) FAILED: Build did NOT complete successfully
//:go_default_test                                              FAILED TO BUILD

(14:07:41) FAILED: Build did NOT complete successfully
```

The important bit of that output is the following line:

```txt
com_github_anapaya_blog_bazel_oso_app/external/com_github_osohq_go_oso/internal/ffi/ffi.go:6:11: fatal error: native/polar.h: No such file or directory
```

## Stages

To resolve this problem, we need to figure out what the required patch on the
generated build files. This is done in multiple stages.

1. Test with automatically generated go_repository (already done above).
1. Test with local repository override.
1. Test with initial patch.
1. Test with follow up patch.
1. Create patch for automatically generated go_repository.

After we have completed the last stage, we can generate a patch that is applied
automatically when fetching the go dependency.
The result can be found in [app-patched](../app-patched/).

### Test with local repository override

To create patches, it is very useful to checkout the dependency locally and add
an override to bazel. The alternative is to edit the build files in the bazel
cache. However, this is very hacky and can lead to stale cache files. We
recommend alwasy checking out the dependency locally. For this, we have
set up [com_github_osohq_go_oso](./third_party/com_github_osohq_go_oso/).

To add this repository, we simply did the following steps:

```txt
mkdir third_party
git clone git@github.com:osohq/go-oso.git third_party/com_github_osohq_go_oso

touch third_party/com_github_osohq_go_oso/WORKSPACE  # Add rules_go and gazelle
bazel run //:gazelle -- update -go_naming_convention import_alias $PWD/third_party/com_github_osohq_go_oso
```

This is essentially what gazelle does for external dependencies in the
`go_repository` rule.

Now that we have the local repository prepared, we can run our tests with
a local repostory override:

```txt
bazel test --override_repository=com_github_osohq_go_oso=$PWD/third_party/com_github_osohq_go_oso //:go_default_test
```

As expected, the test will fail with the exact same error.

### Test with initial patch

We duplicate the local repository, such that we can highlight the patch that
needs to be applied.

```txt
cp -r third_party/com_github_osohq_go_oso third_party/export_polar_header
```

We can run the test again with the following override:

```txt
bazel test --override_repository=com_github_osohq_go_oso=$PWD/third_party/export_polar_header //:go_default_test
```

This results again in the same error:

```txt
com_github_anapaya_blog_bazel_oso_app/external/com_github_osohq_go_oso/internal/ffi/ffi.go:6:11: fatal error: native/polar.h: No such file or directory
```

The crucial information here is that in file `internal/ffi/ffi.go` we need to be
able to access to the header file `native/polar.h`. Gazelle does not detect this
dependency, even though
[internal/ffi/ffi.go:6](https://github.com/osohq/go-oso/blob/476d2e1d68245a1f7817fc4d35b19622c9240b7e/internal/ffi/ffi.go#L6)
clearly lists this dependency.

We can work around this by exporting `native/polar.h` in
[internal/ffi/native/BUILD.bazel](./third_party/export_polar_header/internal/ffi/native/BUILD.bazel)
and adding an explicit dependency on it to
[internal/ffi/BUILD.bazel](./third_party/export_polar_header/internal/ffi/BUILD.bazel)

The diff looks like this:

```diff
--- com_github_osohq_go_oso/internal/ffi/BUILD.bazel    2022-05-26 15:46:11.204042146 +0200
+++ export_polar_header/internal/ffi/BUILD.bazel        2022-05-26 15:28:21.635661987 +0200
@@ -2,7 +2,10 @@
 
 go_library(
     name = "ffi",
-    srcs = ["ffi.go"],
+    srcs = [
+        "ffi.go",
+        "//internal/ffi/native:polar.h",  # keep
+    ],
     cgo = True,
     clinkopts = select({
         "@io_bazel_rules_go//go/platform:android_amd64": [
diff -ru -x 'bazel-*' com_github_osohq_go_oso/internal/ffi/native/BUILD.bazel export_polar_header/internal/ffi/native/BUILD.bazel
--- com_github_osohq_go_oso/internal/ffi/native/BUILD.bazel     2022-05-26 15:46:11.204042146 +0200
+++ export_polar_header/internal/ffi/native/BUILD.bazel 2022-05-26 15:28:33.779528656 +0200
@@ -20,3 +20,7 @@
     actual = ":native",
     visibility = ["//:__subpackages__"],
 )
+
+exports_files([
+    "polar.h",
+])
```

Now we can run our test again:

```txt
$ bazel test --override_repository=com_github_osohq_go_oso=$PWD/third_party/export_polar_header //:go_default_test
(16:13:14) INFO: Invocation ID: 2465dd76-d24b-4dfb-affb-2a3f6150b660
(16:13:14) INFO: Current date is 2022-05-26
(16:13:14) INFO: Analyzed target //:go_default_test (26 packages loaded, 378 targets configured).
(16:13:14) INFO: Found 1 test target...
(16:13:14) ERROR: BAZEL_CACHE/5f6193004b16afccec5c51781d888f5f/external/com_github_osohq_go_oso/internal/ffi/BUILD.bazel:3:11: GoCompilePkg external/com_github_osohq_go_oso/internal/ffi/ffi.a failed: (Exit 1): builder failed: error executing command bazel-out/k8-opt-exec-2B5CBBC6/bin/external/go_sdk/builder compilepkg -sdk external/go_sdk -installsuffix linux_amd64 -src external/com_github_osohq_go_oso/internal/ffi/ffi.go -src ... (remaining 41 arguments skipped)

Use --sandbox_debug to see verbose messages from the sandbox
gcc: error: internal/ffi/native/linux/libpolar.a: No such file or directory
compilepkg: error running subcommand /usr/bin/gcc: exit status 1
Target //:go_default_test failed to build
Use --verbose_failures to see the command lines of failed build steps.
(16:13:14) INFO: Elapsed time: 0.549s, Critical Path: 0.27s
(16:13:14) INFO: 2 processes: 2 internal.
(16:13:14) FAILED: Build did NOT complete successfully
//:go_default_test                                              FAILED TO BUILD

(16:13:14) FAILED: Build did NOT complete successfully
```

We have made some progress. We are greated with a new error.

```txt
gcc: error: internal/ffi/native/linux/libpolar.a: No such file or directory
```

### Test with follow up patch

Again, we duplicate the local repository, such that we can highlight the patch
that needs to be applied.

```txt
cp -r third_party/export_polar_header third_party/export_libpolar
```

We can run the test again with the following override:

```txt
bazel test --override_repository=com_github_osohq_go_oso=$PWD/third_party/export_libpolar //:go_default_test
```

This results again in the same error:

```txt
gcc: error: internal/ffi/native/linux/libpolar.a: No such file or directory
```

However, we see that gazelle generated the `clinkpots` in
[internal/ffi/BUILD.bazel](./third_party/export_polar_header/internal/ffi/BUILD.bazel)
to include `libpolar.a`.

We run the test with `--sandbox_debug` to see what is going on exactly:

```txt
bazel test --override_repository=com_github_osohq_go_oso=$PWD/third_party/export_polar_header --sandbox_debug //:go_default_test
(16:22:34) INFO: Invocation ID: 7bb50715-4ae6-43ec-b7c0-afd042ea07ee
(16:22:34) INFO: Current date is 2022-05-26
(16:22:34) INFO: Analyzed target //:go_default_test (0 packages loaded, 0 targets configured).
(16:22:34) INFO: Found 1 test target...
(16:22:35) ERROR: BAZEL_CACHE/5f6193004b16afccec5c51781d888f5f/external/com_github_osohq_go_oso/internal/ffi/BUILD.bazel:3:11: GoCompilePkg external/com_github_osohq_go_oso/internal/ffi/ffi.a failed: (Exit 1): linux-sandbox failed: error executing command 
  (cd BAZEL_CACHE/5f6193004b16afccec5c51781d888f5f/sandbox/linux-sandbox/17/execroot/com_github_anapaya_blog_bazel_oso_app && \
  exec env - \
    CC=/usr/bin/gcc \
    CGO_ENABLED=1 \
    GOARCH=amd64 \
    GOOS=linux \
    GOPATH='' \
    GOROOT=external/go_sdk \
    GOROOT_FINAL=GOROOT \
    PATH=/usr/bin:/bin \
    TMPDIR=/tmp \
  BAZEL_CACHE/install/d81761ab5244f5f4735b9254de6662ba/linux-sandbox -t 15 -w BAZEL_CACHE/5f6193004b16afccec5c51781d888f5f/sandbox/linux-sandbox/17/execroot/com_github_anapaya_blog_bazel_oso_app -w /tmp -w /dev/shm -D -- bazel-out/k8-opt-exec-2B5CBBC6/bin/external/go_sdk/builder compilepkg -sdk external/go_sdk -installsuffix linux_amd64 -src external/com_github_osohq_go_oso/internal/ffi/ffi.go -src external/com_github_osohq_go_oso/internal/ffi/native/polar.h -embedroot '' -embedroot bazel-out/k8-fastbuild/bin -embedlookupdir external/com_github_osohq_go_oso/internal/ffi -embedlookupdir external/com_github_osohq_go_oso/internal/ffi/native -arc 'github.com/osohq/go-oso/errors=github.com/osohq/go-oso/errors=bazel-out/k8-fastbuild/bin/external/com_github_osohq_go_oso/errors/errors.x' -arc 'github.com/osohq/go-oso/internal/ffi/native=github.com/osohq/go-oso/internal/ffi/native=bazel-out/k8-fastbuild/bin/external/com_github_osohq_go_oso/internal/ffi/native/native.x' -arc 'github.com/osohq/go-oso/types=github.com/osohq/go-oso/types=bazel-out/k8-fastbuild/bin/external/com_github_osohq_go_oso/types/types.x' -importpath github.com/osohq/go-oso/internal/ffi -p github.com/osohq/go-oso/internal/ffi -package_list bazel-out/k8-opt-exec-2B5CBBC6/bin/external/go_sdk/packages.txt -o bazel-out/k8-fastbuild/bin/external/com_github_osohq_go_oso/internal/ffi/ffi.a -x bazel-out/k8-fastbuild/bin/external/com_github_osohq_go_oso/internal/ffi/ffi.x -gcflags '' -asmflags '' -cppflags '-I external/com_github_osohq_go_oso/internal/ffi -I external/com_github_osohq_go_oso/internal/ffi/native -iquote .' -cflags '-U_FORTIFY_SOURCE -fstack-protector -Wunused-but-set-parameter -Wno-free-nonheap-object -fno-omit-frame-pointer -fno-canonical-system-headers -Wno-builtin-macro-redefined -D__DATE__="redacted" -D__TIMESTAMP__="redacted" -D__TIME__="redacted" -g -Wall -fPIC' -cxxflags '-U_FORTIFY_SOURCE -fstack-protector -Wunused-but-set-parameter -Wno-free-nonheap-object -fno-omit-frame-pointer -std=c++0x -fno-canonical-system-headers -Wno-builtin-macro-redefined -D__DATE__="redacted" -D__TIMESTAMP__="redacted" -D__TIME__="redacted" -fPIC' -objcflags '-U_FORTIFY_SOURCE -fstack-protector -Wunused-but-set-parameter -Wno-free-nonheap-object -fno-omit-frame-pointer -fno-canonical-system-headers -Wno-builtin-macro-redefined -D__DATE__="redacted" -D__TIMESTAMP__="redacted" -D__TIME__="redacted" -g -Wall -fPIC' -objcxxflags '-U_FORTIFY_SOURCE -fstack-protector -Wunused-but-set-parameter -Wno-free-nonheap-object -fno-omit-frame-pointer -std=c++0x -fno-canonical-system-headers -Wno-builtin-macro-redefined -D__DATE__="redacted" -D__TIMESTAMP__="redacted" -D__TIME__="redacted" -fPIC' -ldflags '-fuse-ld=gold -Wl,-no-as-needed -Wl,-z,relro,-z,now -B/usr/bin -pass-exit-codes -lstdc++ -lm internal/ffi/native/linux/libpolar.a -ldl -lm')
```

From that ouput we can see that the working directory is set to
`SANDBOX/com_github_anapaya_blog_bazel_oso_app` (via the -w flag). However, we
still try to link `internal/ffi/native/linux/libpolar.a`, which is a reltaive
path. Inspecting the sandbox (e.g., by using `tree`) shows that libpolar.a does
not exist in that location.

Thus, we need a way to make `libpolar.a` accessible in the sandbox during this
build step. According to
<https://github.com/bazelbuild/bazel-gazelle/issues/613#issuecomment-518678630>,
our best bet is to export `libpolar.a` and depend on it explicitely.

We can do this with the following patch:

```diff
--- export_polar_header/internal/ffi/BUILD.bazel        2022-05-26 15:28:21.635661987 +0200
+++ export_libpolar/internal/ffi/BUILD.bazel    2022-05-26 15:29:54.474644070 +0200
@@ -6,6 +6,9 @@
         "ffi.go",
         "//internal/ffi/native:polar.h",  # keep
     ],
+    cdeps = [
+        "//internal/ffi/native/linux:libpolar",  # keep
+    ],
     cgo = True,
     clinkopts = select({
         "@io_bazel_rules_go//go/platform:android_amd64": [
@@ -24,7 +27,7 @@
             "internal/ffi/native/macos/arm64/libpolar.a -ldl -lm",
         ],
         "@io_bazel_rules_go//go/platform:linux_amd64": [
-            "internal/ffi/native/linux/libpolar.a -ldl -lm",
+            "external/com_github_osohq_go_oso/internal/ffi/native/linux/libpolar.a -ldl -lm",  # keep
         ],
         "@io_bazel_rules_go//go/platform:windows_amd64": [
             "internal/ffi/native/windows/libpolar.a -lm -lws2_32 -luserenv -lbcrypt",
diff -ru -x 'bazel-*' export_polar_header/internal/ffi/native/linux/BUILD.bazel export_libpolar/internal/ffi/native/linux/BUILD.bazel
--- export_polar_header/internal/ffi/native/linux/BUILD.bazel   2022-05-26 15:25:18.253683346 +0200
+++ export_libpolar/internal/ffi/native/linux/BUILD.bazel       2022-05-26 15:27:38.580135184 +0200
@@ -1,4 +1,5 @@
 load("@io_bazel_rules_go//go:def.bzl", "go_library")
+load("@rules_cc//cc:defs.bzl", "cc_import")
 
 go_library(
     name = "linux",
@@ -12,3 +13,12 @@
     actual = ":linux",
     visibility = ["//:__subpackages__"],
 )
+
+cc_import(
+    name = "libpolar",
+    hdrs = [
+        "//internal/ffi/native:polar.h",
+    ],
+    static_library = "libpolar.a",
+    visibility = ["//visibility:public"],
+)
```

Now, we can run this test again:

```txt
bazel test --override_repository=com_github_osohq_go_oso=$PWD/third_party/export_libpolar //:go_default_test
(16:39:49) INFO: Invocation ID: 519fbdfa-1e3f-4d63-aad9-8a0d4f64b2e3
(16:39:49) INFO: Current date is 2022-05-26
(16:39:50) INFO: Analyzed target //:go_default_test (26 packages loaded, 441 targets configured).
(16:39:50) INFO: Found 1 test target...
Target //:go_default_test up-to-date:
  bazel-bin/go_default_test_/go_default_test
(16:39:50) INFO: Elapsed time: 0.417s, Critical Path: 0.10s
(16:39:50) INFO: 2 processes: 1 internal, 1 linux-sandbox.
(16:39:50) INFO: Build completed successfully, 2 total actio\
ns
//:go_default_test                                                       PASSED in 0.1s

Executed 1 out of 1 test: 1 test passes.
There were tests whose specified size is too big. Use the --te
(16:39:50) INFO: Build completed successfully, 2 total actio\
ns
```

Success! The test now runs without any issues.

### Creating the patch

Until now, we have used a local repostory with a manual override. We will now
create a patch that we can pass to `go_repository` which is applied after
gazelle has created all the build files for an external dependency.

The advantage is that we do not need to maintain the local repository and
we minimize the additional code that is committed.

In [patching bazel external
dependencies](https://brentley.dev/patching-bazel-external-dependencies/) the
author describes a good way of doing this. For our example here, we can take
advantage of the fact that we created a new copy per stage.

We assume that we have everything committed to git and have a clean checkout.
Then, we simply copy the fix build files to the initial repository and take the
git diff as the patch.

```text
cp -r ./third_party/export_libpolar/* ./third_party/com_github_osohq_go_oso
git diff -- ./third_party/com_github_osohq_go_oso > com_github_osohq_go_oso.patch
```

How this patch is then used is descriebd in [app-patched](../app-patched/)
