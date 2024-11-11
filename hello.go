//go:build mage
// +build mage

package main

import (
	"errors"

	"github.com/magefile/mage/mg"
)

type Hello mg.Namespace

// Hello says hello
func (Hello) Hello() error {
	return errors.New("not implemented")
}
