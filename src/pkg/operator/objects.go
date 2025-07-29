package operator

import (
	"fmt"
	"time"

	"github.com/spechtlabs/tailscale-k8s-auth/api/v1alpha1"
	authenticationv1 "k8s.io/api/authentication/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd/api"
)

func formatSigninObjectName(userName string) string {
	return fmt.Sprintf("tka-user-%s", userName)
}

func newSignin(userName, role string, validUntil time.Time) *v1alpha1.TkaSignin {
	return &v1alpha1.TkaSignin{
		ObjectMeta: metav1.ObjectMeta{
			Name:      formatSigninObjectName(userName),
			Namespace: "tka-dev", // TODO(cedi): make this dynamic...
		},
		Spec: v1alpha1.TkaSigninSpec{
			Username:   userName,
			Role:       role,
			ValidUntil: validUntil.Format(time.RFC3339),
		},
	}
}

func newSigninStatus(validUntil time.Time) *v1alpha1.TkaSigninStatus {
	now := time.Now()
	period := validUntil.Sub(now)

	return &v1alpha1.TkaSigninStatus{
		Provisioned:    false,
		ValidityPeriod: period.String(),
		SignedInAt:     now.Format(time.RFC3339),
	}
}

func newServiceAccount(signIn *v1alpha1.TkaSignin) *corev1.ServiceAccount {
	return &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:      formatSigninObjectName(signIn.Spec.Username),
			Namespace: signIn.Namespace,
			Annotations: map[string]string{
				ValidUntilAnnotation: signIn.Spec.ValidUntil,
			},
		},
	}
}

func newKubeconfig(contextName string, restCfg *rest.Config, token string, clusterName string, userEntry string) *api.Config {
	return &api.Config{
		Kind:           "Config",
		APIVersion:     "v1",
		CurrentContext: contextName,
		Clusters: map[string]*api.Cluster{
			clusterName: {
				Server:                   restCfg.Host,
				CertificateAuthorityData: restCfg.CAData,
				InsecureSkipTLSVerify:    restCfg.Insecure,
			},
		},
		AuthInfos: map[string]*api.AuthInfo{
			userEntry: {
				Token: token,
			},
		},
		Contexts: map[string]*api.Context{
			contextName: {
				Cluster:  clusterName,
				AuthInfo: userEntry,
			},
		},
	}
}

func newTokenRequest(expirationSeconds int64) *authenticationv1.TokenRequest {
	return &authenticationv1.TokenRequest{
		Spec: authenticationv1.TokenRequestSpec{
			ExpirationSeconds: &expirationSeconds,
			// Audiences:         []string{"https://kubernetes.default.svc.cluster.local"}, // TODO(cedi): implement properly
		},
	}
}

func newRoleRef(signIn *v1alpha1.TkaSignin) rbacv1.RoleRef {
	return rbacv1.RoleRef{
		APIGroup: "rbac.authorization.k8s.io",
		Kind:     "ClusterRole",
		Name:     signIn.Spec.Role,
	}
}

func newClusterRoleBinding(signIn *v1alpha1.TkaSignin) *rbacv1.ClusterRoleBinding {
	username := formatSigninObjectName(signIn.Spec.Username)

	return &rbacv1.ClusterRoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-binding", username),
			Namespace: signIn.Namespace,
			Annotations: map[string]string{
				ValidUntilAnnotation: signIn.Spec.ValidUntil,
			},
		},
		Subjects: []rbacv1.Subject{
			{
				Kind:      "ServiceAccount",
				Name:      username,
				Namespace: "tka-dev", // TODO(cedi): make dynamic
			},
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "ClusterRole",
			Name:     signIn.Spec.Role,
		},
	}
}
