package main

import (
	"net/http"
	"os"

	"github.com/spechtlabs/tka/internal/cli/pretty_print"
	"github.com/spechtlabs/tka/pkg/api"
	"github.com/spechtlabs/tka/pkg/models"
	"github.com/spf13/cobra"
)

var cmdSignout = &cobra.Command{
	Use:     "signout",
	Aliases: []string{"logout"},
	Short:   "Sign out and remove access from the cluster",
	Example: "tka signout",
	RunE: func(_ *cobra.Command, _ []string) error {
		_, _, err := doRequestAndDecode[models.UserLoginResponse](http.MethodPost, api.LogoutApiRoute, nil, http.StatusOK, http.StatusProcessing)
		if err != nil {
			pretty_print.PrintError(err.Cause())
			os.Exit(1)
		}

		pretty_print.PrintOk("You have been signed out")
		return nil
	},
}
