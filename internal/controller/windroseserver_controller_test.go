package controller

import (
	"context"
	"testing"

	windrosev1alpha1 "github.com/DataKnifeAI/windrose-operator/api/v1alpha1"
	egv1a1 "github.com/envoyproxy/gateway/api/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"
	gatewayv1alpha2 "sigs.k8s.io/gateway-api/apis/v1alpha2"
)

func testScheme(t *testing.T) *runtime.Scheme {
	t.Helper()
	scheme := runtime.NewScheme()
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(windrosev1alpha1.AddToScheme(scheme))
	utilruntime.Must(gatewayv1.Install(scheme))
	utilruntime.Must(gatewayv1alpha2.Install(scheme))
	utilruntime.Must(egv1a1.AddToScheme(scheme))
	return scheme
}

func testWindroseServer() *windrosev1alpha1.WindroseServer {
	return &windrosev1alpha1.WindroseServer{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "windrose-server",
			Namespace: "game-servers",
		},
		Spec: windrosev1alpha1.WindroseServerSpec{
			Gateway: windrosev1alpha1.GatewayConfig{
				Address: "192.168.14.186",
			},
			MaxPlayerCount: 4,
		},
	}
}

func objectExists(t *testing.T, c client.Client, namespace, name string, obj client.Object) {
	t.Helper()
	if err := c.Get(context.Background(), types.NamespacedName{
		Name: name, Namespace: namespace,
	}, obj); err != nil {
		t.Fatalf("Get(%s) error = %v", name, err)
	}
}

func TestReconcileAddsFinalizer(t *testing.T) {
	scheme := testScheme(t)
	server := testWindroseServer()

	reconciler := &WindroseServerReconciler{
		Client: fake.NewClientBuilder().
			WithScheme(scheme).
			WithObjects(server).
			WithStatusSubresource(server).
			Build(),
		Scheme: scheme,
	}

	req := reconcile.Request{NamespacedName: types.NamespacedName{
		Name: server.Name, Namespace: server.Namespace,
	}}

	if _, err := reconciler.Reconcile(context.Background(), req); err != nil {
		t.Fatalf("Reconcile() error = %v", err)
	}

	updated := &windrosev1alpha1.WindroseServer{}
	if err := reconciler.Get(context.Background(), req.NamespacedName, updated); err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if len(updated.Finalizers) != 1 || updated.Finalizers[0] != finalizerName {
		t.Fatalf("finalizers = %v, want [%s]", updated.Finalizers, finalizerName)
	}
}

func TestReconcileCreatesManagedResources(t *testing.T) {
	scheme := testScheme(t)
	server := testWindroseServer()
	server.Finalizers = []string{finalizerName}

	reconciler := &WindroseServerReconciler{
		Client: fake.NewClientBuilder().
			WithScheme(scheme).
			WithObjects(server).
			WithStatusSubresource(server).
			Build(),
		Scheme: scheme,
	}

	req := reconcile.Request{NamespacedName: types.NamespacedName{
		Name: server.Name, Namespace: server.Namespace,
	}}

	if _, err := reconciler.Reconcile(context.Background(), req); err != nil {
		t.Fatalf("Reconcile() error = %v", err)
	}

	names := deriveNames(server)
	ns := server.Namespace

	objectExists(t, reconciler.Client, ns, names.deploymentName, &appsv1.Deployment{})
	objectExists(t, reconciler.Client, ns, names.pvcName, &corev1.PersistentVolumeClaim{})
	objectExists(t, reconciler.Client, ns, names.configMapName, &corev1.ConfigMap{})
	objectExists(t, reconciler.Client, ns, names.serviceName, &corev1.Service{})
	objectExists(t, reconciler.Client, ns, names.envoyService, &corev1.Service{})
	objectExists(t, reconciler.Client, ns, names.gatewayName, &gatewayv1.Gateway{})
	objectExists(t, reconciler.Client, ns, names.envoyProxyName, &egv1a1.EnvoyProxy{})
	objectExists(t, reconciler.Client, ns, names.tcpRouteName, &gatewayv1alpha2.TCPRoute{})
	objectExists(t, reconciler.Client, ns, names.udpRouteName, &gatewayv1alpha2.UDPRoute{})

	deployment := &appsv1.Deployment{}
	objectExists(t, reconciler.Client, ns, names.deploymentName, deployment)

	container := deployment.Spec.Template.Spec.Containers[0]
	if container.Image != defaultServerImage {
		t.Fatalf("image = %q, want %q", container.Image, defaultServerImage)
	}
	if got := container.Resources.Requests.Memory().String(); got != "12Gi" {
		t.Fatalf("memory request = %s, want 12Gi for 4 players", got)
	}

	gateway := &gatewayv1.Gateway{}
	objectExists(t, reconciler.Client, ns, names.gatewayName, gateway)
	if got := gateway.Spec.Addresses[0].Value; got != server.Spec.Gateway.Address {
		t.Fatalf("gateway address = %q, want %q", got, server.Spec.Gateway.Address)
	}

	updated := &windrosev1alpha1.WindroseServer{}
	if err := reconciler.Get(context.Background(), req.NamespacedName, updated); err != nil {
		t.Fatalf("Get status: %v", err)
	}
	if updated.Status.ConnectionAddress != server.Spec.Gateway.Address {
		t.Fatalf("connectionAddress = %q, want %q", updated.Status.ConnectionAddress, server.Spec.Gateway.Address)
	}
	if updated.Status.ConnectionPort != defaultDirectConnectionPort {
		t.Fatalf("connectionPort = %d, want %d", updated.Status.ConnectionPort, defaultDirectConnectionPort)
	}
}

func TestReconcileFailsWithoutGatewayAddress(t *testing.T) {
	scheme := testScheme(t)
	server := testWindroseServer()
	server.Finalizers = []string{finalizerName}
	server.Spec.Gateway.Address = ""

	reconciler := &WindroseServerReconciler{
		Client: fake.NewClientBuilder().
			WithScheme(scheme).
			WithObjects(server).
			WithStatusSubresource(server).
			Build(),
		Scheme: scheme,
	}

	req := reconcile.Request{NamespacedName: types.NamespacedName{
		Name: server.Name, Namespace: server.Namespace,
	}}

	if _, err := reconciler.Reconcile(context.Background(), req); err == nil {
		t.Fatal("Reconcile() expected error for missing gateway address")
	}

	updated := &windrosev1alpha1.WindroseServer{}
	if err := reconciler.Get(context.Background(), req.NamespacedName, updated); err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if updated.Status.Phase != windrosev1alpha1.PhaseFailed {
		t.Fatalf("phase = %q, want %q", updated.Status.Phase, windrosev1alpha1.PhaseFailed)
	}
}

func TestConnectionAddressFromGateway(t *testing.T) {
	server := testWindroseServer()
	if got := connectionAddressFromGateway(server, nil); got != "192.168.14.186" {
		t.Fatalf("fallback address = %q", got)
	}

	gateway := &gatewayv1.Gateway{
		Status: gatewayv1.GatewayStatus{
			Addresses: []gatewayv1.GatewayStatusAddress{{Value: "10.0.0.1"}},
		},
	}
	if got := connectionAddressFromGateway(server, gateway); got != "10.0.0.1" {
		t.Fatalf("status address = %q, want 10.0.0.1", got)
	}
}
