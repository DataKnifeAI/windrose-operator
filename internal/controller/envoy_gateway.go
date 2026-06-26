package controller

import (
	"context"
	"fmt"

	egv1a1 "github.com/envoyproxy/gateway/api/v1alpha1"
	windrosev1alpha1 "github.com/DataKnifeAI/windrose-operator/api/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"
	gatewayv1alpha2 "sigs.k8s.io/gateway-api/apis/v1alpha2"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

func (r *WindroseServerReconciler) reconcileEnvoyGateway(
	ctx context.Context,
	server *windrosev1alpha1.WindroseServer,
	names derivedNames,
) error {
	if server.Spec.Gateway.Address == "" {
		return fmt.Errorf("spec.gateway.address is required")
	}

	if err := r.reconcileEnvoyProxy(ctx, server, names); err != nil {
		return err
	}
	if err := r.reconcileGateway(ctx, server, names); err != nil {
		return err
	}
	if err := r.reconcileTCPRoute(ctx, server, names); err != nil {
		return err
	}
	if err := r.reconcileUDPRoute(ctx, server, names); err != nil {
		return err
	}
	return nil
}

func (r *WindroseServerReconciler) reconcileEnvoyProxy(
	ctx context.Context,
	server *windrosev1alpha1.WindroseServer,
	names derivedNames,
) error {
	envoyProxy := &egv1a1.EnvoyProxy{
		ObjectMeta: metav1.ObjectMeta{
			Name:      names.envoyProxyName,
			Namespace: server.Namespace,
		},
	}

	op, err := controllerutil.CreateOrUpdate(ctx, r.Client, envoyProxy, func() error {
		if err := controllerutil.SetControllerReference(server, envoyProxy, r.Scheme); err != nil {
			return err
		}
		envoyProxy.Labels = gatewayLabels(server.Name)
		policy := envoyExternalTrafficPolicy(server.Spec)
		envoyProxy.Spec = egv1a1.EnvoyProxySpec{
			Logging: egv1a1.ProxyLogging{
				Level: map[egv1a1.ProxyLogComponent]egv1a1.LogLevel{
					egv1a1.LogComponentDefault: egv1a1.LogLevelWarn,
				},
			},
			Provider: &egv1a1.EnvoyProxyProvider{
				Type: egv1a1.EnvoyProxyProviderTypeKubernetes,
				Kubernetes: &egv1a1.EnvoyProxyKubernetesProvider{
					EnvoyService: &egv1a1.KubernetesServiceSpec{
						Type:                  ptr.To(egv1a1.ServiceTypeLoadBalancer),
						LoadBalancerIP:        ptr.To(server.Spec.Gateway.Address),
						ExternalTrafficPolicy: ptr.To(policy),
					},
				},
			},
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("reconcile EnvoyProxy: %w", err)
	}
	logf.FromContext(ctx).V(1).Info("reconciled EnvoyProxy", "operation", op, "name", names.envoyProxyName)
	return nil
}

func (r *WindroseServerReconciler) reconcileGateway(
	ctx context.Context,
	server *windrosev1alpha1.WindroseServer,
	names derivedNames,
) error {
	port := directConnectionPort(server.Spec)
	gateway := &gatewayv1.Gateway{
		ObjectMeta: metav1.ObjectMeta{
			Name:      names.gatewayName,
			Namespace: server.Namespace,
		},
	}

	op, err := controllerutil.CreateOrUpdate(ctx, r.Client, gateway, func() error {
		if err := controllerutil.SetControllerReference(server, gateway, r.Scheme); err != nil {
			return err
		}
		gateway.Labels = gatewayLabels(server.Name)
		gateway.Spec = gatewayv1.GatewaySpec{
			GatewayClassName: gatewayv1.ObjectName(gatewayClassName(server.Spec)),
			Addresses: []gatewayv1.GatewaySpecAddress{
				{
					Type:  ptr.To(gatewayv1.IPAddressType),
					Value: server.Spec.Gateway.Address,
				},
			},
			Infrastructure: &gatewayv1.GatewayInfrastructure{
				ParametersRef: &gatewayv1.LocalParametersReference{
					Group: gatewayv1.Group(egv1a1.GroupVersion.Group),
					Kind:  gatewayv1.Kind("EnvoyProxy"),
					Name:  names.envoyProxyName,
				},
			},
			Listeners: []gatewayv1.Listener{
				{
					Name:     gatewayv1.SectionName(gatewayListenerGameTCP),
					Port:     gatewayv1.PortNumber(port),
					Protocol: gatewayv1.TCPProtocolType,
					AllowedRoutes: &gatewayv1.AllowedRoutes{
						Namespaces: &gatewayv1.RouteNamespaces{
							From: ptr.To(gatewayv1.NamespacesFromSame),
						},
					},
				},
				{
					Name:     gatewayv1.SectionName(gatewayListenerGameUDP),
					Port:     gatewayv1.PortNumber(port),
					Protocol: gatewayv1.UDPProtocolType,
					AllowedRoutes: &gatewayv1.AllowedRoutes{
						Namespaces: &gatewayv1.RouteNamespaces{
							From: ptr.To(gatewayv1.NamespacesFromSame),
						},
					},
				},
			},
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("reconcile Gateway: %w", err)
	}
	logf.FromContext(ctx).V(1).Info("reconciled Gateway", "operation", op, "name", names.gatewayName)
	return nil
}

func (r *WindroseServerReconciler) reconcileTCPRoute(
	ctx context.Context,
	server *windrosev1alpha1.WindroseServer,
	names derivedNames,
) error {
	port := directConnectionPort(server.Spec)
	tcpRoute := &gatewayv1alpha2.TCPRoute{
		ObjectMeta: metav1.ObjectMeta{
			Name:      names.tcpRouteName,
			Namespace: server.Namespace,
		},
	}

	op, err := controllerutil.CreateOrUpdate(ctx, r.Client, tcpRoute, func() error {
		if err := controllerutil.SetControllerReference(server, tcpRoute, r.Scheme); err != nil {
			return err
		}
		tcpRoute.Labels = gatewayLabels(server.Name)
		tcpRoute.Spec = gatewayv1alpha2.TCPRouteSpec{
			CommonRouteSpec: gatewayv1.CommonRouteSpec{
				ParentRefs: []gatewayv1.ParentReference{
					{
						Name:        gatewayv1.ObjectName(names.gatewayName),
						Namespace:   ptr.To(gatewayv1.Namespace(server.Namespace)),
						SectionName: ptr.To(gatewayv1.SectionName(gatewayListenerGameTCP)),
					},
				},
			},
			Rules: []gatewayv1alpha2.TCPRouteRule{
				{
					BackendRefs: []gatewayv1alpha2.BackendRef{
						{
							BackendObjectReference: gatewayv1.BackendObjectReference{
								Name: gatewayv1.ObjectName(names.envoyService),
								Port: ptr.To(gatewayv1.PortNumber(port)),
							},
						},
					},
				},
			},
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("reconcile TCPRoute: %w", err)
	}
	logf.FromContext(ctx).V(1).Info("reconciled TCPRoute", "operation", op, "name", names.tcpRouteName)
	return nil
}

func (r *WindroseServerReconciler) reconcileUDPRoute(
	ctx context.Context,
	server *windrosev1alpha1.WindroseServer,
	names derivedNames,
) error {
	port := directConnectionPort(server.Spec)
	udpRoute := &gatewayv1alpha2.UDPRoute{
		ObjectMeta: metav1.ObjectMeta{
			Name:      names.udpRouteName,
			Namespace: server.Namespace,
		},
	}

	op, err := controllerutil.CreateOrUpdate(ctx, r.Client, udpRoute, func() error {
		if err := controllerutil.SetControllerReference(server, udpRoute, r.Scheme); err != nil {
			return err
		}
		udpRoute.Labels = gatewayLabels(server.Name)
		udpRoute.Spec = gatewayv1alpha2.UDPRouteSpec{
			CommonRouteSpec: gatewayv1.CommonRouteSpec{
				ParentRefs: []gatewayv1.ParentReference{
					{
						Name:        gatewayv1.ObjectName(names.gatewayName),
						Namespace:   ptr.To(gatewayv1.Namespace(server.Namespace)),
						SectionName: ptr.To(gatewayv1.SectionName(gatewayListenerGameUDP)),
					},
				},
			},
			Rules: []gatewayv1alpha2.UDPRouteRule{
				{
					BackendRefs: []gatewayv1alpha2.BackendRef{
						{
							BackendObjectReference: gatewayv1.BackendObjectReference{
								Name: gatewayv1.ObjectName(names.envoyService),
								Port: ptr.To(gatewayv1.PortNumber(port)),
							},
						},
					},
				},
			},
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("reconcile UDPRoute: %w", err)
	}
	logf.FromContext(ctx).V(1).Info("reconciled UDPRoute", "operation", op, "name", names.udpRouteName)
	return nil
}

func gatewayLabels(instanceName string) map[string]string {
	labels := serverLabels(instanceName)
	labels["app.kubernetes.io/component"] = "envoy-gateway"
	return labels
}

func connectionAddressFromGateway(server *windrosev1alpha1.WindroseServer, gateway *gatewayv1.Gateway) string {
	if gateway != nil && len(gateway.Status.Addresses) > 0 {
		return gateway.Status.Addresses[0].Value
	}
	return server.Spec.Gateway.Address
}
