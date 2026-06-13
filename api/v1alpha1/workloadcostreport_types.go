// Package v1alpha1 contains API Schema definitions for the cost-sentinel.io v1alpha1 API group
// +groupName=cost-sentinel.io
package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// WorkloadCostReportSpec defines the desired state of WorkloadCostReport
type WorkloadCostReportSpec struct {
	// Namespace to monitor. Use "*" for all namespaces.
	// +kubebuilder:default="*"
	Namespace string `json:"namespace,omitempty"`

	// PricingModel controls whether to use on-demand or spot pricing.
	// +kubebuilder:validation:Enum=on-demand;spot
	// +kubebuilder:default="on-demand"
	PricingModel string `json:"pricingModel,omitempty"`

	// Currency for cost estimates.
	// +kubebuilder:default="USD"
	Currency string `json:"currency,omitempty"`

	// RefreshInterval controls how often the operator recalculates costs.
	// +kubebuilder:default="1h"
	RefreshInterval string `json:"refreshInterval,omitempty"`
}

// WorkloadCostEntry holds cost data for a single workload
type WorkloadCostEntry struct {
	Name         string `json:"name"`
	Namespace    string `json:"namespace"`
	Kind         string `json:"kind"`
	InstanceType string `json:"instanceType"`
	HourlyCost   string `json:"hourlyCost"`
	MonthlyEstimate string `json:"monthlyEstimate"`
	LastUpdated  metav1.Time `json:"lastUpdated"`
}

// WorkloadCostReportStatus defines the observed state of WorkloadCostReport
type WorkloadCostReportStatus struct {
	// TotalHourlyCost across all monitored workloads
	TotalHourlyCost string `json:"totalHourlyCost,omitempty"`

	// TotalMonthlyCost estimated across all workloads
	TotalMonthlyCost string `json:"totalMonthlyCost,omitempty"`

	// TopWorkloads lists the highest-cost workloads
	// +listType=atomic
	TopWorkloads []WorkloadCostEntry `json:"topWorkloads,omitempty"`

	// LastReconcileTime is the last time the operator updated costs
	LastReconcileTime metav1.Time `json:"lastReconcileTime,omitempty"`

	// Conditions represents the latest available observations
	// +listType=map
	// +listMapKey=type
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Cluster,shortName=wcr
// +kubebuilder:printcolumn:name="Total/hr",type=string,JSONPath=".status.totalHourlyCost"
// +kubebuilder:printcolumn:name="Total/mo",type=string,JSONPath=".status.totalMonthlyCost"
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=".metadata.creationTimestamp"

// WorkloadCostReport is the Schema for the workloadcostreports API
type WorkloadCostReport struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   WorkloadCostReportSpec   `json:"spec,omitempty"`
	Status WorkloadCostReportStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// WorkloadCostReportList contains a list of WorkloadCostReport
type WorkloadCostReportList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []WorkloadCostReport `json:"items"`
}

func init() {
	SchemeBuilder.Register(&WorkloadCostReport{}, &WorkloadCostReportList{})
}
