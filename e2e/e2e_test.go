/*
Copyright 2024 Red Hat, Inc.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package e2e

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"testing"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/rand"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"

	buildClient "github.com/shipwright-io/build/pkg/client/clientset/versioned"
	buildv1beta1 "github.com/shipwright-io/build/pkg/client/clientset/versioned/typed/build/v1beta1"
	tektonClient "github.com/tektoncd/pipeline/pkg/client/clientset/versioned"
)

// Defaults
const (
	NamespacePrefix           = "test"
	OpenShiftImageRegistry    = "image-registry.openshift-image-registry.svc:5000"
	BuildahStrategyName       = "buildah"
	SourceToImageStrategyName = "source-to-image"
	BuildpackStrategyName     = "buildpacks"
)

var (
	timeout         time.Duration
	restConfig      *rest.Config
	kubeClientset   *kubernetes.Clientset
	buildClientset  *buildClient.Clientset
	tektonClientset *tektonClient.Clientset
	buildInterface  buildv1beta1.BuildInterface
	imageRegistry   string
	imagePushSecret *string
	namespace       string
	sourcePath      string
)

var (
	_ = BeforeSuite(func(ctx SpecContext) {
		//TODO: Add more checks about operator health
		Expect(Setup(ctx)).To(Succeed(), "failed to setup environment")
		Expect(CheckStrategy(ctx)).To(Succeed(), "failed to validate build strategy")
		DeferCleanup(func() {
			Expect(Cleanup(context.TODO())).To(Succeed(), "failed to cleanup environment")
		})
	})
)

func Setup(ctx context.Context) (err error) {
	// Get Kubernetes cluster config
	kubeConfig := os.Getenv("KUBECONFIG")
	if kubeConfig == "" {
		kubeConfig = filepath.Join(os.Getenv("HOME"), ".kube", "config")
	}

	restConfig, err = clientcmd.BuildConfigFromFlags("", kubeConfig)
	if err != nil {
		restConfig, err = rest.InClusterConfig()
		if err != nil {
			return err
		}
	}

	// Setup Kubernetes client
	kubeClientset, err = kubernetes.NewForConfig(restConfig)
	if err != nil {
		return err
	}

	// Setup Shipwright Build client
	buildClientset, err = buildClient.NewForConfig(restConfig)
	if err != nil {
		return err
	}

	// Setup Tekton client
	tektonClientset, err = tektonClient.NewForConfig(restConfig)
	if err != nil {
		return err
	}

	// Use predefined namespace or create one
	if namespace = os.Getenv("TEST_NAMESPACE"); namespace == "" {
		namespace = fmt.Sprintf("%s-%s", NamespacePrefix, rand.String(5))
	}

	_, err = kubeClientset.CoreV1().Namespaces().Create(ctx, &v1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: namespace,
		},
	}, metav1.CreateOptions{})
	if err != nil {
		return err
	}

	// Use defined registry or use OpenShift Image registry
	if imageRegistry = os.Getenv("IMAGE_REGISTRY"); imageRegistry == "" {
		if imageRegistry, err = url.JoinPath(imageRegistry, OpenShiftImageRegistry, namespace); err != nil {
			return err
		}
	}

	// Set source path
	sourcePath = filepath.Join("..", "data", "samples")

	return
}

func CheckStrategy(ctx context.Context) error {
	strategyList := []string{BuildahStrategyName, SourceToImageStrategyName, BuildpackStrategyName}
	for _, strategy := range strategyList {
		_, err := buildClientset.ShipwrightV1beta1().ClusterBuildStrategies().Get(ctx, strategy, metav1.GetOptions{})
		if err != nil {
			return err
		}

	}
	return nil
}

func Cleanup(ctx context.Context) error {
	err := kubeClientset.CoreV1().Namespaces().Delete(ctx, namespace, metav1.DeleteOptions{})
	if err != nil {
		return err
	}
	return nil
}

func TestE2E(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "E2E Test Suite")
}
