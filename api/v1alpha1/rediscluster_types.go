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

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// RedisClusterSpec defines the desired state of RedisCluster
type RedisClusterSpec struct {
	// Size is the number of Redis master nodes in the cluster
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=10
	Size int32 `json:"size"`

	// ReplicasPerMaster is the number of replica nodes per master
	// +kubebuilder:validation:Minimum=0
	// +kubebuilder:validation:Maximum=5
	ReplicasPerMaster int32 `json:"replicasPerMaster"`

	// RedisVersion is the version of Redis to deploy
	// +kubebuilder:validation:Enum="6.2.0";"6.2.6";"7.0.0";"7.0.15";"7.2.0";"7.2.4"
	RedisVersion string `json:"redisVersion"`

	// Resources defines the resource requirements for each redis node
	Resources RedisResources `json:"resources,omitempty"`

	// PersistenceEnabled determines if the Redis data should be persistent
	PersistenceEnabled bool `json:"persistenceEnabled,omitempty"`

	// StorageClassName defines the storage class to use for PVCs
	// +optional
	StorageClassName string `json:"storageClassName,omitempty"`

	// StorageSize defines the storage size for each Redis node
	// +optional
	StorageSize string `json:"storageSize,omitempty"`

	// Password is the password for Redis authentication (will be auto-generated if empty)
	// +optional
	Password string `json:"password,omitempty"`
}

// RedisResources defines CPU/Memory resources requests/limits
type RedisResources struct {
	// CPU request for Redis containers
	// +optional
	CPURequest string `json:"cpuRequest,omitempty"`

	// Memory request for Redis containers
	// +optional
	MemoryRequest string `json:"memoryRequest,omitempty"`

	// CPU limit for Redis containers
	// +optional
	CPULimit string `json:"cpuLimit,omitempty"`

	// Memory limit for Redis containers
	// +optional
	MemoryLimit string `json:"memoryLimit,omitempty"`
}

// RedisClusterStatus defines the observed state of RedisCluster
type RedisClusterStatus struct {
	// Nodes contains the current status of each Redis node in the cluster
	Nodes []RedisNodeStatus `json:"nodes,omitempty"`

	// ClusterStatus is the current status of the Redis cluster
	// +kubebuilder:validation:Enum=Creating;Ready;Scaling;Updating;Error
	ClusterStatus string `json:"clusterStatus,omitempty"`

	// LastBackupTime is the timestamp of the last successful backup
	// +optional
	LastBackupTime *metav1.Time `json:"lastBackupTime,omitempty"`
}

// RedisNodeStatus contains status information about an individual Redis node
type RedisNodeStatus struct {
	// PodName is the name of the Pod running this Redis node
	PodName string `json:"podName"`

	// Role is the role of this Redis node (master or replica)
	Role string `json:"role"`

	// MasterNodeID is the ID of the master node (only for replicas)
	// +optional
	MasterNodeID string `json:"masterNodeId,omitempty"`

	// NodeID is the Redis node ID
	NodeID string `json:"nodeId"`

	// Status is the status of the Redis node
	// +kubebuilder:validation:Enum=Pending;Running;Failed
	Status string `json:"status"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:printcolumn:name="Size",type="integer",JSONPath=".spec.size",description="Number of master nodes"
//+kubebuilder:printcolumn:name="Replicas",type="integer",JSONPath=".spec.replicasPerMaster",description="Replicas per master"
//+kubebuilder:printcolumn:name="Status",type="string",JSONPath=".status.clusterStatus",description="Cluster status"
//+kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"

// RedisCluster is the Schema for the redisclusters API
type RedisCluster struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   RedisClusterSpec   `json:"spec,omitempty"`
	Status RedisClusterStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// RedisClusterList contains a list of RedisCluster
type RedisClusterList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []RedisCluster `json:"items"`
}

func init() {
	SchemeBuilder.Register(&RedisCluster{}, &RedisClusterList{})
}
