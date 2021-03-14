/*


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

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/api/errors"
	meta "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	apiv1 "github.com/crudgen-org/crudgen-orchestrator/api/v1"
)

// CRUDReconciler reconciles a CRUD object
type CRUDReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme

	RootDomain string
}

func key(object meta.Object) types.NamespacedName {
	return types.NamespacedName{
		Namespace: object.GetNamespace(),
		Name:      object.GetName(),
	}
}

// +kubebuilder:rbac:groups=api.crudgen.org,resources=cruds,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=api.crudgen.org,resources=cruds/status,verbs=get;update;patch

func (r *CRUDReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	ctx := context.Background()
	logger := r.Log.WithValues("crud", req.NamespacedName)

	crud := &apiv1.CRUD{}
	switch err := r.Get(ctx, req.NamespacedName, crud); {
	case errors.IsNotFound(err):
		logger.Info("object to reconcile does not exists", "object", req.NamespacedName)
		return ctrl.Result{}, nil

	case err != nil:
		logger.Error(err, "could not retrieve crud object", "object", req.NamespacedName)
		return ctrl.Result{}, err

	case !crud.GetDeletionTimestamp().IsZero():
		return r.reconcileCleanUp(ctx, logger, crud)

	default:
		if !crud.Status.ImageReady {
			logger.Info("CRUD resource not ready for deployment. stopping...")
			return ctrl.Result{}, nil
		}
		return r.reconcile(ctx, logger, crud)
	}
}

func (r *CRUDReconciler) reconcile(ctx context.Context, logger logr.Logger, crud *apiv1.CRUD) (ctrl.Result, error) {
	if err := r.ensureDeployment(ctx, logger, crud); err != nil {
		return ctrl.Result{}, err
	}
	if err := r.ensureService(ctx, logger, crud); err != nil {
		return ctrl.Result{}, err
	}
	if err := r.ensureIngress(ctx, logger, crud); err != nil {
		return ctrl.Result{}, err
	}
	if !crud.Status.Deployed {
		crud.Status.Deployed = true
		if err := r.Update(ctx, crud); err != nil {
			return ctrl.Result{}, err
		}
	}
	return ctrl.Result{}, nil
}

func (r *CRUDReconciler) reconcileCleanUp(ctx context.Context, logger logr.Logger, crud *apiv1.CRUD) (ctrl.Result, error) {
	return ctrl.Result{}, nil
}

func (r *CRUDReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&apiv1.CRUD{}).
		Complete(r)
}
