load("@rules_go//go:def.bzl", _go_binary = "go_binary", _go_library = "go_library")

GO_MODULE = "github.com/xuzixiang/tiny-llm-serving-gateway"

def go_importpath(package = None):
    if package == None:
        package = native.package_name()
    if package:
        return GO_MODULE + "/" + package
    return GO_MODULE

def go_library(importpath = None, **kwargs):
    if importpath == None:
        importpath = go_importpath()
    _go_library(importpath = importpath, **kwargs)

go_binary = _go_binary
