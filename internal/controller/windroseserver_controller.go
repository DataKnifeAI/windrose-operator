package controller

import (
	"context"
	"fmt"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"
	gatewayv1alpha2 "sigs.k8s.io/gateway-api/apis/v1alpha2"

	egv1a1 "github.com/envoyproxy/gateway/api/v1alpha1"
	windrosev1alpha1 "github.com/DataKnifeAI/windrose-operator/api/v1alpha1"
)

// WindroseServerReconciler reconciles a WindroseServer object.
type WindroseServerReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=windrose.dataknife.ai,resources=windroseservers,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=windrose.dataknife.ai,resources=windroseservers/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=windrose.dataknife.ai,resources=windroseservers/finalizers,verbs=update
// +kubebuilder:rbac:groups="",resources=persistentvolumeclaims,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="",resources=configmaps,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="",resources=services,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=gateway.networking.k8s.io,resources=gateways,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=gateway.networking.k8s.io,resources=tcproutes,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=gateway.networking.k8s.io,resources=udproutes,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=gateway.envoyproxy.io,resources=envoyproxies,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="",resources=events,verbs=create;patch

func (r *WindroseServerReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := logf.FromContext(ctx)

	server := &windrosev1alpha1.WindroseServer{}
	if err := r.Get(ctx, req.NamespacedName, server); err != nil {
		if apierrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	if !controllerutil.ContainsFinalizer(server, finalizerName) {
		controllerutil.AddFinalizer(server, finalizerName)
		if err := r.Update(ctx, server); err != nil {
			return ctrl.Result{}, err
		}
		return ctrl.Result{Requeue: true}, nil
	}

	if !server.DeletionTimestamp.IsZero() {
		controllerutil.RemoveFinalizer(server, finalizerName)
		if err := r.Update(ctx, server); err != nil {
			return ctrl.Result{}, err
		}
		return ctrl.Result{}, nil
	}

	names := deriveNames(server)

	if err := r.reconcilePVC(ctx, server, names.pvcName); err != nil {
		return r.failStatus(ctx, server, err)
	}
	if err := r.reconcileConfigMap(ctx, server, names.configMapName); err != nil {
		return r.failStatus(ctx, server, err)
	}
	if err := r.reconcileDeployment(ctx, server, names); err != nil {
		return r.failStatus(ctx, server, err)
	}
	if err := r.reconcileClusterIPService(ctx, server, names.serviceName, serverLabels(server.Name)); err != nil {
		return r.failStatus(ctx, server, err)
	}
	if err := r.reconcileClusterIPService(ctx, server, names.envoyService, envoyBackendLabels(server.Name)); err != nil {
		return r.failStatus(ctx, server, err)
	}
	if err := r.reconcileEnvoyGateway(ctx, server, names); err != nil {
		return r.failStatus(ctx, server, err)
	}

	deployment := &appsv1.Deployment{}
	if err := r.Get(ctx, types.NamespacedName{Name: names.deploymentName, Namespace: server.Namespace}, deployment); err != nil {
		return r.failStatus(ctx, server, err)
	}

	gateway := &gatewayv1.Gateway{}
	if err := r.Get(ctx, types.NamespacedName{Name: names.gatewayName, Namespace: server.Namespace}, gateway); err != nil {
		return r.failStatus(ctx, server, err)
	}

	ready := deployment.Status.ReadyReplicas > 0
	phase := windrosev1alpha1.PhasePending
	message := "Waiting for game server pod"
	if ready {
		phase = windrosev1alpha1.PhaseRunning
		message = "Game server is running"
	}

	server.Status.Phase = phase
	server.Status.Ready = ready
	server.Status.InviteCode = server.Spec.InviteCode
	server.Status.ConnectionPort = directConnectionPort(server.Spec)
	server.Status.ConnectionAddress = connectionAddressFromGateway(server, gateway)
	server.Status.Message = message
	meta.SetStatusCondition(&server.Status.Conditions, metav1.Condition{
		Type:               "Ready",
		Status:             conditionStatus(ready),
		Reason:             phase,
		Message:            message,
		ObservedGeneration: server.Generation,
		LastTransitionTime: metav1.NewTime(time.Now()),
	})

	if err := r.Status().Update(ctx, server); err != nil {
		log.Error(err, "failed to update status")
		return ctrl.Result{}, err
	}

	if !ready {
		return ctrl.Result{RequeueAfter: 15 * time.Second}, nil
	}
	return ctrl.Result{}, nil
}

func (r *WindroseServerReconciler) reconcilePVC(ctx context.Context, server *windrosev1alpha1.WindroseServer, name string) error {
	pvc := &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: server.Namespace,
		},
	}

	op, err := controllerutil.CreateOrUpdate(ctx, r.Client, pvc, func() error {
		if err := controllerutil.SetControllerReference(server, pvc, r.Scheme); err != nil {
			return err
		}
		pvc.Labels = serverLabels(server.Name)
		if pvc.Spec.AccessModes == nil {
			pvc.Spec.AccessModes = []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce}
		}
		if pvc.Spec.Resources.Requests == nil {
			pvc.Spec.Resources.Requests = corev1.ResourceList{
				corev1.ResourceStorage: resourceQuantity(storageSize(server.Spec)),
			}
		}
		if server.Spec.StorageClassName != "" {
			pvc.Spec.StorageClassName = &server.Spec.StorageClassName
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("reconcile PVC: %w", err)
	}
	logf.FromContext(ctx).V(1).Info("reconciled PVC", "operation", op, "name", name)
	return nil
}

