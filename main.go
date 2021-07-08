package main

import (
	"bufio"
	"flag"
	"log"
	"os"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

var (
	kubeconfig string
)

func createClient() kubernetes.Interface {
	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		log.Fatal("failed to build config", err)
	}

	client, err := kubernetes.NewForConfig(config)
	if err != nil {
		log.Fatal("failed to create client", err)
	}

	log.Println("client created")
	return client
}

func main() {
	flag.Parse()

	client := createClient()

	podsInformerFactory := informers.NewSharedInformerFactoryWithOptions(
		client,
		time.Second * 30,
		informers.WithNamespace(metav1.NamespaceDefault))

	controller := CreateController(client, podsInformerFactory.Core().V1().Pods())

	stopCh := make(chan struct{})
	defer close(stopCh)
	podsInformerFactory.Start(stopCh)
	go controller.Run(stopCh)

	reader := bufio.NewReader(os.Stdin)
	text, _ := reader.ReadString('\n')
	log.Println(text)
}

func init() {
	flag.StringVar(&kubeconfig, "kubeconfig", "", "Path to the kubeconfig")
}
