load("@io_bazel_rules_go//go:def.bzl", "go_library", "go_test")

go_library(
    name = "go_default_library",
    srcs = [
        "call.go",
        "callback.go",
        "execution_context.go",
        "resource.go",
        "upvalue.go",
    ],
    importpath = "github.com/joesonw/drlee/pkg/core",
    visibility = ["//visibility:public"],
    deps = [
        "//pkg/utils:go_default_library",
        "@com_github_yuin_gopher_lua//:go_default_library",
        "@org_uber_go_atomic//:go_default_library",
        "@org_uber_go_zap//:go_default_library",
    ],
)

go_test(
    name = "go_default_test",
    srcs = [
        "core_test.go",
        "execution_context_test.go",
        "resource_test.go",
    ],
    embed = [":go_default_library"],
    deps = [
        "@com_github_onsi_ginkgo//:go_default_library",
        "@com_github_onsi_gomega//:go_default_library",
        "@com_github_yuin_gopher_lua//:go_default_library",
    ],
)
