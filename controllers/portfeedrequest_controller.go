/*
Copyright 2023.

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

package controllers

import (
	"context"
	"fmt"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	nmv1alpha1 "github.com/dinoallo/sealos-networkmanager-synchronizer/api/v1alpha1"
	"github.com/dinoallo/sealos-networkmanager-synchronizer/store"
	"github.com/go-logr/logr"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

const (
	PFR_FINALIZER_NAME = "networking.sealos.io/pfr-protection"
)

// PortFeedRequestReconciler reconciles a PortFeedRequest object
type PortFeedRequestReconciler struct {
	client.Client
	Scheme *runtime.Scheme
	Logger logr.Logger
	Store  *store.Store
}

//+kubebuilder:rbac:groups=networking.sealos.io,resources=portfeedrequests,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=networking.sealos.io,resources=portfeedrequests/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=networking.sealos.io,resources=portfeedrequests/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the PortFeedRequest object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.14.1/pkg/reconcile
func (r *PortFeedRequestReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := r.Logger.WithValues("port_feed_request", req.NamespacedName)
	var pfr nmv1alpha1.PortFeedRequest

	if err := r.Get(ctx, req.NamespacedName, &pfr); err != nil {
		log.Info("unable to fetch the pfr for syncing; ignore for now")
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}
	// first, check if the tsr is set up for deletion
	// this tsr is set up for deletion
	if !pfr.DeletionTimestamp.IsZero() {
		// re-synchronize the last time for this request before deletion
		if err := r.syncTraffic(ctx, &pfr); err != nil {
			log.Error(err, "unable to synchronize the port feed the last time before deletion")
			return ctrl.Result{}, err
		}
		// if it's successful, we remove the finalizer
		controllerutil.RemoveFinalizer(&pfr, PFR_FINALIZER_NAME)
		if err := r.Update(ctx, &pfr); err != nil {
			log.Error(err, "unable to remove the finalizer")
			return ctrl.Result{}, err
		}
		return ctrl.Result{}, nil
	}
	// this tsr is not set up for deletion

	// if the tsr doesn't have a finalizer, we add one to prevent there is always a
	// force re-synchronization before the tsr is deleted
	if !controllerutil.ContainsFinalizer(&pfr, PFR_FINALIZER_NAME) {
		controllerutil.AddFinalizer(&pfr, PFR_FINALIZER_NAME)
		if err := r.Update(ctx, &pfr); err != nil {
			log.Error(err, "unable to add the finalizer")
			return ctrl.Result{}, err
		}
	}
	syncPeriod := pfr.Spec.SyncPeriod
	// the time for synchronization has not yet come

	if !r.checkIfSyncRequired(ctx, &pfr) {
		return ctrl.Result{RequeueAfter: syncPeriod.Duration}, nil
	}
	if err := r.syncTraffic(ctx, &pfr); err != nil {
		log.Error(err, "failed to sync traffic")
		return ctrl.Result{}, err
	}
	newPfr := pfr.DeepCopy()
	newPfr.Status.LastSyncTime = metav1.Now()

	if err := r.Status().Update(ctx, newPfr); err != nil {
		log.Error(err, "failed to update the status")
		return ctrl.Result{}, err
	}

	return ctrl.Result{RequeueAfter: syncPeriod.Duration}, nil
}

func (r *PortFeedRequestReconciler) checkIfSyncRequired(ctx context.Context, pfr *nmv1alpha1.PortFeedRequest) bool {
	if pfr.Status.LastSyncTime.IsZero() {
		return true
	}
	lst := pfr.Status.LastSyncTime.Time
	now := metav1.Now().Time
	sp := pfr.Spec.SyncPeriod
	if now.Before(lst.Add(sp.Duration)) {
		return false
	}
	return true
}

func (r *PortFeedRequestReconciler) syncTraffic(ctx context.Context, pfr *nmv1alpha1.PortFeedRequest) error {
	if pfr == nil || r.Store == nil {
		return nil
	}
	_nn := types.NamespacedName{
		Namespace: pfr.Spec.AssociatedNamespace,
		Name:      pfr.Spec.AssociatedPod,
	}
	nn := _nn.String()
	addr := pfr.Spec.Address
	tag := fmt.Sprint(pfr.Spec.Port)

	var pta store.PodTrafficAccount
	var lastSentByteMark uint64 = 0
	var curSentByteMark uint64 = 0
	if found, err := r.Store.FindPTA(ctx, nn, &pta); err != nil {
		return err
	} else if found {
		if err := pta.GetByteMark(addr, tag, 1, true, &lastSentByteMark); err != nil {
			return err
		}
		if err := pta.GetByteMark(addr, tag, 1, false, &curSentByteMark); err != nil {
			return err
		}
	}
	if curSentByteMark < lastSentByteMark {
		// stale byte mark found; not sync this time
		return nil
	}

	sentBytes := curSentByteMark - lastSentByteMark
	req := store.PortFeedProp{
		Namespace: pfr.Spec.AssociatedNamespace,
		Pod:       pfr.Spec.AssociatedPod,
		Addr:      pfr.Spec.Address,
		Port:      pfr.Spec.Port,
	}
	if err := r.Store.IncPFByteField(ctx, req, "sent_bytes", sentBytes); err != nil {
		return err
	}
	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *PortFeedRequestReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&nmv1alpha1.PortFeedRequest{}).
		WithEventFilter(predicate.Funcs{
			CreateFunc: func(ce event.CreateEvent) bool { return true },
			UpdateFunc: func(ue event.UpdateEvent) bool {
				// only reconcile if spec changes
				oldGen := ue.ObjectOld.GetGeneration()
				newGen := ue.ObjectNew.GetGeneration()
				return oldGen != newGen
			},
			DeleteFunc: func(de event.DeleteEvent) bool {
				return true
			},
		}).WithOptions(controller.Options{MaxConcurrentReconciles: 5}).
		Complete(r)
}
