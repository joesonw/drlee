load("@io_bazel_rules_go//go:def.bzl", "go_library", "go_test")

go_library(
    name = "go_default_library",
    srcs = ["sql.go"],
    importpath = "github.com/joesonw/drlee/pkg/core/sql",
    visibility = ["//visibility:public"],
    deps = [
        "//pkg/core:go_default_library",
        "//pkg/core/helpers:go_default_library",
        "//pkg/core/helpers/params:go_default_library",
        "//pkg/core/object:go_default_library",
        "//pkg/core/stream:go_default_library",
        "//pkg/utils:go_default_library",
        "@com_github_satori_go_uuid//:go_default_library",
        "@com_github_yuin_gopher_lua//:go_default_library",
    ],
)

go_test(
    name = "go_default_test",
    srcs = ["sql_test.go"],
    embed = [":go_default_library"],
    deps = [
        "//pkg/core:go_default_library",
        "//pkg/core/test:go_default_library",
        "@com_github_data_dog_go_sqlmock//:go_default_library",
        "@com_github_onsi_ginkgo//:go_default_library",
        "@com_github_onsi_gomega//:go_default_library",
        "@com_github_yuin_gopher_lua//:go_default_library",
    ],
)
