package resources

import (
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"

	etalbaasv1alpha1 "github.com/kou-etal/etalbaas/operator/api/v1alpha1"
)

// DesiredDefaultDenyNetworkPolicy creates a default-deny ingress NetworkPolicy.
func DesiredDefaultDenyNetworkPolicy(project *etalbaasv1alpha1.Project) *networkingv1.NetworkPolicy {
	namespace := "project-" + project.Name
	projectID := project.Name
	userID := project.Labels[LabelUserID]
	plan := project.Spec.Plan

	return &networkingv1.NetworkPolicy{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "default-deny-ingress",
			Namespace: namespace,
			Labels:    ComponentLabels(projectID, userID, plan, "networkpolicy"),
		},
		Spec: networkingv1.NetworkPolicySpec{
			PodSelector: metav1.LabelSelector{},
			PolicyTypes: []networkingv1.PolicyType{
				networkingv1.PolicyTypeIngress,
			},
		},
	}
}

// DesiredAllowIntraNamespaceNetworkPolicy creates a NetworkPolicy allowing
// communication within the same namespace.
func DesiredAllowIntraNamespaceNetworkPolicy(project *etalbaasv1alpha1.Project) *networkingv1.NetworkPolicy {
	namespace := "project-" + project.Name
	projectID := project.Name
	userID := project.Labels[LabelUserID]
	plan := project.Spec.Plan

	return &networkingv1.NetworkPolicy{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "allow-intra-namespace",
			Namespace: namespace,
			Labels:    ComponentLabels(projectID, userID, plan, "networkpolicy"),
		},
		Spec: networkingv1.NetworkPolicySpec{
			PodSelector: metav1.LabelSelector{},
			PolicyTypes: []networkingv1.PolicyType{
				networkingv1.PolicyTypeIngress,
			},
			Ingress: []networkingv1.NetworkPolicyIngressRule{
				{
					From: []networkingv1.NetworkPolicyPeer{
						{
							PodSelector: &metav1.LabelSelector{},
						},
					},
				},
			},
		},
	}
}

// DesiredAllowPlatformNetworkPolicy creates a NetworkPolicy allowing
// ingress from the platform-system namespace (Gateway, Operator).
func DesiredAllowPlatformNetworkPolicy(project *etalbaasv1alpha1.Project, platformNamespace string) *networkingv1.NetworkPolicy {
	namespace := "project-" + project.Name
	projectID := project.Name
	userID := project.Labels[LabelUserID]
	plan := project.Spec.Plan

	return &networkingv1.NetworkPolicy{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "allow-platform",
			Namespace: namespace,
			Labels:    ComponentLabels(projectID, userID, plan, "networkpolicy"),
		},
		Spec: networkingv1.NetworkPolicySpec{
			PodSelector: metav1.LabelSelector{},
			PolicyTypes: []networkingv1.PolicyType{
				networkingv1.PolicyTypeIngress,
			},
			Ingress: []networkingv1.NetworkPolicyIngressRule{
				{
					From: []networkingv1.NetworkPolicyPeer{
						{
							NamespaceSelector: &metav1.LabelSelector{
								MatchLabels: map[string]string{
									"kubernetes.io/metadata.name": platformNamespace,
								},
							},
						},
					},
				},
			},
		},
	}
}

// DesiredAllowCNPGNetworkPolicy creates a NetworkPolicy allowing ingress from
// the cnpg-system namespace. The CNPG controller needs to reach PostgreSQL pods
// on port 8000 to extract instance status via the /pg/status endpoint.
func DesiredAllowCNPGNetworkPolicy(project *etalbaasv1alpha1.Project) *networkingv1.NetworkPolicy {
	namespace := "project-" + project.Name
	projectID := project.Name
	userID := project.Labels[LabelUserID]
	plan := project.Spec.Plan

	statusPort := intstr.FromInt32(8000)
	tcp := corev1.ProtocolTCP

	return &networkingv1.NetworkPolicy{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "allow-cnpg-controller",
			Namespace: namespace,
			Labels:    ComponentLabels(projectID, userID, plan, "networkpolicy"),
		},
		Spec: networkingv1.NetworkPolicySpec{
			PodSelector: metav1.LabelSelector{},
			PolicyTypes: []networkingv1.PolicyType{
				networkingv1.PolicyTypeIngress,
			},
			Ingress: []networkingv1.NetworkPolicyIngressRule{
				{
					From: []networkingv1.NetworkPolicyPeer{
						{
							NamespaceSelector: &metav1.LabelSelector{
								MatchLabels: map[string]string{
									"kubernetes.io/metadata.name": "cnpg-system",
								},
							},
						},
					},
					Ports: []networkingv1.NetworkPolicyPort{
						{Protocol: &tcp, Port: &statusPort},
					},
				},
			},
		},
	}
}

