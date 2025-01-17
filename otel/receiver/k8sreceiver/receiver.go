package k8sreceiver

import (
	"context"
	"fmt"
	"os"
	"sync"

	"github.com/get-glu/glu/otel/internal/ids"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/receiver"
	semconv "go.opentelemetry.io/otel/semconv/v1.27.0"
	"go.uber.org/zap"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/informers/apps"
	"k8s.io/client-go/informers/core"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/clientcmd"
)

type k8sReceiver struct {
	cfg    *Config
	logger *zap.Logger

	ctx    context.Context
	cancel func()

	logsConsumer        consumer.Logs
	factory             informers.SharedInformerFactory
	podsInformer        cache.SharedInformer
	replicasetInformer  cache.SharedInformer
	deploymentsInformer cache.SharedInformer

	states sync.Map
}

// Start tells the component to start. Host parameter can be used for communicating
// with the host after Start() has already returned. If an error is returned by
// Start() then the collector startup will be aborted.
// If this is an exporter component it may prepare for exporting
// by connecting to the endpoint.
//
// If the component needs to perform a long-running starting operation then it is recommended
// that Start() returns quickly and the long-running operation is performed in background.
// In that case make sure that the long-running operation does not use the context passed
// to Start() function since that context will be cancelled soon and can abort the long-running
// operation. Create a new context from the context.Background() for long-running operations.
func (k *k8sReceiver) Start(ctx context.Context, host component.Host) error {
	k.ctx, k.cancel = context.WithCancel(ctx)

	clientset, err := k8sClient(k.cfg)
	if err != nil {
		return err
	}

	k.factory = informers.NewSharedInformerFactory(clientset, 0)

	// wait for cache sync so that we have replicasets before observing pods
	deploymentsInformer := apps.New(k.factory, "", func(lo *metav1.ListOptions) {}).V1().Deployments().Informer()
	k.deploymentsInformer = deploymentsInformer

	_, err = deploymentsInformer.AddEventHandler(TypedEventHandler[*appsv1.Deployment]{
		logger:     k.logger,
		AddFunc:    k.onAddDeployment,
		UpdateFunc: k.onUpdateDeployment,
		DeleteFunc: k.onDeleteDeployment,
	})
	if err != nil {
		return err
	}

	k.logger.Info("Starting deployments informer")
	go deploymentsInformer.Run(k.ctx.Done())

	k.logger.Debug("Waiting for cache sync")
	k.factory.WaitForCacheSync(ctx.Done())

	rsInformer := apps.New(k.factory, "", func(lo *metav1.ListOptions) {}).V1().ReplicaSets().Informer()
	k.replicasetInformer = rsInformer

	k.logger.Info("Starting replicasets informer")
	go rsInformer.Run(k.ctx.Done())

	podsInformer := core.New(k.factory, "", func(lo *metav1.ListOptions) {}).V1().Pods().Informer()
	k.podsInformer = podsInformer
	_, err = podsInformer.AddEventHandler(TypedEventHandler[*corev1.Pod]{
		logger:     k.logger,
		UpdateFunc: k.onUpdatePod,
	})
	if err != nil {
		return err
	}

	k.logger.Info("Starting pods informer")
	go podsInformer.Run(k.ctx.Done())

	return nil
}

// Shutdown is invoked during service shutdown. After Shutdown() is called, if the component
// accepted data in any way, it should not accept it anymore.
//
// This method must be safe to call:
//   - without Start() having been called
//   - if the component is in a shutdown state already
//
// If there are any background operations running by the component they must be aborted before
// this function returns. Remember that if you started any long-running background operations from
// the Start() method, those operations must be also cancelled. If there are any buffers in the
// component, they should be cleared and the data sent immediately to the next component.
//
// The component's lifecycle is completed once the Shutdown() method returns. No other
// methods of the component are called after that. If necessary a new component with
// the same or different configuration may be created and started (this may happen
// for example if we want to restart the component).
func (k *k8sReceiver) Shutdown(ctx context.Context) error {
	if k.cancel != nil {
		k.cancel()
	}
	return nil
}

type deploymentContainer struct {
	namespace string
	name      string
	container string
}

type deploymentContainerState struct {
	isUpdating bool
	image      string
	digest     string
}

func (k *k8sReceiver) onAddDeployment(deployment *appsv1.Deployment) {
	_ = k.logger.With(zap.String("namespace", deployment.Namespace), zap.String("name", deployment.Name))

	meta := deployment.ObjectMeta
	for _, c := range deployment.Spec.Template.Spec.Containers {
		k.updateState(meta.Namespace, meta.Name, c.Name, c.Image)
	}
}

