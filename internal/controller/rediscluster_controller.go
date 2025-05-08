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
	"strconv"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	cachev1alpha1 "github.com/scalewithlee/redis-operator/api/v1alpha1"
)

// RedisClusterReconciler reconciles a RedisCluster object
type RedisClusterReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=cache.my.domain,resources=redisclusters,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=cache.my.domain,resources=redisclusters/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=cache.my.domain,resources=redisclusters/finalizers,verbs=update
//+kubebuilder:rbac:groups=apps,resources=statefulsets,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=core,resources=services,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=core,resources=configmaps,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=core,resources=secrets,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=core,resources=pods,verbs=get;list;watch;exec
//+kubebuilder:rbac:groups=core,resources=persistentvolumeclaims,verbs=get;list;watch;create;update;patch;delete

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *RedisClusterReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	// Fetch (Get) the RedisCluster instance
	redisCluster := &cachev1alpha1.RedisCluster{}
	err := r.Get(ctx, req.NamespacedName, redisCluster)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted
			return ctrl.Result{}, nil
		}
		// Error reading the object
		return ctrl.Result{}, err
	}

	// Initialize status if it's a new cluster
	if redisCluster.Status.ClusterStatus == "" {
		redisCluster.Status.ClusterStatus = "Creating"
		err = r.Status().Update(ctx, redisCluster)
		if err != nil {
			log.Error(err, "Failed to update RedisCluster status")
			return ctrl.Result{}, err
		}
	}

	// Check if ConfigMap exists, if not, create it
	configMap := &corev1.ConfigMap{}
	err = r.Get(ctx, types.NamespacedName{Name: redisCluster.Name + "-config", Namespace: redisCluster.Namespace}, configMap)
	if err != nil && errors.IsNotFound(err) {
		// Create ConfigMap for Redis configuration
		cm, err := r.configMapForRedis(redisCluster)
		if err != nil {
			log.Error(err, "Failed to define ConfigMap")
			return ctrl.Result{}, err
		}
		log.Info("Creating a new ConfigMap", "ConfigMap.Namespace", cm.Namespace, "ConfigMap.Name", cm.Name)
		err = r.Create(ctx, cm)
		if err != nil {
			log.Error(err, "Failed to create ConfigMap")
			return ctrl.Result{}, err
		}
	} else if err != nil {
		log.Error(err, "Failed to get ConfigMap")
		return ctrl.Result{}, err
	}

	// Check if Secret exists, if not, create it
	secret := &corev1.Secret{}
	err = r.Get(ctx, types.NamespacedName{Name: redisCluster.Name + "-password", Namespace: redisCluster.Namespace}, secret)
	if err != nil && errors.IsNotFound(err) {
		// Create Secret for Redis password
		sec, err := r.secretForRedis(redisCluster)
		if err != nil {
			log.Error(err, "Failed to define Secret")
			return ctrl.Result{}, err
		}
		log.Info("Creating a new Secret", "Secret.Namespace", sec.Namespace, "Secret.Name", sec.Name)
		err = r.Create(ctx, sec)
		if err != nil {
			log.Error(err, "Failed to create Secret")
			return ctrl.Result{}, err
		}
	} else if err != nil {
		log.Error(err, "Failed to get Secret")
		return ctrl.Result{}, err
	}

	// Check if headless Service exists, if not create it
	headlessSvc := &corev1.Service{}
	err = r.Get(ctx, types.NamespacedName{Name: redisCluster.Name + "-headless", Namespace: redisCluster.Namespace}, headlessSvc)
	if err != nil && errors.IsNotFound(err) {
		// Create headless Service
		svc := r.headlessServiceForRedis(redisCluster)
		log.Info("Creating a new headless Service", "Service.Namespace", svc.Namespace, "Service.Name", svc.Name)
		err = r.Create(ctx, svc)
		if err != nil {
			log.Error(err, "Failed to create headless Service")
			return ctrl.Result{}, err
		}
	} else if err != nil {
		log.Error(err, "Failed to get headless Service")
		return ctrl.Result{}, err
	}

	// Check if client Service exists, if not create it
	clientSvc := &corev1.Service{}
	err = r.Get(ctx, types.NamespacedName{Name: redisCluster.Name, Namespace: redisCluster.Namespace}, clientSvc)
	if err != nil && errors.IsNotFound(err) {
		// Create client Service
		svc := r.clientServiceForRedis(redisCluster)
		log.Info("Creating a new client Service", "Service.Namespace", svc.Namespace, "Service.Name", svc.Name)
		err = r.Create(ctx, svc)
		if err != nil {
			log.Error(err, "Failed to create client Service")
			return ctrl.Result{}, err
		}
	} else if err != nil {
		log.Error(err, "Failed to get client Service")
		return ctrl.Result{}, err
	}

	// Define StatefulSet for Redis cluster
	statefulSet := &appsv1.StatefulSet{}
	err = r.Get(ctx, types.NamespacedName{Name: redisCluster.Name, Namespace: redisCluster.Namespace}, statefulSet)
	if err != nil && errors.IsNotFound(err) {
		// Create StatefulSet
		sts, err := r.statefulSetForRedis(redisCluster)
		if err != nil {
			log.Error(err, "Failed to define StatefulSet")
			return ctrl.Result{}, err
		}
		log.Info("Creating a new StatefulSet", "StatefulSet.Namespace", sts.Namespace, "StatefulSet.Name", sts.Name)
		err = r.Create(ctx, sts)
		if err != nil {
			log.Error(err, "Failed to create StatefulSet")
			return ctrl.Result{}, err
		}
		// StatefulSet created, update status and return
		redisCluster.Status.ClusterStatus = "Creating"
		err = r.Status().Update(ctx, redisCluster)
		if err != nil {
			log.Error(err, "Failed to update RedisCluster status")
			return ctrl.Result{}, err
		}
		return ctrl.Result{RequeueAfter: time.Second * 10}, nil
	} else if err != nil {
		log.Error(err, "Failed to get StatefulSet")
		return ctrl.Result{}, err
	}

	// StatefulSet exists, check if it needs to be updated
	if statefulSet.Spec.Replicas != nil && *statefulSet.Spec.Replicas != totalRedisNodes(redisCluster) {
		log.Info("Updating StatefulSet", "Old replicas", *statefulSet.Spec.Replicas, "New replicas", totalRedisNodes(redisCluster))
		// Update StatefulSet with new size
		statefulSet.Spec.Replicas = &[]int32{totalRedisNodes(redisCluster)}[0]
		err = r.Update(ctx, statefulSet)
		if err != nil {
			log.Error(err, "Failed to update StatefulSet")
			return ctrl.Result{}, err
		}
		// StatefulSet updated, update status and return
		redisCluster.Status.ClusterStatus = "Scaling"
		err = r.Status().Update(ctx, redisCluster)
		if err != nil {
			log.Error(err, "Failed to update RedisCluster status")
			return ctrl.Result{}, err
		}
		return ctrl.Result{RequeueAfter: time.Second * 10}, nil
	}

	// Check if all pods are running and ready
	// If statefulset is still deploying, requeue
	if statefulSet.Status.ReadyReplicas < totalRedisNodes(redisCluster) {
		log.Info("Waiting for StatefulSet to be ready", "ReadyReplicas", statefulSet.Status.ReadyReplicas, "DesiredReplicas", totalRedisNodes(redisCluster))
		return ctrl.Result{RequeueAfter: time.Second * 10}, nil
	}

	// All pods are ready, update node status
	podList := &corev1.PodList{}
	labelSelector := client.MatchingLabels{"app": redisCluster.Name}
	err = r.List(ctx, podList, client.InNamespace(redisCluster.Namespace), labelSelector)
	if err != nil {
		log.Error(err, "Failed to list pods")
		return ctrl.Result{}, err
	}

	// Build node status entries
	var nodeStatuses []cachev1alpha1.RedisNodeStatus
	for _, pod := range podList.Items {
		if pod.Status.Phase == corev1.PodRunning {
			ordinal, err := getStatefulSetPodOrdinal(pod.Name)
			if err != nil {
				log.Error(err, "Failed to get pod ordinal", "Pod", pod.Name)
				continue
			}

			role := "master"
			var masterNodeID string
			if int32(ordinal) >= redisCluster.Spec.Size {
				role = "replica"
				// Calculate which master this replica belongs to
				masterIndex := (ordinal - int(redisCluster.Spec.Size)) % int(redisCluster.Spec.Size)
				masterNodeID = fmt.Sprintf("node-%d", masterIndex)
			}

			nodeStatuses = append(nodeStatuses, cachev1alpha1.RedisNodeStatus{
				PodName:      pod.Name,
				Role:         role,
				MasterNodeID: masterNodeID,
				NodeID:       fmt.Sprintf("node-%d", ordinal),
				Status:       "Running",
			})
		}
	}

	// Update status
	redisCluster.Status.Nodes = nodeStatuses
	redisCluster.Status.ClusterStatus = "Ready"
	err = r.Status().Update(ctx, redisCluster)
	if err != nil {
		log.Error(err, "Failed to update RedisCluster status")
		return ctrl.Result{}, err
	}
	return ctrl.Result{RequeueAfter: time.Minute}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *RedisClusterReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&cachev1alpha1.RedisCluster{}).
		Owns(&appsv1.StatefulSet{}).
		Owns(&corev1.Service{}).
		Owns(&corev1.ConfigMap{}).
		Owns(&corev1.Secret{}).
		Complete(r)
}