// DesiredAllowEnvoyGatewayNetworkPolicy creates a NetworkPolicy allowing ingress
// from the envoy-gateway-system namespace. The Envoy proxy needs to reach function
// pods on their HTTP ports to forward invocation requests.
func DesiredAllowEnvoyGatewayNetworkPolicy(project *etalbaasv1alpha1.Project, gatewayNamespace string) *networkingv1.NetworkPolicy {
	namespace := "project-" + project.Name
	projectID := project.Name
	userID := project.Labels[LabelUserID]
	plan := project.Spec.Plan

	return &networkingv1.NetworkPolicy{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "allow-envoy-gateway",
			Namespace: namespace,
			Labels:    ComponentLabels(projectID, userID, plan, "networkpolicy"),
		},
		Spec: networkingv1.NetworkPolicySpec{
			PodSelector: metav1.LabelSelector{},
			PolicyTypes: []networkingv1.PolicyType{
				networkingv1.PolicyTypeIngress,
			},
			Ingress: []networkingv1.NetworkPolicyIngressRule{
				{
					From: []networkingv1.NetworkPolicyPeer{
						{
							NamespaceSelector: &metav1.LabelSelector{
								MatchLabels: map[string]string{
									"kubernetes.io/metadata.name": gatewayNamespace,
								},
							},
						},
					},
				},
			},
		},
	}
}

// DesiredEgressNetworkPolicy creates a NetworkPolicy restricting egress traffic.
// Allows: DNS (kube-system:53), HTTPS (443), intra-namespace,
//         NATS (platform-system:4222), Storage REST (platform-system:8080).
// Blocks: cloud metadata services (169.254.169.254, etc.) to prevent IAM token theft.
func DesiredEgressNetworkPolicy(project *etalbaasv1alpha1.Project, platformNamespace string) *networkingv1.NetworkPolicy {
	namespace := "project-" + project.Name
	projectID := project.Name
	userID := project.Labels[LabelUserID]
	plan := project.Spec.Plan

	dnsPort := intstr.FromInt32(53)
	httpsPort := intstr.FromInt32(443)
	apiServerPort := intstr.FromInt32(6443)
	natsPort := intstr.FromInt32(4222)
	storagePort := intstr.FromInt32(8080)
	udp := corev1.ProtocolUDP
	tcp := corev1.ProtocolTCP

	return &networkingv1.NetworkPolicy{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "egress-restrict",
			Namespace: namespace,
			Labels:    ComponentLabels(projectID, userID, plan, "networkpolicy"),
		},
		Spec: networkingv1.NetworkPolicySpec{
			PodSelector: metav1.LabelSelector{},
			PolicyTypes: []networkingv1.PolicyType{
				networkingv1.PolicyTypeEgress,
			},
			Egress: []networkingv1.NetworkPolicyEgressRule{
				// Allow DNS to kube-system
				{
					To: []networkingv1.NetworkPolicyPeer{
						{
							NamespaceSelector: &metav1.LabelSelector{
								MatchLabels: map[string]string{
									"kubernetes.io/metadata.name": "kube-system",
								},
							},
						},
					},
					Ports: []networkingv1.NetworkPolicyPort{
						{Protocol: &udp, Port: &dnsPort},
						{Protocol: &tcp, Port: &dnsPort},
					},
				},
				// Allow HTTPS outbound + Kubernetes API server, block cloud metadata services.
				// Prevents IAM token theft (Capital One 2019-style attack) in multi-tenant env.
				// Port 6443: kube-proxy DNAT rewrites ClusterIP:443 → node:6443; Calico
				// evaluates post-DNAT, so 6443 must be allowed for CNPG init jobs.
				{
					To: []networkingv1.NetworkPolicyPeer{
						{
							IPBlock: &networkingv1.IPBlock{
								CIDR: "0.0.0.0/0",
								Except: []string{
									"169.254.169.254/32", // AWS / GCP / Azure metadata service
									"100.100.100.200/32", // Alibaba Cloud metadata service
									"169.254.170.2/32",   // AWS ECS task metadata endpoint
								},
							},
						},
					},
					Ports: []networkingv1.NetworkPolicyPort{
						{Protocol: &tcp, Port: &httpsPort},
						{Protocol: &tcp, Port: &apiServerPort},
					},
				},
				// Allow intra-namespace communication
				{
					To: []networkingv1.NetworkPolicyPeer{
						{
							PodSelector: &metav1.LabelSelector{},
						},
					},
				},
				// Allow NATS + Storage REST in platform-system
				{
					To: []networkingv1.NetworkPolicyPeer{
						{
							NamespaceSelector: &metav1.LabelSelector{
								MatchLabels: map[string]string{
									"kubernetes.io/metadata.name": platformNamespace,
								},
							},
						},
					},
					Ports: []networkingv1.NetworkPolicyPort{
						{Protocol: &tcp, Port: &natsPort},
						{Protocol: &tcp, Port: &storagePort},
					},
				},
			},
		},
	}
}