func (k *k8sReceiver) onUpdateDeployment(oldD, newD *appsv1.Deployment) {
	meta := newD.ObjectMeta
	for _, c := range newD.Spec.Template.Spec.Containers {
		k.updateState(meta.Namespace, meta.Name, c.Name, c.Image)
	}
}

func (k *k8sReceiver) onUpdatePod(oldP, newP *corev1.Pod) {
	logger := k.logger.With(zap.String("namespace", newP.Namespace), zap.String("name", newP.Name))

	deployment, ok := k.getDeployment(logger, newP.Namespace, newP.OwnerReferences...)
	if !ok {
		logger.Warn("Could not find deployment state ")
		return
	}

	for _, status := range newP.Status.ContainerStatuses {
		key := deploymentContainer{deployment.Namespace, deployment.Name, status.Name}
		v, ok := k.states.Load(key)
		if !ok {
			logger.Warn("Deployment for pod not found")
			continue
		}

		existing, ok := v.(deploymentContainerState)
		if !ok {
			logger.Error("Unexpected deployment state type")
			continue
		}

		// if new image ID observed
		logger.Info("Pod Update", zap.String("image", status.ImageID), zap.Bool("isUpdating", existing.isUpdating))
		if status.ImageID != "" && existing.isUpdating {
			existing.isUpdating = false
			existing.digest = status.ImageID
			k.states.Store(key, existing)

			ociAttrs, err := fetchOCIAttributes(k.ctx, status.ImageID)
			if err != nil {
				logger.Warn("Error fetching OCI attributes", zap.Error(err))
				// here we simply leave the map as unassigned so that it
				// returns false on any lookup
			}

			var traceID pcommon.TraceID
			rev, ok := ociAttrs["org.opencontainers.image.revision"]
			if !ok {
				traceID, err = ids.TraceFromString(status.ImageID)
				if err != nil {
					logger.Error("Generating trace ID for Image SHA", zap.Error(err))
				}
			} else {
				traceID, err = ids.TraceFromString(rev)
				if err != nil {
					logger.Error("Generating trace ID for revision", zap.Error(err))
					continue
				}
			}

			// emit first observation of new image version for container
			logs := plog.NewLogs()
			log := logs.ResourceLogs().AppendEmpty()

			attrs := log.Resource().Attributes()
			attrs.PutStr(string(semconv.ServiceNameKey), fmt.Sprintf("kubernetes/%s", k.cfg.ClusterName))
			attrs.PutStr(string(semconv.K8SNamespaceNameKey), deployment.Namespace)
			attrs.PutStr(string(semconv.K8SDeploymentNameKey), deployment.Name)
			attrs.PutStr(string(semconv.K8SContainerNameKey), status.Name)
			attrs.PutStr(string(semconv.OciManifestDigestKey), status.ImageID)

			scopeLogs := log.ScopeLogs().AppendEmpty()
			scopeLog := scopeLogs.LogRecords().AppendEmpty()
			scopeLog.SetSeverityNumber(plog.SeverityNumberInfo)
			scopeLog.SetSeverityText(plog.SeverityNumberInfo.String())
			scopeLog.SetTimestamp(pcommon.NewTimestampFromTime(status.State.Running.StartedAt.Time))
			scopeLog.SetTraceID(traceID)
			scopeLog.Body().SetStr("New Image Observed")

			if err := k.logsConsumer.ConsumeLogs(k.ctx, logs); err != nil {
				logger.Error("attempting to consume log", zap.Error(err))
			}
		}
	}
}

func (k *k8sReceiver) onDeleteDeployment(deployment *appsv1.Deployment) {
	for _, c := range deployment.Spec.Template.Spec.Containers {
		k.states.Delete(deploymentContainer{deployment.Namespace, deployment.Name, c.Name})
	}
}

func (k *k8sReceiver) updateState(namespace, name, container, image string) {
	logger := k.logger.With(zap.String("namespace", namespace), zap.String("name", name))

	key := deploymentContainer{namespace, name, container}
	v, ok := k.states.LoadOrStore(key, deploymentContainerState{image: image})
	if !ok {
		logger.Debug("New deployment observed")
		return
	}

	existing, ok := v.(deploymentContainerState)
	if !ok {
		logger.Error("Unexpected container state type")
		return
	}

	existing.isUpdating = existing.isUpdating || existing.image != image
	existing.image = image
	k.states.Store(key, existing)

	logger.Debug("Container state updated", zap.String("image", image), zap.Bool("isUpdating", existing.isUpdating))
}

