//go:build mage

package main

import (
	"context"

	"github.com/magefile/mage/mg"

	//mage:import
	"github.com/dosquad/mage"
)

func init() {
}

// TestLocal update, protoc, format, tidy, lint & test.
func TestLocal(ctx context.Context) {
	mg.SerialCtxDeps(ctx, mage.Golang.Lint)
	mg.SerialCtxDeps(ctx, mage.Golang.Test)
}

var Default = TestLocal
