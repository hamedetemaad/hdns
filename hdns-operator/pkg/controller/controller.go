package controller

import (
	"context"
	"errors"
	"time"

	"github.com/gotway/gotway/pkg/log"

	dbv1alpha1 "github.com/hamedetemaad/hdns-operator/pkg/dnsblock/v1alpha1"
	dbv1alpha1clientset "github.com/hamedetemaad/hdns-operator/pkg/dnsblock/v1alpha1/apis/clientset/versioned"
	dbinformers "github.com/hamedetemaad/hdns-operator/pkg/dnsblock/v1alpha1/apis/informers/externalversions"

	"github.com/hamedetemaad/hdns-operator/internal/config"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
)

type Controller struct {
	kubeClientSet kubernetes.Interface

	dbInformer cache.SharedIndexInformer

	queue workqueue.RateLimitingInterface

	namespace string

	logger log.Logger

	config config.Config
}

func (c *Controller) Run(ctx context.Context, numWorkers int, config config.Config) error {
	c.config = config
	defer utilruntime.HandleCrash()
	defer c.queue.ShutDown()

	c.logger.Info("starting controller")

	c.logger.Info("starting informers")
	for _, i := range []cache.SharedIndexInformer{
		c.dbInformer,
	} {
		go i.Run(ctx.Done())
	}

	c.logger.Info("waiting for informer caches to sync")
	if !cache.WaitForCacheSync(ctx.Done(), []cache.InformerSynced{
		c.dbInformer.HasSynced,
	}...) {
		err := errors.New("failed to wait for informers caches to sync")
		utilruntime.HandleError(err)
		return err
	}

	c.logger.Infof("starting %d workers", numWorkers)
	for i := 0; i < numWorkers; i++ {
		go wait.Until(func() {
			c.runWorker(ctx)
		}, time.Second, ctx.Done())
	}
	c.logger.Info("controller ready")

	<-ctx.Done()
	c.logger.Info("stopping controller")

	return nil
}

func (c *Controller) addDNSBlock(obj interface{}) {
	c.logger.Debug("adding dnsblock")
	db, ok := obj.(*dbv1alpha1.DNSBlock)
	if !ok {
		c.logger.Errorf("unexpected object %v", obj)
		return
	}
	c.queue.Add(event{
		eventType: addDNSBlock,
		newObj:    db.DeepCopy(),
	})
}

func New(
	kubeClientSet kubernetes.Interface,
	dbClientSet dbv1alpha1clientset.Interface,
	namespace string,
	logger log.Logger,
) *Controller {

	dbInformerFactory := dbinformers.NewSharedInformerFactory(
		dbClientSet,
		10*time.Second,
	)
	dbInformer := dbInformerFactory.Hdns().V1alpha1().DNSBlocks().Informer()

	queue := workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter())

	ctrl := &Controller{
		kubeClientSet: kubeClientSet,

		dbInformer: dbInformer,

		queue: queue,

		namespace: namespace,

		logger: logger,
	}

	dbInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: ctrl.addDNSBlock,
	})

	return ctrl
}
