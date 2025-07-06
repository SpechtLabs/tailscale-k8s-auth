package operator

import (
	"github.com/go-logr/zapr"
	"github.com/sierrasoftworks/humane-errors-go"
	"github.com/spechtlabs/tailscale-k8s-auth/api/v1alpha1"
	"github.com/spechtlabs/tailscale-k8s-auth/internal/controller"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/metrics/server"

	"github.com/spechtlabs/go-otel-utils/otelzap"

	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
)

var scheme = runtime.NewScheme()

func NewK8sOperator() humane.Error {
	// Register the schemes
	if err := clientgoscheme.AddToScheme(scheme); err != nil {
		otelzap.L().WithError(err).Fatal("failed to add clientgoscheme to scheme")
	}

	if err := v1alpha1.AddToScheme(scheme); err != nil {
		otelzap.L().WithError(err).Fatal("failed to add v1alpha1 to scheme")
	}

	ctrl.SetLogger(zapr.NewLogger(otelzap.L().Logger))

	// GetConfigOrDie creates a *rest.Config for talking to a Kubernetes API server.
	// If --kubeconfig is set, will use the kubeconfig file at that location.  Otherwise will assume running
	// in cluster and use the cluster provided kubeconfig.
	//
	// The returned `*rest.Config` has client-side ratelimting disabled as we can rely on API priority and
	// fairness. Set its QPS to a value equal or bigger than 0 to re-enable it.
	//
	// It also applies saner defaults for QPS and burst based on the Kubernetes
	// controller manager defaults (20 QPS, 30 burst)
	//
	// Config precedence:
	//
	// * --kubeconfig flag pointing at a file
	//
	// * KUBECONFIG environment variable pointing at a file
	//
	// * In-cluster config if running in cluster
	//
	// * $HOME/.kube/config if exists.

	// If we run in-cluster then we also do leader election.
	// But for local debugging, that's not needed
	inCluster := isInCluster()
	leaderElectionNamespace := ""
	if !inCluster {
		leaderElectionNamespace = "default"
	}

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme:                  scheme,
		HealthProbeBindAddress:  "0",
		LeaderElection:          inCluster,
		LeaderElectionNamespace: leaderElectionNamespace,
		LeaderElectionID:        "controller.tka.specht.dev",
		Metrics: server.Options{
			BindAddress: "0",
		},
	})
	if err != nil {
		otelzap.L().WithError(err).Fatal("failed to start manager")
	}

	if err := controller.NewTKAReconciler(mgr.GetClient(), mgr.GetScheme()).SetupWithManager(mgr); err != nil {
		otelzap.L().WithError(err).Fatal("failed to create controller")
	}

	return nil
}

func isInCluster() bool {
	_, err := rest.InClusterConfig()
	return err == nil
}
