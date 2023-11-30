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

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	nmv1alpha1 "github.com/dinoallo/sealos-networkmanager-synchronizer/api/v1alpha1"
	nmaclient "github.com/dinoallo/sealos-networkmanager-synchronizer/client"
	"github.com/dinoallo/sealos-networkmanager-synchronizer/store"
	"github.com/go-logr/logr"
)

const (
	AGENT_PORT         = "50051"
	TSR_FINALIZER_NAME = "networking.sealos.io/tsr-protection"
)

// TrafficSyncRequestReconciler reconciles a TrafficSyncRequest object
type TrafficSyncRequestReconciler struct {
	client.Client
	Scheme *runtime.Scheme
	Logger logr.Logger
	Store  *store.Store
}

// +kubebuilder:rbac:groups=networking.sealos.io,resources=trafficsyncrequests,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=networking.sealos.io,resources=trafficsyncrequests/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=networking.sealos.io,resources=trafficsyncrequests/finalizers,verbs=update

func (r *TrafficSyncRequestReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := r.Logger.WithValues("traffic_sync_request", req.NamespacedName)
	var tsr nmv1alpha1.TrafficSyncRequest
	if err := r.Get(ctx, req.NamespacedName, &tsr); err != nil {
		log.Info("unable to fetch the tsr for syncing; ignore for now")
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}
	// first, check if the tsr is set up for deletion
	// this tsr is set up for deletion
	if !tsr.DeletionTimestamp.IsZero() {
		// re-synchronize the last time for this request before deletion
		for _, tag := range tsr.Spec.Tags {
			if err := r.syncTraffic(ctx, &tsr, tag); err != nil {
				log.Error(err, "unable to synchronize the traffic the last time before deletion")
				return ctrl.Result{}, err
			}
		}
		// if it's successful, we remove the finalizer
		controllerutil.RemoveFinalizer(&tsr, TSR_FINALIZER_NAME)
		if err := r.Update(ctx, &tsr); err != nil {
			log.Error(err, "unable to remove the finalizer")
			return ctrl.Result{}, err
		}
		return ctrl.Result{}, nil
	}
	// this tsr is not set up for deletion

	// if the tsr doesn't have a finalizer, we add one to prevent there is always a
	// force re-synchronization before the tsr is deleted
	if !controllerutil.ContainsFinalizer(&tsr, TSR_FINALIZER_NAME) {
		controllerutil.AddFinalizer(&tsr, TSR_FINALIZER_NAME)
		if err := r.Update(ctx, &tsr); err != nil {
			log.Error(err, "unable to add the finalizer")
			return ctrl.Result{}, err
		}
	}
	lastSyncTime := tsr.Status.LastSyncTime
	syncPeriod := tsr.Spec.SyncPeriod
	// the time for synchronization has not yet come
	for _, tag := range tsr.Spec.Tags {
		if !r.checkIfSyncRequired(ctx, &tsr, tag) {
			continue
		}
		if err := r.syncTraffic(ctx, &tsr, tag); err != nil {
			log.Error(err, "failed to sync traffic")
			return ctrl.Result{}, err
		}
		newTsr := tsr.DeepCopy()
		if lastSyncTime == nil {
			lst := make(map[string]metav1.Time)
			newTsr.Status.LastSyncTime = lst
		}
		newTsr.Status.LastSyncTime[tag] = metav1.Now()

		if err := r.Status().Update(ctx, newTsr); err != nil {
			log.Error(err, "failed to update the status")
			return ctrl.Result{}, err
		}
	}

	return ctrl.Result{RequeueAfter: syncPeriod.Duration}, nil
}

func (r *TrafficSyncRequestReconciler) checkIfSyncRequired(ctx context.Context, tsr *nmv1alpha1.TrafficSyncRequest, tagToSync string) bool {
	if tsr.Status.LastSyncTime == nil {
		return true
	}
	if _lst, ok := tsr.Status.LastSyncTime[tagToSync]; !ok {
		return true
	} else {
		now := metav1.Now().Time
		syncPeriod := tsr.Spec.SyncPeriod
		if !_lst.IsZero() && now.Before(_lst.Add(syncPeriod.Duration)) {
			return false
		}
		return true
	}
}

func (r *TrafficSyncRequestReconciler) syncTraffic(ctx context.Context, tsr *nmv1alpha1.TrafficSyncRequest, tagToSync string) error {
	if tsr == nil || r.Store == nil {
		return nil
	}
	nodeIP := tsr.Spec.NodeIP
	addr := tsr.Spec.Address
	ac, err := nmaclient.NewClient(nodeIP)
	if err != nil {
		return err
	}
	defer ac.Close()

	if resp, err := ac.DumpTraffic(ctx, addr, tagToSync, false); err != nil {
		return err
	} else {
		sentByteMark := resp.SentBytes
		_nn := types.NamespacedName{
			Namespace: tsr.Spec.AssociatedNamespace,
			Name:      tsr.Spec.AssociatedPod,
		}
		nn := _nn.String()
		var pta store.PodTrafficAccount
		var curSentByteMark uint64 = 0
		if found, err := r.Store.FindPTA(ctx, nn, &pta); err != nil {
			return err
		} else if found {
			if err := pta.GetByteMark(addr, tagToSync, 1, false, &curSentByteMark); err != nil {
				return err
			}
		}
		var sentBytes uint64
		sentBytes = sentByteMark - curSentByteMark
		req := store.TagPropReq{
			NamespacedName: nn,
			Addr:           addr,
			Tag:            tagToSync,
		}
		if err := r.Store.UpdateFieldUint64(ctx, req, "$inc", "sent_bytes", sentBytes); err != nil {
			return err
		}
		if err := r.Store.UpdateFieldUint64(ctx, req, "$set", "last_sent_bytes", curSentByteMark); err != nil {
			return err
		}
		if err := r.Store.UpdateFieldUint64(ctx, req, "$set", "cur_sent_bytes", sentByteMark); err != nil {
			return err
		}
	}
	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *TrafficSyncRequestReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&nmv1alpha1.TrafficSyncRequest{}).
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
		}).
		WithOptions(controller.Options{MaxConcurrentReconciles: 5}).
		Complete(r)
}