func (r *WindroseServerReconciler) reconcileConfigMap(ctx context.Context, server *windrosev1alpha1.WindroseServer, name string) error {
	content, err := buildServerDescription(server.Spec)
	if err != nil {
		return fmt.Errorf("build ServerDescription.json: %w", err)
	}

	configMap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: server.Namespace,
		},
	}

	op, err := controllerutil.CreateOrUpdate(ctx, r.Client, configMap, func() error {
		if err := controllerutil.SetControllerReference(server, configMap, r.Scheme); err != nil {
			return err
		}
		configMap.Labels = serverLabels(server.Name)
		configMap.Data = map[string]string{
			serverDescriptionConfigKey: string(content),
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("reconcile ConfigMap: %w", err)
	}
	logf.FromContext(ctx).V(1).Info("reconciled ConfigMap", "operation", op, "name", name)
	return nil
}

func (r *WindroseServerReconciler) reconcileDeployment(
	ctx context.Context,
	server *windrosev1alpha1.WindroseServer,
	names derivedNames,
) error {
	port := directConnectionPort(server.Spec)
	replicas := int32(1)
	runAsUser := containerUser
	runAsNonRoot := true

	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      names.deploymentName,
			Namespace: server.Namespace,
		},
	}

	op, err := controllerutil.CreateOrUpdate(ctx, r.Client, deployment, func() error {
		if err := controllerutil.SetControllerReference(server, deployment, r.Scheme); err != nil {
			return err
		}

		deployment.Labels = serverLabels(server.Name)
		deployment.Spec.Replicas = &replicas
		deployment.Spec.Strategy = appsv1.DeploymentStrategy{Type: appsv1.RecreateDeploymentStrategyType}
		deployment.Spec.Selector = &metav1.LabelSelector{
			MatchLabels: serverLabels(server.Name),
		}
		deployment.Spec.Template.ObjectMeta.Labels = serverLabels(server.Name)

		podSpec := corev1.PodSpec{
			SecurityContext: &corev1.PodSecurityContext{
				FSGroup: &runAsUser,
			},
			TerminationGracePeriodSeconds: int64Ptr(30),
			Containers: []corev1.Container{
				{
					Name:            "windrose",
					Image:           serverImage(server.Spec),
					ImagePullPolicy: imagePullPolicy(server.Spec),
					SecurityContext: &corev1.SecurityContext{
						RunAsUser:                &runAsUser,
						RunAsNonRoot:             &runAsNonRoot,
						AllowPrivilegeEscalation: boolPtr(false),
					},
					Ports: []corev1.ContainerPort{
						{Name: "game-tcp", ContainerPort: port, Protocol: corev1.ProtocolTCP},
						{Name: "game-udp", ContainerPort: port, Protocol: corev1.ProtocolUDP},
					},
					VolumeMounts: []corev1.VolumeMount{
						{Name: "saves", MountPath: savedMountPath},
						{
							Name:      "server-description",
							MountPath: serverDescriptionMountPath,
							SubPath:   serverDescriptionConfigKey,
						},
					},
					Resources: defaultResources(server.Spec),
				},
			},
			Volumes: []corev1.Volume{
				{
					Name: "saves",
					VolumeSource: corev1.VolumeSource{
						PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
							ClaimName: names.pvcName,
						},
					},
				},
				{
					Name: "server-description",
					VolumeSource: corev1.VolumeSource{
						ConfigMap: &corev1.ConfigMapVolumeSource{
							LocalObjectReference: corev1.LocalObjectReference{Name: names.configMapName},
						},
					},
				},
			},
		}

		if len(server.Spec.ImagePullSecrets) > 0 {
			podSpec.ImagePullSecrets = server.Spec.ImagePullSecrets
		}
		if len(server.Spec.NodeSelector) > 0 {
			podSpec.NodeSelector = server.Spec.NodeSelector
		}

		deployment.Spec.Template.Spec = podSpec
		return nil
	})
	if err != nil {
		return fmt.Errorf("reconcile Deployment: %w", err)
	}
	logf.FromContext(ctx).V(1).Info("reconciled Deployment", "operation", op, "name", names.deploymentName)
	return nil
}

