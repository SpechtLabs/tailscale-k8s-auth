package main

import (
	"context"
	"net/http"
	"os"
	"time"

	"github.com/sierrasoftworks/humane-errors-go"
	async_op2 "github.com/spechtlabs/tailscale-k8s-auth/internal/cli/async_operation"
	"github.com/spechtlabs/tailscale-k8s-auth/internal/cli/pretty_print"
	tkaApi "github.com/spechtlabs/tailscale-k8s-auth/pkg/api"
	"github.com/spf13/cobra"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/clientcmd/api"
)

func init() {
	cmdRoot.AddCommand(cmdKubeconfig)
	cmdGet.AddCommand(cmdGetKubeconfig)
}

var cmdKubeconfig = &cobra.Command{
	Use:     "kubeconfig",
	Short:   "Fetch your temporary kubeconfig",
	Example: "tka kubeconfig",
	Args:    cobra.ExactArgs(0),
	RunE:    getKubeconfig,
}

var cmdGetKubeconfig = &cobra.Command{
	Use:     "kubeconfig",
	Short:   "Fetch your temporary kubeconfig",
	Example: "tka get kubeconfig",
	Args:    cobra.ExactArgs(0),
	RunE:    getKubeconfig,
}

func getKubeconfig(_ *cobra.Command, _ []string) error {
	kubecfg, err := fetchKubeConfig()
	if err != nil {
		pretty_print.PrintError(err)
		os.Exit(1)
	}

	file, err := serializeKubeconfig(kubecfg)
	if err != nil {
		pretty_print.PrintError(err)
		os.Exit(1)
	}

	pretty_print.PrintOk("kubeconfig saved to", file)

	return nil
}

func fetchKubeConfig() (*api.Config, humane.Error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	pollFunc := func() (api.Config, humane.Error) {
		if cfg, _, err := doRequestAndDecode[api.Config](http.MethodGet, tkaApi.KubeconfigApiRoute, nil, http.StatusOK); err == nil {
			return *cfg, nil
		} else {
			return api.Config{}, err
		}
	}

	operation := async_op2.NewSpinner[api.Config](pollFunc,
		async_op2.WithInProgressMessage("Waiting for kubeconfig to be ready..."),
		async_op2.WithDoneMessage("Kubeconfig is ready."),
		async_op2.WithFailedMessage("Fetching kubeconfig failed."),
	)

	result, err := operation.Run(ctx)
	if err != nil {
		return nil, humane.Wrap(err, "failed to fetch kubeconfig")
	}
	return result, nil
}

func serializeKubeconfig(kubecfg *api.Config) (string, humane.Error) {
	out, err := clientcmd.Write(*kubecfg)
	if err != nil {
		return "", humane.Wrap(err, "failed to serialize kubeconfig")
	}

	tempFile, err := os.CreateTemp("", "kubeconfig-*.yaml")
	if err != nil {
		return "", humane.Wrap(err, "failed to create temp kubeconfig")
	}
	defer func() { _ = tempFile.Close() }()

	_, err = tempFile.Write(out)
	if err != nil {
		return "", humane.Wrap(err, "failed to write temp kubeconfig")
	}

	if err := os.Setenv("KUBECONFIG", tempFile.Name()); err != nil {
		return "", humane.Wrap(err, "failed to set KUBECONFIG")
	}

	return tempFile.Name(), nil
}
