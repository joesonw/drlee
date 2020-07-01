package(default_visibility = ["//visibility:public"])

load("@io_bazel_rules_go//proto:compiler.bzl", "go_proto_compiler")
load("@bazel_gazelle//:def.bzl", "gazelle")
load("@io_bazel_rules_go//go:def.bzl", "TOOLS_NOGO", "go_binary", "go_library", "nogo")
load("@com_github_bazelbuild_buildtools//buildifier:def.bzl", "buildifier")

#gazelle:exclude proto
#gazelle:prefix github.com/joesonw/drlee
gazelle(
    name = "gazelle",
    args = [
        "-build_file_name",
        "BUILD,BUILD.bazel",
    ],
    command = "fix",
    prefix = "github.com/joesonw/drlee",
)

buildifier(
    name = "buildifier",
)

buildifier(
    name = "buildifier_check",
    mode = "check",
)

go_library(
    name = "go_default_library",
    srcs = ["main.go"],
    importpath = "github.com/joesonw/drlee",
    deps = [
        "//pkg/commands:go_default_library",
        "//pkg/utils:go_default_library",
        "@com_github_spf13_cobra//:go_default_library",
        "@org_uber_go_zap//:go_default_library",
    ],
)

go_binary(
    name = "drlee",
    embed = [":go_default_library"],
)
