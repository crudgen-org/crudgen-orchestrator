package controllers

import (
	"context"

	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	apps "k8s.io/api/apps/v1"
	core "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	meta "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/utils/pointer"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	apiv1 "github.com/crudgen-org/crudgen-orchestrator/api/v1"
)

func (r *CRUDReconciler) ensureDeployment(ctx context.Context, logger logr.Logger, crud *apiv1.CRUD) error {
	deploy := &apps.Deployment{}

	switch err := r.Get(ctx, key(crud), deploy); {
	case apierrors.IsNotFound(err):
		deploy = &apps.Deployment{
			ObjectMeta: meta.ObjectMeta{
				Name:      crud.Name,
				Namespace: crud.Namespace,
			},
			Spec: apps.DeploymentSpec{
				Replicas: pointer.Int32Ptr(1),
				Selector: &meta.LabelSelector{
					MatchLabels: crud.LabelSelectors(),
				},
				Template: core.PodTemplateSpec{
					ObjectMeta: meta.ObjectMeta{
						Labels: crud.LabelSelectors(),
					},
					Spec: core.PodSpec{
						Containers: []core.Container{
							{
								Name:  crud.Name,
								Image: crud.Status.Image,
							},
						},
					},
				},
			},
		}
		if err := controllerutil.SetControllerReference(crud, deploy, r.Scheme); err != nil {
			return errors.Wrap(err, "could not set owner reference on deployment")
		}
		if err := r.Create(ctx, deploy); err != nil {
			return errors.Wrap(err, "could not update deployment")
		}

	case err != nil:
		return errors.Wrap(err, "could not retrieve deployment")

	default:
		updateDeploy := false
		if deploy.Spec.Template.Spec.Containers == nil ||
			len(deploy.Spec.Template.Spec.Containers) == 0 {
			return errors.New("containers in deployment is nil.")
		}
		if deploy.Spec.Template.Spec.Containers[0].Image != crud.Status.Image {
			deploy.Spec.Template.Spec.Containers[0].Image = crud.Status.Image
			updateDeploy = true
		}
		if updateDeploy {
			if err := r.Update(ctx, deploy); err != nil {
				return errors.Wrap(err, "could not update deployment")
			}
		}
	}
	return nil
}

func (r *CRUDReconciler) ensureService(ctx context.Context, logger logr.Logger, crud *apiv1.CRUD) error {
	service := &core.Service{}

	switch err := r.Get(ctx, key(crud), service); {
	case apierrors.IsNotFound(err):
		service := &core.Service{
			ObjectMeta: meta.ObjectMeta{
				Name:      crud.Name,
				Namespace: crud.Namespace,
			},
			Spec: core.ServiceSpec{
				Ports: []core.ServicePort{
					{
						Name: "api",
						Port: crud.Status.Port,
						TargetPort: intstr.IntOrString{
							IntVal: crud.Status.Port,
						},
					},
				},
				Selector: crud.LabelSelectors(),
				Type:     core.ServiceTypeClusterIP,
			},
		}
		if err := controllerutil.SetControllerReference(crud, service, r.Scheme); err != nil {
			return errors.Wrap(err, "could not set controller reference on service")
		}
		if err := r.Create(ctx, service); err != nil {
			return errors.Wrap(err, "could not create service")
		}

	case err != nil:
		return errors.Wrap(err, "could not get service")

	default:
		// TODO: update if necessary
	}
	return nil
}

func (r *CRUDReconciler) ensureIngress(ctx context.Context, logger logr.Logger, crud *apiv1.CRUD) error {
	return nil
}
