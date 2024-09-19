package pkg

import (
	"context"
	"fmt"
	v17 "k8s.io/api/core/v1"
	v15 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	v16 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	v13 "k8s.io/client-go/informers/core/v1"
	v14 "k8s.io/client-go/informers/networking/v1"
	"k8s.io/client-go/kubernetes"
	v12 "k8s.io/client-go/listers/core/v1"
	v1 "k8s.io/client-go/listers/networking/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
	"reflect"
	"strconv"
	"time"
)

const workNum = 5
const maxRetry = 10

type controller struct {
	client        kubernetes.Interface
	ingressList   v1.IngressLister
	serviceLister v12.ServiceLister
	queue         workqueue.RateLimitingInterface
}

func (c *controller) addService(obj interface{}) {
	c.enqueue(obj)
}

func (c *controller) deleteService(obj interface{}) {
	service, ok := obj.(*v17.Service)
	if !ok {
		runtime.HandleError(fmt.Errorf("expected Service but got %T", obj))
		return
	}

	namespace := service.Namespace
	name := service.Name

	if _, hasAnnotation := service.GetAnnotations()["ingress/http"]; !hasAnnotation {
		return
	}

	_, err := c.ingressList.Ingresses(namespace).Get(name)
	if err != nil {
		if errors.IsNotFound(err) {
			return
		}
		runtime.HandleError(fmt.Errorf("error getting ingress for service %s/%s: %w", namespace, name, err))
		return
	}

	err = c.client.NetworkingV1().Ingresses(namespace).Delete(context.TODO(), name, v16.DeleteOptions{})
	if err != nil {
		runtime.HandleError(fmt.Errorf("error deleting ingress for service %s/%s: %w", namespace, name, err))
	} else {
		fmt.Printf("Successfully deleted ingress for service %s/%s\n", namespace, name)
	}
}

func (c *controller) updateService(oldObj interface{}, newObj interface{}) {

	// 更新时，比较 old 和 new 对象
	if reflect.DeepEqual(oldObj, newObj) {
		return
	}
	c.enqueue(newObj)
}

func (c *controller) enqueue(obj interface{}) {
	// 获取 MetaNamespaceKey
	key, err := cache.MetaNamespaceKeyFunc(obj)
	if err != nil {
		runtime.HandleError(fmt.Errorf("error getting key from cache: %v", err))
		return
	}
	// 将 key 添加到队列中
	fmt.Printf("Enqueue key: %s\n", key)
	c.queue.Add(key)
}

func (c *controller) deleteIngress(obj interface{}) {
	ingress := obj.(*v15.Ingress)

	//  判断这个 Ingress 对象是否有一个 Service 控制器（即它是否由某个 Service 所创建）
	service := v16.GetControllerOf(ingress)
	// 过滤掉非 service 控制器
	if service == nil {
		return
	}

	if service.Kind != "service" {
		return
	}

	c.queue.Add(ingress.Namespace + "/" + ingress.Name)
}

func (c *controller) Run(stopCh chan struct{}) {
	// 确保 Informer 缓存同步

	fmt.Println("Starting workers...")
	for i := 0; i < workNum; i++ {
		go wait.Until(c.worker, time.Second, stopCh)
	}
	<-stopCh
	fmt.Println("Shutting down workers...")
}

func (c *controller) worker() {
	for c.processNextItem() {
	}
}

func (c *controller) processNextItem() bool {
	item, shutdown := c.queue.Get()
	if shutdown {
		return false
	}
	defer c.queue.Done(item)

	// 处理队列中的 item
	key, ok := item.(string)
	if !ok {
		runtime.HandleError(fmt.Errorf("error casting item to string: %v", item))
		return false
	}

	err := c.syncService(key)
	if err != nil {
		c.handleError(key, err)
		return false
	}
	return true
}

