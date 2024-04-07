package controller

import (
	"context"
	"encoding/json"
	"fmt"

	dbv1alpha1 "github.com/hamedetemaad/hdns-operator/pkg/dnsblock/v1alpha1"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/tools/cache"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const maxRetries = 3

type RequestBody struct {
	Name        string `json:"name"`
	Path        string `json:"path"`
	ActiveUsers int    `json:"activeUsers"`
	Host        string `json:"host"`
}

type ResponseBody struct {
	Status  string `json:"status"`
	Message string `json:"message"`
}

func (c *Controller) runWorker(ctx context.Context) {
	for c.processNextItem(ctx) {
	}
}

func (c *Controller) processNextItem(ctx context.Context) bool {
	obj, shutdown := c.queue.Get()
	if shutdown {
		return false
	}
	defer c.queue.Done(obj)

	err := c.processEvent(ctx, obj)
	if err == nil {
		c.logger.Debug("processed item")
		c.queue.Forget(obj)
	} else if c.queue.NumRequeues(obj) < maxRetries {
		c.logger.Errorf("error processing event: %v, retrying", err)
		c.queue.AddRateLimited(obj)
	} else {
		c.logger.Errorf("error processing event: %v, max retries reached", err)
		c.queue.Forget(obj)
		utilruntime.HandleError(err)
	}

	return true
}

func (c *Controller) processEvent(ctx context.Context, obj interface{}) error {
	event, ok := obj.(event)
	if !ok {
		c.logger.Error("unexpected event ", obj)
		return nil
	}
	switch event.eventType {
	case addDNSBlock:
		return c.processAddDNSBlock(ctx, event.newObj.(*dbv1alpha1.DNSBlock))
	}
	return nil
}

func (c *Controller) processAddDNSBlock(ctx context.Context, db *dbv1alpha1.DNSBlock) error {
	hdnsCm, _ := c.kubeClientSet.CoreV1().ConfigMaps("hdns").Get(context.TODO(), "hdns-cm", metav1.GetOptions{})

	cfg := hdnsCm.Data["hdns.cfg"]

	var resultCfg map[string]interface{}

	json.Unmarshal([]byte(cfg), &resultCfg)

	oldDomains := resultCfg["block_domains"]
	a := oldDomains.([]interface{})
	tmp := make([]string, len(a))
	for i, v := range a {
		str, ok := v.(string)
		if !ok {
			fmt.Printf("Element at index %d is not a string\n", i)
			return nil
		}
		tmp[i] = str
	}

	domains := append(tmp, db.Spec.Domains...)
	resultCfg["block_domains"] = domains
	fmt.Println(resultCfg)
	jsonBytes, err := json.Marshal(resultCfg)
	if err != nil {
		fmt.Println("Error marshaling JSON:", err)
		return nil
	}

	// Convert JSON bytes to a string
	newCfg := string(jsonBytes)
	hdnsCm.Data = map[string]string{
		"hdns.cfg": newCfg,
	}

	c.kubeClientSet.CoreV1().ConfigMaps("hdns").Update(context.TODO(), hdnsCm, metav1.UpdateOptions{})

	return nil
}

func resourceExists(obj interface{}, indexer cache.Indexer) (bool, error) {
	key, err := cache.MetaNamespaceKeyFunc(obj)
	if err != nil {
		return false, fmt.Errorf("error getting key %v", err)
	}
	_, exists, err := indexer.GetByKey(key)
	return exists, err
}
