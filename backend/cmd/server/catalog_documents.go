//go:build !no_addons || addon_documents

package main

import (
	"github.com/orkestra-cc/orkestra-addon-documents"
	"github.com/orkestra-cc/orkestra-sdk/module"
)

func init() {
	optionalModules["documents"] = func() module.Module { return documents.NewModule() }
}
