package main

import (
	"demo_1/pkg"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"log"
)

func main() {
	// 1.config
	config, err := clientcmd.BuildConfigFromFlags("", clientcmd.RecommendedHomeFile)

	if err != nil {
		inClusterConfig, err := rest.InClusterConfig()
		if err != nil {
			log.Fatalln("Can't get config ")
		}
		config = inClusterConfig
	}
	// 2.client
	clientSet, err := kubernetes.NewForConfig(config)
	if err != nil {
		log.Fatalln("Can't create client")
	}

	// 3.informer

	//factory := informers.NewFilteredSharedInformerFactory(clientSet, time.Second*30, "default", nil)    监听特点命名空间
	factory := informers.NewSharedInformerFactory(clientSet, 0)
	//
	servicesInformer := factory.Core().V1().Services()
	ingressInformer := factory.Networking().V1().Ingresses()

	// 4. add event handler
	controller := pkg.NewController(clientSet, servicesInformer, ingressInformer)

	// 5. informer.Start
	stopCh := make(chan struct{})
	factory.Start(stopCh)
	factory.WaitForCacheSync(stopCh)
	// 	c.queue.Add(ingress.Namespace + "/" + ingress.Name)
	controller.Run(stopCh)
}
