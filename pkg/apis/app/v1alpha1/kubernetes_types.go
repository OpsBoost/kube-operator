package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// KubernetesSpec defines the desired state of Kubernetes
type KubernetesSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "operator-sdk generate k8s" to regenerate code after modifying this file
	// Add custom validation using kubebuilder tags: https://book-v1.book.kubebuilder.io/beyond_basics/generating_crd.html
	Version              string `json:"version,omitempty"`
	EtcdNamespace        string `json:"etcdNamespace,omitempty"`
	EtcdServiceNamespace string `json:"etcdServiceNamespace,omitempty"`
	EtcdService          string `json:"etcdService"`
	EtcdClientSecretRef  string `json:"etcdClientSecretRef,omitempty"`
}

// KubernetesStatus defines the observed state of Kubernetes
type KubernetesStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "operator-sdk generate k8s" to regenerate code after modifying this file
	// Add custom validation using kubebuilder tags: https://book-v1.book.kubebuilder.io/beyond_basics/generating_crd.html
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// Kubernetes is the Schema for the kubernetes API
// +kubebuilder:subresource:status
// +kubebuilder:resource:path=kubernetes,scope=Namespaced
type Kubernetes struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   KubernetesSpec   `json:"spec,omitempty"`
	Status KubernetesStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// KubernetesList contains a list of Kubernetes
type KubernetesList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Kubernetes `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Kubernetes{}, &KubernetesList{})
}
