package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var cmdRoot = &cobra.Command{
	Use:   "tka",
	Short: "tka is a CLI for Tailscale Kubernetes Auth",
	Long: `tka is a small CLI to sign in to a Kubernetes cluster using Tailscale identity.
It talks to a tka-api instance and helps you fetch ephemeral kubeconfigs.`,
}

var cmdGet = &cobra.Command{
	Use:   "get",
	Short: "Get resources from the tka server",
	Long: `Get command retrieves resources from your Tailscale Kubernetes Auth server.
It can be used to fetch various resources like kubeconfigs or clusters.`,
}

func init() {
	cobra.OnInitialize(initConfig)
	cmdRoot.AddCommand(cmdGet)
}

func initConfig() {
	viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))
	viper.AutomaticEnv()

	if os.Getenv("TKA_SERVER") == "" {
		log.Fatal("TKA_SERVER environment variable must be set to your tka-api server (e.g. http://tka-1.sphinx-map.ts.net:8123)")
	}

	viper.Set("server", os.Getenv("TKA_SERVER"))
}

func renderError(resp *http.Response) {
	var body map[string]any
	_ = json.NewDecoder(resp.Body).Decode(&body)

	msg := ""
	if errMsg, ok := body["error"].(string); ok {
		msg = errMsg
	} else {
		msg = resp.Status
	}
	fmt.Fprintf(os.Stderr, "Error: %s\n", msg)
}
