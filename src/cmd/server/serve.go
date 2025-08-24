package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/spechtlabs/go-otel-utils/otelzap"
	"github.com/spechtlabs/tailscale-k8s-auth/pkg/api"
	"github.com/spechtlabs/tailscale-k8s-auth/pkg/operator"
	ts "github.com/spechtlabs/tailscale-k8s-auth/pkg/tailscale"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"go.uber.org/zap"

	"tailscale.com/tailcfg"
)

var (
	serveCmd = &cobra.Command{
		Use:  "serve",
		Args: cobra.ExactArgs(0),
		RunE: runE,
	}
)

func runE(cmd *cobra.Command, _ []string) error {
	if debug {
		if file, err := os.ReadFile(viper.GetViper().ConfigFileUsed()); err == nil && len(file) > 0 {
			otelzap.L().Sugar().With("config_file", string(file)).Debug("Config file used")
		}
	}

	ctx, cancelFn := context.WithCancelCause(cmd.Context())
	interruptHandler(ctx, cancelFn)

	k8sOperator, err := operator.NewK8sOperator()
	if err != nil {
		cancelFn(err)
		return fmt.Errorf("%s", err.Display())
	}

	srv := ts.NewServer(hostname,
		ts.WithDebug(debug),
		ts.WithPort(port),
		ts.WithStateDir(tsNetStateDir),
		ts.WithReadTimeout(10*time.Second),
		ts.WithReadHeaderTimeout(5*time.Second),
		ts.WithWriteTimeout(20*time.Second),
		ts.WithIdleTimeout(120*time.Second),
	)

	tkaServer, err := api.NewTKAServer(srv, k8sOperator,
		api.WithDebug(debug),
		api.WithPeerCapName(tailcfg.PeerCapability(capName)),
	)
	if err != nil {
		cancelFn(err)
		return fmt.Errorf("%s", err.Display())
	}

	go func() {
		if err := tkaServer.Serve(ctx); err != nil {
			cancelFn(err.Cause())
			otelzap.L().WithError(err).FatalContext(ctx, "Failed to start TKA tailscale")
		}
	}()

	go func() {
		if err := k8sOperator.Start(ctx); err != nil {
			cancelFn(err.Cause())
			otelzap.L().WithError(err).FatalContext(ctx, "Failed to start k8s operator")
		}
	}()

	// Wait for context done
	<-ctx.Done()
	// No more logging to ctx from here onwards

	ctx = context.Background()
	if err := tkaServer.Shutdown(ctx); err != nil {
		return fmt.Errorf("%s", err.Display())
	}

	// Terminate accordingly
	if err := ctx.Err(); !errors.Is(err, context.Canceled) {
		otelzap.L().WithError(err).Fatal("Exiting")
	} else {
		otelzap.L().Info("Exiting")
	}

	return nil
}

func interruptHandler(ctx context.Context, cancelCtx context.CancelCauseFunc) {
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	go func() {
		select {
		// Wait for context cancel
		case <-ctx.Done():

		// Wait for signal
		case sig := <-sigs:
			switch sig {
			case syscall.SIGTERM:
				fallthrough
			case syscall.SIGINT:
				fallthrough
			case syscall.SIGQUIT:
				// On terminate signal, cancel context causing the program to terminate
				cancelCtx(context.Canceled)

			default:
				otelzap.L().WarnContext(ctx, "Received unknown signal", zap.String("signal", sig.String()))
			}
		}
	}()
}