// Utility function to calculate total Redis nodes (masters + replicas)
func totalRedisNodes(cluster *cachev1alpha1.RedisCluster) int32 {
	return cluster.Spec.Size + (cluster.Spec.Size * cluster.Spec.ReplicasPerMaster)
}

// Extract the ordinal from a StatefulSet pod name
func getStatefulSetPodOrdinal(podName string) (int, error) {
	// <statefulset-name>-<ordinal>
	// Extract the ordinal from the end of the pod name
	for i := len(podName) - 1; i >= 0; i-- {
		if podName[i] == '-' {
			return strconv.Atoi(podName[i+1:])
		}
	}
	return 0, fmt.Errorf("pod name %s does not follow StatefulSet naming convention", podName)
}

// Define the ConfigMap for Redis
func (r *RedisClusterReconciler) configMapForRedis(cluster *cachev1alpha1.RedisCluster) (*corev1.ConfigMap, error) {
	redisConfig := `
# Redis configuration
port 6379
cluster-enabled yes
cluster-config-file /data/nodes.conf
cluster-node-timeout 5000
appendonly yes
protected-mode node
`
	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      cluster.Name + "-config",
			Namespace: cluster.Namespace,
		},
		Data: map[string]string{
			"redis.conf": redisConfig,
		},
	}

	// Set owner reference
	err := ctrl.SetControllerReference(cluster, cm, r.Scheme)
	if err != nil {
		return nil, err
	}

	return cm, nil
}

