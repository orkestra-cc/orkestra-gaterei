//go:build !no_addons || addon_payments

package main

import (
	"github.com/orkestra/backend/internal/addons/payments"
	"github.com/orkestra/backend/pkg/sdk/module"
)

func init() {
	optionalModules["payments"] = func() module.Module { return payments.NewModule() }
}
