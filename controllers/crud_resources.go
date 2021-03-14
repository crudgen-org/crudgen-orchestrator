package controllers

import (
	"context"
	"fmt"
	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	apps "k8s.io/api/apps/v1"
	autoscaling "k8s.io/api/autoscaling/v1"
	core "k8s.io/api/core/v1"
	networking "k8s.io/api/networking/v1beta1"
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
				Name:      crud.DeploymentName(),
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
				Name:      crud.ServiceName(),
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
	fullDomain := fmt.Sprintf("%s.%s", crud.Spec.DomainPrefix, r.RootDomain)
	ingress := &networking.Ingress{}

	switch err := r.Get(ctx, key(crud), ingress); {
	case apierrors.IsNotFound(err):
		ingress = &networking.Ingress{
			ObjectMeta: meta.ObjectMeta{
				Name:      crud.Name,
				Namespace: crud.Namespace,
			},
			Spec: networking.IngressSpec{
				TLS: nil,
				Rules: []networking.IngressRule{
					{
						Host: fullDomain,
						IngressRuleValue: networking.IngressRuleValue{
							HTTP: &networking.HTTPIngressRuleValue{
								Paths: []networking.HTTPIngressPath{
									{
										Backend: networking.IngressBackend{
											ServiceName: crud.ServiceName(),
											ServicePort: intstr.IntOrString{
												IntVal: crud.Status.Port,
											},
										},
									},
								},
							},
						},
					},
				},
			},
		}
		if crud.Spec.EnableTLS {
			ingress.Annotations = map[string]string{
				"cert-manager.io/cluster-issuer": r.ClusterIssuer,
			}
			ingress.Spec.TLS = []networking.IngressTLS{
				{
					Hosts:      []string{fullDomain},
					SecretName: crud.TLSSecretName(),
				},
			}
		}
		if err := controllerutil.SetControllerReference(crud, ingress, r.Scheme); err != nil {
			return errors.Wrap(err, "could not set controller reference on ingress")
		}
		if err := r.Create(ctx, ingress); err != nil {
			return errors.Wrap(err, "could not create ingress")
		}

	case err != nil:
		return errors.Wrap(err, "could not get ingress")

	default:
		// TODO: update if necessary
	}
	return nil
}

func (r *CRUDReconciler) ensureHPA(ctx context.Context, logger logr.Logger, crud *apiv1.CRUD) error {
	hpa := &autoscaling.HorizontalPodAutoscaler{}

	switch err := r.Get(ctx, key(crud), hpa); {
	case apierrors.IsNotFound(err):
		hpa = &autoscaling.HorizontalPodAutoscaler{
			ObjectMeta: meta.ObjectMeta{
				Name:      crud.Name,
				Namespace: crud.Namespace,
			},
			Spec: autoscaling.HorizontalPodAutoscalerSpec{
				ScaleTargetRef: autoscaling.CrossVersionObjectReference{
					Kind:       "Deployment",
					Name:       crud.DeploymentName(),
					APIVersion: "apps/v1",
				},
				MinReplicas:                    pointer.Int32Ptr(1),
				MaxReplicas:                    10,
				TargetCPUUtilizationPercentage: pointer.Int32Ptr(80),
			},
		}
		if err := controllerutil.SetControllerReference(crud, hpa, r.Scheme); err != nil {
			return errors.Wrap(err, "could not set owner reference on hpa")
		}
		if err := r.Create(ctx, hpa); err != nil {
			return errors.Wrap(err, "could not update hpa")
		}

	case err != nil:
		return errors.Wrap(err, "could not retrieve hpa")

	default:
		// TODO change this if crud changes
	}
	return nil
}
