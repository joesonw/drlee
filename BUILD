package(default_visibility = ["//visibility:public"])

load("@io_bazel_rules_go//proto:compiler.bzl", "go_proto_compiler")
load("@bazel_gazelle//:def.bzl", "gazelle")
load("@io_bazel_rules_go//go:def.bzl", "TOOLS_NOGO", "nogo")
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
