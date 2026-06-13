package controllers

import (
	"context"
	"fmt"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	costsentinelv1alpha1 "github.com/goutham-annem/eks-cost-sentinel/api/v1alpha1"
)

const (
	annotationHourlyCost      = "cost-sentinel.io/hourly-cost"
	annotationMonthlyEstimate = "cost-sentinel.io/monthly-estimate"
	annotationInstanceType    = "cost-sentinel.io/instance-type"
	annotationLastUpdated     = "cost-sentinel.io/last-updated"
	annotationPricingModel    = "cost-sentinel.io/pricing-model"
)

// WorkloadCostReportReconciler reconciles a WorkloadCostReport object
type WorkloadCostReportReconciler struct {
	client.Client
	Scheme       *runtime.Scheme
	PricingClient PricingClient
}

// PricingClient abstracts AWS Pricing API calls (swappable for tests)
type PricingClient interface {
	GetInstanceHourlyPrice(ctx context.Context, instanceType, region, pricingModel string) (float64, error)
}

// +kubebuilder:rbac:groups=cost-sentinel.io,resources=workloadcostreports,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=cost-sentinel.io,resources=workloadcostreports/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=apps,resources=deployments;statefulsets;daemonsets,verbs=get;list;watch;update;patch
// +kubebuilder:rbac:groups=core,resources=nodes,verbs=get;list;watch

func (r *WorkloadCostReportReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	report := &costsentinelv1alpha1.WorkloadCostReport{}
	if err := r.Get(ctx, req.NamespacedName, report); err != nil {
		if errors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	logger.Info("Reconciling WorkloadCostReport", "name", report.Name)

	targetNamespace := report.Spec.Namespace
	if targetNamespace == "*" {
		targetNamespace = ""
	}

	var deployments appsv1.DeploymentList
	if err := r.List(ctx, &deployments, client.InNamespace(targetNamespace)); err != nil {
		return ctrl.Result{}, fmt.Errorf("listing deployments: %w", err)
	}

	var entries []costsentinelv1alpha1.WorkloadCostEntry
	var totalHourly float64

	for i := range deployments.Items {
		dep := &deployments.Items[i]
		entry, cost, err := r.processWorkload(ctx, dep, "Deployment", report.Spec.PricingModel)
		if err != nil {
			logger.Error(err, "Failed to process workload", "name", dep.Name)
			continue
		}
		entries = append(entries, entry)
		totalHourly += cost
	}

	// Sort entries by cost descending, take top 10
	topN := 10
	if len(entries) < topN {
		topN = len(entries)
	}

	report.Status.TotalHourlyCost = fmt.Sprintf("%.4f", totalHourly)
	report.Status.TotalMonthlyCost = fmt.Sprintf("%.2f", totalHourly*730)
	report.Status.TopWorkloads = entries[:topN]
	report.Status.LastReconcileTime = metav1.NewTime(time.Now())

	if err := r.Status().Update(ctx, report); err != nil {
		return ctrl.Result{}, fmt.Errorf("updating status: %w", err)
	}

	refreshInterval, _ := time.ParseDuration(report.Spec.RefreshInterval)
	if refreshInterval == 0 {
		refreshInterval = time.Hour
	}

	return ctrl.Result{RequeueAfter: refreshInterval}, nil
}

func (r *WorkloadCostReportReconciler) processWorkload(
	ctx context.Context,
	obj client.Object,
	kind string,
	pricingModel string,
) (costsentinelv1alpha1.WorkloadCostEntry, float64, error) {
	// In a real implementation:
	// 1. Get node selectors / node affinity from the workload
	// 2. Find matching nodes and their instance types
	// 3. Call PricingClient.GetInstanceHourlyPrice(...)
	// 4. Multiply by replica count
	// 5. Annotate the workload object

	instanceType := "m5.xlarge" // placeholder — derive from node labels
	hourlyCost := 0.192          // placeholder — from Pricing API

	annotations := obj.GetAnnotations()
	if annotations == nil {
		annotations = make(map[string]string)
	}
	annotations[annotationInstanceType] = instanceType
	annotations[annotationHourlyCost] = fmt.Sprintf("%.4f", hourlyCost)
	annotations[annotationMonthlyEstimate] = fmt.Sprintf("%.2f", hourlyCost*730)
	annotations[annotationLastUpdated] = time.Now().UTC().Format(time.RFC3339)
	annotations[annotationPricingModel] = pricingModel
	obj.SetAnnotations(annotations)

	if err := r.Update(ctx, obj); err != nil {
		return costsentinelv1alpha1.WorkloadCostEntry{}, 0, err
	}

	return costsentinelv1alpha1.WorkloadCostEntry{
		Name:            obj.GetName(),
		Namespace:       obj.GetNamespace(),
		Kind:            kind,
		InstanceType:    instanceType,
		HourlyCost:      fmt.Sprintf("%.4f", hourlyCost),
		MonthlyEstimate: fmt.Sprintf("%.2f", hourlyCost*730),
		LastUpdated:     metav1.NewTime(time.Now()),
	}, hourlyCost, nil
}

func (r *WorkloadCostReportReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&costsentinelv1alpha1.WorkloadCostReport{}).
		Owns(&appsv1.Deployment{}).
		Complete(r)
}
