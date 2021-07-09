package main

import (
	"k8s.io/klog/v2"
	"time"

	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	corev1 "k8s.io/client-go/informers/core/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
)

type Controller struct {
	kubeClientSet kubernetes.Interface
	workqueue     workqueue.RateLimitingInterface
	podInformer   corev1.PodInformer
	podsSynced    cache.InformerSynced
}

func CreateController(kubeClientSet kubernetes.Interface, podInformer corev1.PodInformer) *Controller {
	controller := &Controller{
		kubeClientSet: kubeClientSet,
		workqueue:     workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter()),
		podInformer:   podInformer,
		podsSynced:    podInformer.Informer().HasSynced,
	}

	podInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			key, err := cache.MetaNamespaceKeyFunc(obj)
			klog.Info("Add pod:", key)
			if err == nil {
				controller.workqueue.Add(key)
			}
		},
		UpdateFunc: func(oldObj, newObj interface{}) {
			key, err := cache.MetaNamespaceKeyFunc(newObj)
			klog.Info("Update pod:", key)
			if err == nil {
				controller.workqueue.Add(key)
			}
		},
		DeleteFunc: func(obj interface{}) {
			key, err := cache.MetaNamespaceKeyFunc(obj)
			klog.Info("Delete pod:", key)
			if err == nil {
				controller.workqueue.Add(key)
			}
		},
	})
	return controller
}

func (c *Controller) Run(stopCh <-chan struct{}) {
	defer utilruntime.HandleCrash()
	defer c.workqueue.ShutDown()

	klog.Info("Starting Controller")

	if !cache.WaitForCacheSync(stopCh, c.podsSynced) {
		klog.Fatal("failed to sync cache")
	}

	klog.Info("cache synced")
	wait.Until(c.runWorker, time.Second, stopCh)
}

func (c *Controller) runWorker() {
	for c.processNextWorkItem() {
	}

	klog.Info("worker finished")
}

func (c *Controller) processNextWorkItem() bool {
	klog.Info("controller process next item")

	key, shutdown := c.workqueue.Get()

	if shutdown {
		return false
	}

	defer c.workqueue.Done(key)

	keyRaw := key.(string)
	_, _, err := c.podInformer.Informer().GetIndexer().GetByKey(keyRaw)

	if err != nil {
		if c.workqueue.NumRequeues(key) < 5 {
			klog.Error("failed to process key", key, "with error, retrying ...", err)
			c.workqueue.AddRateLimited(key)
		} else {
			klog.Error("failed to process key", key, "with error, discarding ...", err)
			c.workqueue.Forget(key)
			utilruntime.HandleError(err)
		}
	}

	return true
}
