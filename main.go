package main

import (
	"flag"
	"os"
	"os/signal"
	"syscall"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/klog/v2"
)

var (
	kubeconfig string
)

func createClient() kubernetes.Interface {
	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		klog.Fatal("failed to build config", err)
	}

	client, err := kubernetes.NewForConfig(config)
	if err != nil {
		klog.Fatal("failed to create client", err)
	}

	klog.Info("client created")
	return client
}

func main() {
	klog.InitFlags(nil)
	flag.Parse()

	client := createClient()

	podsInformerFactory := informers.NewSharedInformerFactoryWithOptions(
		client,
		time.Second*30,
		informers.WithNamespace(metav1.NamespaceDefault))

	controller := CreateController(client, podsInformerFactory.Core().V1().Pods())

	stopCh := make(chan struct{})
	defer close(stopCh)
	podsInformerFactory.Start(stopCh)
	go controller.Run(stopCh)

	sigTerm := make(chan os.Signal, 1)
	signal.Notify(sigTerm, syscall.SIGTERM)
	signal.Notify(sigTerm, syscall.SIGINT)
	<-sigTerm

	klog.Info("controller stopped")
}

func init() {
	flag.StringVar(&kubeconfig, "kubeconfig", "", "Path to the kubeconfig")
}