func (k *k8sReceiver) getDeployment(logger *zap.Logger, namespace string, refs ...metav1.OwnerReference) (*appsv1.Deployment, bool) {
	for _, ref := range refs {
		if ref.APIVersion != "apps/v1" {
			continue
		}

		switch ref.Kind {
		case "ReplicaSet":
			obj, exists, err := k.replicasetInformer.GetStore().Get(&appsv1.ReplicaSet{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: namespace,
					Name:      ref.Name,
				},
			})
			if err != nil {
				logger.Error("Getting replicaset", zap.Error(err))
				return nil, false
			}

			if !exists {
				logger.Debug("Could not locate pods owner replicaset")
				return nil, false
			}

			replicaset, ok := obj.(*appsv1.ReplicaSet)
			if !ok {
				logger.Warn("Unexpected type for ReplicaSet")
				return nil, false
			}

			// descend into replicaset
			return k.getDeployment(logger, namespace, replicaset.OwnerReferences...)
		case "Deployment":
			obj, exists, err := k.deploymentsInformer.GetStore().Get(&appsv1.Deployment{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: namespace,
					Name:      ref.Name,
				},
			})
			if err != nil {
				logger.Error("Getting deployment", zap.Error(err))
				return nil, false
			}

			if !exists {
				logger.Debug("Could not locate pods owner deployment")
				return nil, false
			}

			deployment, ok := obj.(*appsv1.Deployment)
			return deployment, ok
		}
	}

	return nil, false
}

func newReceiver(
	settings receiver.Settings,
	cfg component.Config,
) receiver.Logs {
	return &k8sReceiver{
		cfg:    cfg.(*Config),
		logger: settings.Logger,
	}
}

func createLogsConsumer(
	ctx context.Context,
	settings receiver.Settings,
	ccfg component.Config,
	nextConsumer consumer.Logs,
) (receiver.Logs, error) {
	r := receivers.GetOrAdd(settings.ID, func() component.Component {
		return newReceiver(settings, ccfg.(*Config))
	})

	r.Unwrap().(*k8sReceiver).logsConsumer = nextConsumer

	return r, nil
}

func k8sClient(cfg *Config) (kubernetes.Interface, error) {
	switch cfg.AuthType {
	case AuthTypeKubeConfig:
		if cfg := os.Getenv("KUBECONFIG"); cfg != "" {
			kcfg, err := clientcmd.BuildConfigFromFlags("", cfg)
			if err != nil {
				return nil, err
			}

			return kubernetes.NewForConfig(kcfg)
		}

		loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
		configOverrides := &clientcmd.ConfigOverrides{}
		authConf, err := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
			loadingRules, configOverrides).ClientConfig()
		if err != nil {
			return nil, fmt.Errorf("error connecting to k8s with auth_type=%s: %w", AuthTypeKubeConfig, err)
		}

		return kubernetes.NewForConfig(authConf)
	case AuthTypeServiceAccount:
		cfg, err := rest.InClusterConfig()
		if err != nil {
			return nil, err
		}

		return kubernetes.NewForConfig(cfg)
	default:
		authConf := &rest.Config{
			Host: "localhost:6443",
		}
		authConf.Insecure = true
		return kubernetes.NewForConfig(authConf)
	}
}

type TypedEventHandler[T any] struct {
	logger     *zap.Logger
	AddFunc    func(T)
	UpdateFunc func(T, T)
	DeleteFunc func(T)
}

// OnAdd calls AddFunc if it's not nil.
func (t TypedEventHandler[T]) OnAdd(obj interface{}, isInInitialList bool) {
	if t.logger != nil {
		t.logger.Debug("Resource added")
	}

	if t.AddFunc != nil {
		t.AddFunc(obj.(T))
	}
}

// OnUpdate calls UpdateFunc if it's not nil.
func (t TypedEventHandler[T]) OnUpdate(oldObj, newObj interface{}) {
	if t.logger != nil {
		t.logger.Debug("Resource updated")
	}

	if t.UpdateFunc != nil {
		var oldT T
		if oldObj != nil {
			oldT = oldObj.(T)
		}

		var newT T
		if newObj != nil {
			newT = newObj.(T)
		}

		t.UpdateFunc(oldT, newT)
	}
}

// OnDelete calls DeleteFunc if it's not nil.
func (t TypedEventHandler[T]) OnDelete(obj interface{}) {
	if t.logger != nil {
		t.logger.Debug("Resource deleted")
	}

	if t.DeleteFunc != nil {
		t.DeleteFunc(obj.(T))
	}
}
