load("@io_bazel_rules_go//go:def.bzl", "go_library")

go_library(
    name = "go_default_library",
    srcs = [
        "command.go",
        "debug.go",
        "server.go",
    ],
    importpath = "github.com/joesonw/drlee/pkg/commands",
    visibility = ["//visibility:public"],
    deps = [
        "//_proto:go_default_library",
        "//pkg/plugin:go_default_library",
        "//pkg/server:go_default_library",
        "@com_github_hashicorp_memberlist//:go_default_library",
        "@com_github_nsqio_go_diskqueue//:go_default_library",
        "@com_github_spf13_cobra//:go_default_library",
        "@in_gopkg_abiosoft_ishell_v2//:go_default_library",
        "@in_gopkg_yaml_v2//:go_default_library",
        "@org_golang_google_grpc//:go_default_library",
        "@org_uber_go_zap//:go_default_library",
    ],
)
