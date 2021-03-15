package controllers

import (
	"context"
	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	apps "k8s.io/api/apps/v1"
	core "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	meta "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/utils/pointer"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	apiv1 "github.com/crudgen-org/crudgen-orchestrator/api/v1"
)

func (r *CRUDReconciler) ensureDatabseStatefulset(ctx context.Context, logger logr.Logger, crud *apiv1.CRUD) error {
	sts := &apps.StatefulSet{}
	resourceQuantity, _ := resource.ParseQuantity("3G")

	switch err := r.Get(ctx, key(crud), sts); {
	case apierrors.IsNotFound(err):
		sts = &apps.StatefulSet{
			ObjectMeta: meta.ObjectMeta{
				Name:      crud.DatabaseStatefulName(),
				Namespace: crud.Namespace,
			},
			Spec: apps.StatefulSetSpec{
				Replicas: pointer.Int32Ptr(1),
				Selector: &meta.LabelSelector{
					MatchLabels: crud.DatabaseLabel(),
				},
				Template: core.PodTemplateSpec{
					ObjectMeta: meta.ObjectMeta{
						Labels: map[string]string{
							"app":  "postgres",
							"crud": crud.GetName(),
						},
					},
					Spec: core.PodSpec{
						Volumes:        nil,
						InitContainers: nil,
						Containers: []core.Container{
							{
								Name:  "pg",
								Image: "postgres:13",
								Ports: []core.ContainerPort{
									{
										Name:          "ordb",
										ContainerPort: 5432,
									},
								},
								EnvFrom: []core.EnvFromSource{
									{
										ConfigMapRef: &core.ConfigMapEnvSource{
											LocalObjectReference: core.LocalObjectReference{
												Name: crud.DatabaseConfigMapName(),
											},
										},
									},
								},
								VolumeMounts: []core.VolumeMount{
									{
										Name:      "ordb",
										MountPath: "/var/lib/PostgreSQL/data",
										SubPath:   "Postgres",
									},
								},
								LivenessProbe:  nil, // TODO
								ReadinessProbe: nil, // TODO
							},
						},
					},
				},
				VolumeClaimTemplates: []core.PersistentVolumeClaim{
					{
						ObjectMeta: meta.ObjectMeta{
							Name: "ordb",
						},
						Spec: core.PersistentVolumeClaimSpec{
							AccessModes: []core.PersistentVolumeAccessMode{
								"ReadWriteOnce",
							},
							Resources: core.ResourceRequirements{
								Requests: map[core.ResourceName]resource.Quantity{
									"storage": resourceQuantity,
								},
							},
							StorageClassName: pointer.StringPtr("hiops"),
						},
					},
				},
			},
		}
		if err := controllerutil.SetControllerReference(crud, sts, r.Scheme); err != nil {
			return errors.Wrap(err, "could not set owner reference on database statefulset")
		}
		if err := r.Create(ctx, sts); err != nil {
			return errors.Wrap(err, "could not update statefulset")
		}

	case err != nil:
		return errors.Wrap(err, "could not retrieve statefulset")

	default:
		//updateDeploy := false
		//if sts.Spec.Template.Spec.Containers == nil ||
		//	len(sts.Spec.Template.Spec.Containers) == 0 {
		//	return errors.New("containers in deployment is nil.")
		//}
		//if sts.Spec.Template.Spec.Containers[0].Image != crud.Status.Image {
		//	sts.Spec.Template.Spec.Containers[0].sts = crud.Status.Image
		//	updateDeploy = true
		//}
		//if updateDeploy {
		//	if err := r.Update(ctx, sts); err != nil {
		//		return errors.Wrap(err, "could not update deployment")
		//	}
		//}
		// TODO update if necessary
	}
	return nil
}

func (r *CRUDReconciler) ensureDatabaseService(ctx context.Context, logger logr.Logger, crud *apiv1.CRUD) error {
	service := &core.Service{}

	key := key(crud)
	key.Name = crud.DatabaseServiceName()
	switch err := r.Get(ctx, key, service); {
	case apierrors.IsNotFound(err):
		service := &core.Service{
			ObjectMeta: meta.ObjectMeta{
				Name:      crud.DatabaseServiceName(),
				Namespace: crud.Namespace,
			},
			Spec: core.ServiceSpec{
				Ports: []core.ServicePort{
					{
						Name: "pg",
						Port: 5432,
						TargetPort: intstr.IntOrString{
							IntVal: 5432,
						},
					},
				},
				Selector: crud.DatabaseLabel(),
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