// Define the Secret for Redis password
func (r *RedisClusterReconciler) secretForRedis(cluster *cachev1alpha1.RedisCluster) (*corev1.Secret, error) {
	password := cluster.Spec.Password
	if password == "" {
		password = "changeme" // This is just an example
	}

	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      cluster.Name + "-password",
			Namespace: cluster.Namespace,
		},
		Type: corev1.SecretTypeOpaque,
		StringData: map[string]string{
			"password": password,
		},
	}

	// Set owner reference
	err := ctrl.SetControllerReference(cluster, secret, r.Scheme)
	if err != nil {
		return nil, err
	}

	return secret, nil
}

// Define headless Service for Redis cluster communication
func (r *RedisClusterReconciler) headlessServiceForRedis(cluster *cachev1alpha1.RedisCluster) *corev1.Service {
	service := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      cluster.Name + "-headless",
			Namespace: cluster.Namespace,
			Labels: map[string]string{
				"app":  cluster.Name,
				"role": "headless",
			},
		},
		Spec: corev1.ServiceSpec{
			ClusterIP: "None", // Headless service
			Ports: []corev1.ServicePort{
				{
					Name:       "redis",
					Port:       6379,
					TargetPort: intstr.FromInt(6379),
				},
				{
					Name:       "cluster",
					Port:       16379,
					TargetPort: intstr.FromInt(16379),
				},
			},
			Selector: map[string]string{
				"app": cluster.Name,
			},
		},
	}

	// Set owner reference
	ctrl.SetControllerReference(cluster, service, r.Scheme)
	return service
}

// Define client Service for Redis clients
func (r *RedisClusterReconciler) clientServiceForRedis(cluster *cachev1alpha1.RedisCluster) *corev1.Service {
	service := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      cluster.Name,
			Namespace: cluster.Namespace,
			Labels: map[string]string{
				"app":  cluster.Name,
				"role": "client",
			},
		},
		Spec: corev1.ServiceSpec{
			Type: corev1.ServiceTypeClusterIP,
			Ports: []corev1.ServicePort{
				{
					Name:       "redis",
					Port:       6379,
					TargetPort: intstr.FromInt(6379),
				},
			},
			Selector: map[string]string{
				"app": cluster.Name,
			},
		},
	}

	// Set owner reference
	ctrl.SetControllerReference(cluster, service, r.Scheme)
	return service
}

