//go:build !no_addons || addon_dev

package main

import (
	"github.com/orkestra-cc/orkestra-addon-dev"
	"github.com/orkestra-cc/orkestra-sdk/module"
)

func init() {
	optionalModules["dev"] = func() module.Module { return dev.NewModule() }
}
