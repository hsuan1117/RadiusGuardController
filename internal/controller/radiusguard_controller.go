/*
Copyright 2025.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controller

import (
	"context"
	"fmt"

	v1 "k8s.io/api/apps/v1"
	v2 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	networkingv1 "github.com/hsuan1117/RadiusGuardController/api/v1"
)

// RADIUSGuardReconciler reconciles a RADIUSGuard object
type RADIUSGuardReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=networking.hsuan.app,resources=radiusguards,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=networking.hsuan.app,resources=radiusguards/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=networking.hsuan.app,resources=radiusguards/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the RADIUSGuard object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.21.0/pkg/reconcile
func (r *RADIUSGuardReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = logf.FromContext(ctx)

	var radiusGuard networkingv1.RADIUSGuard
	if err := r.Get(ctx, req.NamespacedName, &radiusGuard); err != nil {
		// Handle not found error (object deleted)
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	configMap := r.ConstructConfigMap(&radiusGuard)
	// if backend service and secret specified, add proxy.conf using provided secret
	if radiusGuard.Spec.BackendServiceName != "" && radiusGuard.Spec.BackendSecret != "" {
		var backendSvc v2.Service
		if err := r.Get(ctx, types.NamespacedName{Name: radiusGuard.Spec.BackendServiceName, Namespace: radiusGuard.Namespace}, &backendSvc); err != nil {
			return ctrl.Result{}, err
		}
		secretValue := radiusGuard.Spec.BackendSecret
		host := fmt.Sprintf("%s.%s.svc.cluster.local", radiusGuard.Spec.BackendServiceName, radiusGuard.Namespace)
		authPort := radiusGuard.Spec.AuthPort
		if authPort == 0 {
			authPort = 1812
		}
		acctPort := radiusGuard.Spec.AcctPort
		if acctPort == 0 {
			acctPort = 1813
		}
		proxyConf := fmt.Sprintf(`realm DEFAULT {
    type = radius
    secret = %s
    authhost = %s:%d
    accthost = %s:%d

    nostrip
}

`, secretValue, host, authPort, host, acctPort)
		configMap.Data["proxy.conf"] = proxyConf
	}

	if err := controllerutil.SetControllerReference(&radiusGuard, configMap, r.Scheme); err != nil {
		return ctrl.Result{}, err
	}

	foundConfigMap := &v2.ConfigMap{}
	err := r.Get(ctx, types.NamespacedName{Name: configMap.Name, Namespace: configMap.Namespace}, foundConfigMap)
	if err != nil && errors.IsNotFound(err) {
		ctrl.Log.Info("Creating a new ConfigMap", "ConfigMap.Namespace", configMap.Namespace, "ConfigMap.Name", configMap.Name)
		err = r.Create(ctx, configMap)
		if err != nil {
			return ctrl.Result{}, err
		}
	} else if err != nil {
		return ctrl.Result{}, err
	} else {
		foundConfigMap.Data = configMap.Data
		err = r.Update(ctx, foundConfigMap)
		if err != nil {
			return ctrl.Result{}, err
		}
	}

	daemonSet := r.ConstructDaemonSet(&radiusGuard)
	if err := controllerutil.SetControllerReference(&radiusGuard, daemonSet, r.Scheme); err != nil {
		return ctrl.Result{}, err
	}

	foundDaemonSet := &v1.DaemonSet{}
	err = r.Get(ctx, types.NamespacedName{Name: daemonSet.Name, Namespace: daemonSet.Namespace}, foundDaemonSet)
	if err != nil && errors.IsNotFound(err) {
		ctrl.Log.Info("Creating a new DaemonSet", "DaemonSet.Namespace", daemonSet.Namespace, "DaemonSet.Name", daemonSet.Name)
		err = r.Create(ctx, daemonSet)
		if err != nil {
			return ctrl.Result{}, err
		}
		return ctrl.Result{Requeue: true}, nil
	} else if err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (r *RADIUSGuardReconciler) ConstructConfigMap(radiusGuard *networkingv1.RADIUSGuard) *v2.ConfigMap {
	clientsConf := ""
	for _, client := range radiusGuard.Spec.Clients {
		clientsConf += fmt.Sprintf(`client %s {
    ipaddr = %s
    secret = %s
}

`, client.Name, client.IPAddress, client.Secret)
	}

	return &v2.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      radiusGuard.Name + "-config",
			Namespace: radiusGuard.Namespace,
			Labels: map[string]string{
				"app":         "radiusguard",
				"radiusguard": radiusGuard.Name,
			},
		},
		Data: map[string]string{
			"clients.conf": clientsConf,
		},
	}
}

func (r *RADIUSGuardReconciler) ConstructDaemonSet(radiusGuard *networkingv1.RADIUSGuard) *v1.DaemonSet {
	authPort := radiusGuard.Spec.AuthPort
	if authPort == 0 {
		authPort = 1812
	}
	acctPort := radiusGuard.Spec.AcctPort
	if acctPort == 0 {
		acctPort = 1813
	}

	labels := map[string]string{
		"app":         "radiusguard",
		"radiusguard": radiusGuard.Name,
	}

	// prepare volume mounts conditionally including proxy.conf
	mounts := []v2.VolumeMount{
		{Name: "clients-config", MountPath: "/etc/raddb/clients.conf", SubPath: "clients.conf"},
	}
	if radiusGuard.Spec.BackendServiceName != "" && radiusGuard.Spec.BackendSecret != "" {
		mounts = append(mounts, v2.VolumeMount{Name: "clients-config", MountPath: "/etc/raddb/proxy.conf", SubPath: "proxy.conf"})
	}
	return &v1.DaemonSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      radiusGuard.Name + "-guard",
			Namespace: radiusGuard.Namespace,
			Labels:    labels,
		},
		Spec: v1.DaemonSetSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: labels,
			},
			Template: v2.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: labels,
				},
				Spec: v2.PodSpec{
					DNSPolicy: v2.DNSClusterFirstWithHostNet,
					SecurityContext: &v2.PodSecurityContext{
						RunAsUser: new(int64),
					},
					HostNetwork: true,
					Containers: []v2.Container{
						{
							Name:            "freeradius",
							Image:           "docker.io/freeradius/freeradius-server:latest",
							ImagePullPolicy: v2.PullAlways,
							Command: []string{
								"freeradius",
								"-X",
							},
							Ports: []v2.ContainerPort{
								{
									Name:          "radius",
									ContainerPort: 1812,
									HostPort:      int32(authPort),
									Protocol:      v2.ProtocolUDP,
								},
								{
									Name:          "radius-acct",
									ContainerPort: 1813,
									HostPort:      int32(acctPort),
									Protocol:      v2.ProtocolUDP,
								},
							},
							TTY:          true,
							VolumeMounts: mounts,
						},
					},
					Volumes: []v2.Volume{
						{
							Name: "clients-config",
							VolumeSource: v2.VolumeSource{
								ConfigMap: &v2.ConfigMapVolumeSource{
									LocalObjectReference: v2.LocalObjectReference{
										Name: radiusGuard.Name + "-config",
									},
								},
							},
						},
					},
				},
			},
		},
	}
}

// SetupWithManager sets up the controller with the Manager.
func (r *RADIUSGuardReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&networkingv1.RADIUSGuard{}).
		Named("radiusguard").
		Complete(r)
}