// Define StatefulSet for Redis cluster
func (r *RedisClusterReconciler) statefulSetForRedis(cluster *cachev1alpha1.RedisCluster) (*appsv1.StatefulSet, error) {
	// Calculate resources
	cpuRequest := resource.MustParse("100m")     // Default CPU request
	memoryRequest := resource.MustParse("128Mi") // Default memory request
	cpuLimit := resource.MustParse("200m")       // Default CPU limit
	memoryLimit := resource.MustParse("256Mi")   // Default memory limit

	if cluster.Spec.Resources.CPURequest != "" {
		cpuRequest = resource.MustParse(cluster.Spec.Resources.CPURequest)
	}
	if cluster.Spec.Resources.MemoryRequest != "" {
		memoryRequest = resource.MustParse(cluster.Spec.Resources.MemoryRequest)
	}
	if cluster.Spec.Resources.CPULimit != "" {
		cpuLimit = resource.MustParse(cluster.Spec.Resources.CPULimit)
	}
	if cluster.Spec.Resources.MemoryLimit != "" {
		memoryLimit = resource.MustParse(cluster.Spec.Resources.MemoryLimit)
	}

	// Default values for Redis version and storage
	redisVersion := "7.2.4"
	storageSize := "1Gi"

	if cluster.Spec.RedisVersion != "" {
		redisVersion = cluster.Spec.RedisVersion
	}
	if cluster.Spec.StorageSize != "" {
		storageSize = cluster.Spec.StorageSize
	}

	// Total redis nodes = masters + replicas
	replicas := totalRedisNodes(cluster)

	// Define the StatefulSet
	sts := &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      cluster.Name,
			Namespace: cluster.Namespace,
		},
		Spec: appsv1.StatefulSetSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": cluster.Name,
				},
			},
			ServiceName: cluster.Name + "-headless",
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app": cluster.Name,
					},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{{
						Name:  "redis",
						Image: "redis:" + redisVersion,
						Ports: []corev1.ContainerPort{
							{
								Name:          "redis",
								ContainerPort: 6379,
							},
							{
								Name:          "cluster",
								ContainerPort: 16379,
							},
						},
						Command: []string{
							"redis-server",
							"/etc/redis/redis.conf",
						},
						Resources: corev1.ResourceRequirements{
							Requests: corev1.ResourceList{
								corev1.ResourceCPU:    cpuRequest,
								corev1.ResourceMemory: memoryRequest,
							},
							Limits: corev1.ResourceList{
								corev1.ResourceCPU:    cpuLimit,
								corev1.ResourceMemory: memoryLimit,
							},
						},
						VolumeMounts: []corev1.VolumeMount{
							{
								Name:      "config",
								MountPath: "/etc/redis",
							},
							{
								Name:      "data",
								MountPath: "/data",
							},
						},
						Env: []corev1.EnvVar{
							{
								Name: "POD_IP",
								ValueFrom: &corev1.EnvVarSource{
									FieldRef: &corev1.ObjectFieldSelector{
										FieldPath: "status.podIP",
									},
								},
							},
							{
								Name: "REDIS_PASSWORD",
								ValueFrom: &corev1.EnvVarSource{
									SecretKeyRef: &corev1.SecretKeySelector{
										LocalObjectReference: corev1.LocalObjectReference{
											Name: cluster.Name + "-password",
										},
										Key: "password",
									},
								},
							},
						},
						LivenessProbe: &corev1.Probe{
							ProbeHandler: corev1.ProbeHandler{
								Exec: &corev1.ExecAction{
									Command: []string{
										"sh",
										"-c",
										"redis-cli -h $(hostname) ping",
									},
								},
							},
							InitialDelaySeconds: 30,
							TimeoutSeconds:      5,
							PeriodSeconds:       10,
						},
						ReadinessProbe: &corev1.Probe{
							ProbeHandler: corev1.ProbeHandler{
								Exec: &corev1.ExecAction{
									Command: []string{
										"sh",
										"-c",
										"redis-cli -h $(hostname) ping",
									},
								},
							},
							InitialDelaySeconds: 5,
							TimeoutSeconds:      5,
							PeriodSeconds:       10,
						},
					}},
					Volumes: []corev1.Volume{
						{
							Name: "config",
							VolumeSource: corev1.VolumeSource{
								ConfigMap: &corev1.ConfigMapVolumeSource{
									LocalObjectReference: corev1.LocalObjectReference{
										Name: cluster.Name + "-config",
									},
								},
							},
						},
					},
				},
			},
		},
	}
	// Add persistent volume claim template if persistence is enabled
	if cluster.Spec.PersistenceEnabled {
		storageClass := cluster.Spec.StorageClassName
		sts.Spec.VolumeClaimTemplates = []corev1.PersistentVolumeClaim{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name: "data",
				},
				Spec: corev1.PersistentVolumeClaimSpec{
					AccessModes: []corev1.PersistentVolumeAccessMode{
						corev1.ReadWriteOnce,
					},
					Resources: corev1.ResourceRequirements{
						Requests: corev1.ResourceList{
							corev1.ResourceStorage: resource.MustParse(storageSize),
						},
					},
					StorageClassName: &storageClass,
				},
			},
		}
	} else {
		// If persistence is not enabled, use emptyDir
		sts.Spec.Template.Spec.Volumes = append(sts.Spec.Template.Spec.Volumes, corev1.Volume{
			Name: "data",
			VolumeSource: corev1.VolumeSource{
				EmptyDir: &corev1.EmptyDirVolumeSource{},
			},
		})
	}

	// Set owner reference
	err := ctrl.SetControllerReference(cluster, sts, r.Scheme)
	if err != nil {
		return nil, err
	}

	return sts, nil
}