func (c *controller) syncService(key string) error {
	namespace, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		return err
	}

	// 获取 service
	service, err := c.serviceLister.Services(namespace).Get(name)
	if errors.IsNotFound(err) {
		fmt.Printf("Service %s/%s not found\n", namespace, name)
		return nil
	}
	if err != nil {
		return err
	}

	// 获取 ingress
	ingress, igerr := c.ingressList.Ingresses(namespace).Get(name)
	if err != nil && !errors.IsNotFound(err) {
		return err
	}

	//  如果 svc  Annotations 带 ingress/http 则创建 ingress
	_, ok := service.GetAnnotations()["ingress/http"]

	domainName := getAnnotationOrDefault(service, "ingress/domain", "www.example.com")
	path := getAnnotationOrDefault(service, "ingress/Path", "/")
	portStr := getAnnotationOrDefault(service, "ingress/Port", "80")
	rewriteTarget := getAnnotationOrDefault(service, "ingress/targetPath", "/")

	port, err := strconv.Atoi(portStr)
	if err != nil {
		port = 80
	}

	if ok && errors.IsNotFound(igerr) {
		// 创建 ingress
		fmt.Printf("Creating ingress for service %s/%s\n", namespace, name)

		ingress := c.constructIngress(domainName, path, int32(port), rewriteTarget, *service)
		_, err := c.client.NetworkingV1().Ingresses(namespace).Create(context.TODO(), ingress, v16.CreateOptions{})
		if err != nil {
			return err
		}
	} else if !ok && ingress != nil {
		// 删除 ingress
		fmt.Printf("Deleting ingress for service %s/%s\n", namespace, name)
		err := c.client.NetworkingV1().Ingresses(namespace).Delete(context.TODO(), name, v16.DeleteOptions{})
		if err != nil {
			return err
		}
	}

	return nil
}

func getAnnotationOrDefault(service *v17.Service, key, defaultValue string) string {
	if value, exists := service.GetAnnotations()[key]; exists {
		return value
	}
	return defaultValue
}

func (c *controller) handleError(key string, err error) {
	if c.queue.NumRequeues(key) < maxRetry {
		fmt.Printf("Error syncing service %s, retrying: %v\n", key, err)
		c.queue.AddRateLimited(key)
	} else {
		fmt.Printf("Max retries reached for service %s, dropping from queue: %v\n", key, err)
		c.queue.Forget(key)
	}
	runtime.HandleError(err)
}

// DomainName string, Path string, SvcPort int32, service
func (c *controller) constructIngress(DomainName string, Path string, SvcPort int32, rewriteTarget string, service v17.Service) *v15.Ingress {

	pathType := v15.PathTypeExact
	IngressCLASS := "nginx"
	ingress := &v15.Ingress{
		ObjectMeta: v16.ObjectMeta{
			Name:      service.Name,
			Namespace: service.Namespace,
			OwnerReferences: []v16.OwnerReference{ // 添加 控制器为 service
				*v16.NewControllerRef(&service, v17.SchemeGroupVersion.WithKind("service")),
			},
			Annotations: map[string]string{
				"nginx.ingress.kubernetes.io/rewrite-target": rewriteTarget,
			},
		},
		Spec: v15.IngressSpec{
			IngressClassName: &IngressCLASS,
			Rules: []v15.IngressRule{
				{
					Host: DomainName,
					IngressRuleValue: v15.IngressRuleValue{
						HTTP: &v15.HTTPIngressRuleValue{
							Paths: []v15.HTTPIngressPath{
								{
									Path:     Path,
									PathType: &pathType, // 设置 PathType
									Backend: v15.IngressBackend{
										Service: &v15.IngressServiceBackend{
											Name: service.Name,
											Port: v15.ServiceBackendPort{
												Number: SvcPort,
											},
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
	return ingress
}

func NewController(client kubernetes.Interface, serviceInformer v13.ServiceInformer, ingressInformer v14.IngressInformer) *controller {
	c := &controller{
		client:        client,
		ingressList:   ingressInformer.Lister(),
		serviceLister: serviceInformer.Lister(),
		queue:         workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "ingressManager"),
	}

	serviceInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    c.addService,
		UpdateFunc: c.updateService,
		DeleteFunc: c.deleteService,
	})

	ingressInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		DeleteFunc: c.deleteIngress,
	})

	return c
}
