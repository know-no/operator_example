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
	"github.com/go-logr/logr"
	appv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	devopsv1 "my.domain/example/api/v1"
)

// DemoMicroServiceReconciler reconciles a DemoMicroService object
type DemoMicroServiceReconciler struct {
	client.Client
	log    logr.Logger
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=devops.my.domain,resources=demomicroservices,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=devops.my.domain,resources=demomicroservices/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=devops.my.domain,resources=demomicroservices/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the DemoMicroService object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.13.0/pkg/reconcile
func (r *DemoMicroServiceReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = log.FromContext(ctx)

	// TODO(user): your logic here

	return ctrl.Result{}, nil
}

func (r *DemoMicroServiceReconciler) DoReconcile1(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	dms := &devopsv1.DemoMicroService{}
	if err := r.Get(ctx, req.NamespacedName, dms, &client.GetOptions{}); err != nil { // 调用的reader的接口
		if err := client.IgnoreNotFound(err); err == nil { // 判断是否是not found 才
			r.log.Info("此时没有找到对应的 DemoMicroService resource, 即此处进入了 resource 被删除成功后的生命周期")
			return ctrl.Result{}, nil
		} else {
			r.log.Error(err, "不是未找到的删除错误， 直接返回")
			return ctrl.Result{}, err
		}
	}
	r.log.Info("走到这里意味着 DemoMicroService resource 被找到，即该 resource 被成功创建，进入到了可根据该 resource 来执行逻辑的主流程")
	podLabels := map[string]string{
		"app": req.Name,
	}
	deployment := appv1.Deployment{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Deployment",
			APIVersion: "apps/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      req.Name,
			Namespace: req.Namespace,
		},
		Spec: appv1.DeploymentSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: podLabels,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: podLabels,
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:            req.Name,
							Image:           dms.Spec.Image,
							ImagePullPolicy: "Always",
							Ports: []corev1.ContainerPort{
								{
									ContainerPort: 9898,
								},
							},
						},
					},
				},
			},
		},
	}

	if err := r.Create(ctx, &deployment); err != nil {
		r.log.Error(err, "create deployment wrong")
		return ctrl.Result{}, nil
	}

	return ctrl.Result{}, nil
}

const (
	demoMicroServiceFinalizer string = "demomicroservice.finalizers.devops.my.domain"
)

func (r *DemoMicroServiceReconciler) DoReconcile2(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	dms := &devopsv1.DemoMicroService{}
	if err := r.Get(ctx, req.NamespacedName, dms, &client.GetOptions{}); err != nil { // 调用的reader的接口
		if err := client.IgnoreNotFound(err); err == nil { // 判断是否是not found 才
			r.log.Info("此时没有找到对应的 DemoMicroService resource, 即此处进入了 resource 被删除成功后的生命周期")
			return ctrl.Result{}, nil
		} else {
			r.log.Error(err, "不是未找到的删除错误， 直接返回")
			return ctrl.Result{}, err
		}
	}

	if dms.ObjectMeta.DeletionTimestamp.IsZero() {
		r.log.Info("进入到 apply 这个 DemoMicroService CR 的逻辑")
		r.log.Info("此时必须确保 resource 的 finalizers 里有控制器指定的 finalizer")

		if !ContainsString(dms.ObjectMeta.Finalizers, demoMicroServiceFinalizer) {
			dms.ObjectMeta.Finalizers = append(dms.ObjectMeta.Finalizers, demoMicroServiceFinalizer)
			if err := r.Update(ctx, dms); err != nil {
				return ctrl.Result{}, err
			}
		}
		if _, err := r.applyDeployment(ctx, req, dms); err != nil { // todo

		}
	} else { // 已经发送了删除指令
		r.log.Info("进入都删除 demo micro service cr的逻辑")
		if ContainsString(dms.ObjectMeta.Finalizers, demoMicroServiceFinalizer) { // 还存在finalizer，说明物理上还没删除
			r.log.Info("如果 finalizers 被清空，则该 DemoMicroService CR 就已经不存在了，所以必须在次之前删除 Deployment")
			if err := r.cleanDeployment(ctx, req); err != nil {
				return ctrl.Result{}, nil
			}
		}
		r.log.Info("清空 finalizers，在此之后该 DemoMicroService CR 才会真正消失")
		dms.ObjectMeta.Finalizers = RemoveString(dms.ObjectMeta.Finalizers, demoMicroServiceFinalizer)
		if err := r.Update(ctx, dms); err != nil {
			return ctrl.Result{}, err
		}
	}
	return ctrl.Result{}, nil
}

func (r *DemoMicroServiceReconciler) applyDeployment(ctx context.Context, req ctrl.Request,
	dms *devopsv1.DemoMicroService) (*appv1.Deployment, error) {

	podLabels := map[string]string{
		"app": req.Name,
	}
	deployment := appv1.Deployment{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Deployment",
			APIVersion: "apps/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      req.Name,
			Namespace: req.Namespace,
		},
		Spec: appv1.DeploymentSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: podLabels,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: podLabels,
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:            req.Name,
							Image:           dms.Spec.Image,
							ImagePullPolicy: "Always",
							Ports: []corev1.ContainerPort{
								{
									ContainerPort: 9898,
								},
							},
						},
					},
				},
			},
		},
	}

	oldDeployment := &appv1.Deployment{}
	if err := r.Get(ctx, req.NamespacedName, oldDeployment); err != nil {
		if err := client.IgnoreNotFound(err); err == nil {
			if err := r.Create(ctx, &deployment); err != nil {
				return nil, err
			}
		} else {
			return nil, err
		}
	}
	// 标识已存在, 准备update
	if err := r.Update(ctx, &deployment); err != nil {
		return nil, err
	}

	return &deployment, nil
}
func (r *DemoMicroServiceReconciler) cleanDeployment(ctx context.Context, req ctrl.Request) error {
	deployment := &appv1.Deployment{}
	if err := r.Get(ctx, req.NamespacedName, deployment); err != nil {
		if err := client.IgnoreNotFound(err); err == nil {
			return nil
		} else {
			return err
		}
	}
	if err := r.Delete(ctx, deployment); err != nil {
		return err
	}
	return nil

}

// SetupWithManager sets up the controller with the Manager.
func (r *DemoMicroServiceReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&devopsv1.DemoMicroService{}).
		Complete(r)
}

func ContainsString(slice []string, s string) bool {
	for _, item := range slice {
		if item == s {
			return true
		}
	}
	return false
}

func RemoveString(slice []string, s string) (result []string) {
	for _, item := range slice {
		if item == s {
			continue
		}
		result = append(result, item)
	}
	return
}
