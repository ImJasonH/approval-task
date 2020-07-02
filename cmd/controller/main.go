package main

import (
	"context"

	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1alpha1"
	runinformer "github.com/tektoncd/pipeline/pkg/client/injection/informers/pipeline/v1alpha1/run"
	runreconciler "github.com/tektoncd/pipeline/pkg/client/injection/reconciler/pipeline/v1alpha1/run"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/tools/cache"
	"knative.dev/pkg/configmap"
	"knative.dev/pkg/controller"
	"knative.dev/pkg/injection/sharedmain"
	"knative.dev/pkg/logging"
	"knative.dev/pkg/reconciler"
)

const (
	controllerName = "cel-task-controller"
)

func main() {
	sharedmain.Main(controllerName, newController)
}

func newController(ctx context.Context, cmw configmap.Watcher) *controller.Impl {
	c := &Reconciler{}
	impl := runreconciler.NewImpl(ctx, c, func(impl *controller.Impl) controller.Options {
		return controller.Options{
			AgentName: controllerName,
		}
	})

	runinformer.Get(ctx).Informer().AddEventHandler(cache.FilteringResourceEventHandler{
		FilterFunc: FilterRunRef("cel.example.dev/v0", "Approval"),
		Handler:    controller.HandleAll(impl.Enqueue),
	})

	return impl
}

// FilterRunRef returns a filter that can be passed to a Run Informer, which
// filters out Runs for apiVersion and kinds that this controller doesn't care
// about.
// TODO: Provide this as a helper function.
func FilterRunRef(apiVersion, kind string) func(interface{}) bool {
	return func(obj interface{}) bool {
		r, ok := obj.(*v1alpha1.Run)
		if !ok {
			// Somehow got informed of a non-Run object.
			// Ignore.
			return false
		}
		if r == nil || r.Spec.Ref == nil {
			// These are invalid, but just in case they get
			// created somehow, don't panic.
			return false
		}

		return r.Spec.Ref.APIVersion == apiVersion && r.Spec.Ref.Kind == v1alpha1.TaskKind(kind)
	}
}

type Reconciler struct {
}

// ReconcileKind implements Interface.ReconcileKind.
func (c *Reconciler) ReconcileKind(ctx context.Context, r *v1alpha1.Run) reconciler.Event {
	logger := logging.FromContext(ctx)
	logger.Infof("Reconciling %s/%s", r.Namespace, r.Name)

	if r.IsDone() {
		logger.Info("Run is finished, done reconciling")
		return nil
	}
	return reconciler.NewEvent(corev1.EventTypeNormal, "RunReconciled", "Run reconciled: \"%s/%s\"", r.Namespace, r.Name)
}
