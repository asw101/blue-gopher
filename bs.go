//go:build mage
// +build mage

package main

import (
	"errors"

	"github.com/magefile/mage/mg"
)

type Tmp mg.Namespace

// Hello says hello
func (Tmp) Hello() error {
	return errors.New("not implemented")

}