func (r *WindroseServerReconciler) reconcileClusterIPService(
	ctx context.Context,
	server *windrosev1alpha1.WindroseServer,
	name string,
	labels map[string]string,
) error {
	port := directConnectionPort(server.Spec)

	service := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: server.Namespace,
		},
	}

	op, err := controllerutil.CreateOrUpdate(ctx, r.Client, service, func() error {
		if err := controllerutil.SetControllerReference(server, service, r.Scheme); err != nil {
			return err
		}

		service.Labels = labels
		service.Spec.Type = corev1.ServiceTypeClusterIP
		service.Spec.Selector = serverLabels(server.Name)
		service.Spec.Ports = gameServicePorts(port)
		return nil
	})
	if err != nil {
		return fmt.Errorf("reconcile Service %s: %w", name, err)
	}
	logf.FromContext(ctx).V(1).Info("reconciled Service", "operation", op, "name", name)
	return nil
}

func (r *WindroseServerReconciler) failStatus(ctx context.Context, server *windrosev1alpha1.WindroseServer, err error) (ctrl.Result, error) {
	server.Status.Phase = windrosev1alpha1.PhaseFailed
	server.Status.Ready = false
	server.Status.Message = err.Error()
	meta.SetStatusCondition(&server.Status.Conditions, metav1.Condition{
		Type:               "Ready",
		Status:             metav1.ConditionFalse,
		Reason:             windrosev1alpha1.PhaseFailed,
		Message:            err.Error(),
		ObservedGeneration: server.Generation,
		LastTransitionTime: metav1.NewTime(time.Now()),
	})
	if statusErr := r.Status().Update(ctx, server); statusErr != nil {
		return ctrl.Result{}, statusErr
	}
	return ctrl.Result{}, err
}

func (r *WindroseServerReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&windrosev1alpha1.WindroseServer{}).
		Owns(&appsv1.Deployment{}).
		Owns(&corev1.Service{}).
		Owns(&corev1.PersistentVolumeClaim{}).
		Owns(&corev1.ConfigMap{}).
		Owns(&gatewayv1.Gateway{}).
		Owns(&gatewayv1alpha2.TCPRoute{}).
		Owns(&gatewayv1alpha2.UDPRoute{}).
		Owns(&egv1a1.EnvoyProxy{}).
		Complete(r)
}

func serverLabels(name string) map[string]string {
	return map[string]string{
		"app.kubernetes.io/name":       "windrose-server",
		"app.kubernetes.io/instance":   name,
		"app.kubernetes.io/managed-by": "windrose-operator",
	}
}

func envoyBackendLabels(instanceName string) map[string]string {
	labels := serverLabels(instanceName)
	labels["app.kubernetes.io/component"] = "envoy-backend"
	return labels
}

func conditionStatus(ready bool) metav1.ConditionStatus {
	if ready {
		return metav1.ConditionTrue
	}
	return metav1.ConditionFalse
}

func int64Ptr(value int64) *int64 {
	return &value
}

func boolPtr(value bool) *bool {
	return &value
}
