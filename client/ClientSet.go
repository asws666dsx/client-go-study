package main

import (
	"context"
	"fmt"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"log"
)

func test2() {

	//config
	config, err := clientcmd.BuildConfigFromFlags("", clientcmd.RecommendedHomeFile)
	if err != nil {
		log.Fatalln(err)
	}

	// client
	clientSet, err := kubernetes.NewForConfig(config)
	if err != nil {
		log.Fatalln(err)
	}

	// clientSet 中 以 groupVersion 对象进行分组 ,每个 group Version 都有自己对应的client
	coreV1 := clientSet.CoreV1()

	pod, err := coreV1.Pods("default").Get(context.TODO(), "test", v1.GetOptions{})

	if err != nil {
		log.Fatalln(err)
	} else {
		fmt.Println(pod.Spec)
	}
}
