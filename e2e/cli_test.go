package e2e

import (
	"context"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega" //nolint:golint,revive
	buildv1beta1 "github.com/shipwright-io/build/pkg/apis/build/v1beta1"
	clibuild "github.com/shipwright-io/cli/pkg/shp/cmd/build"
	cliparams "github.com/shipwright-io/cli/pkg/shp/params"
	"io"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/cli-runtime/pkg/genericiooptions"
	"k8s.io/utils/ptr"
	"net/url"
	"os"
	"path/filepath"
)

var _ = When("Building from local source", Ordered, Label("local-source-build"), func() {
	var (
		ctx   context.Context
		build *buildv1beta1.Build
	)

	BeforeAll(func() {
		ctx = context.Background()
		DeferCleanup(func() {
			err := buildClientset.ShipwrightV1beta1().Builds(build.Namespace).Delete(context.TODO(), build.Name, metav1.DeleteOptions{})
			Expect(err).To(Succeed(), "failed to delete build")
		})
	})

	It("should be able to create Build resource ", func() {
		image, err := url.JoinPath(imageRegistry, "local-source")
		Expect(err).NotTo(HaveOccurred(), "failed to get image registry")
		build = &buildv1beta1.Build{
			ObjectMeta: metav1.ObjectMeta{
				GenerateName: "local-source-",
				Namespace:    namespace,
			},
			Spec: buildv1beta1.BuildSpec{
				Strategy: buildv1beta1.Strategy{
					Kind: ptr.To(buildv1beta1.ClusterBuildStrategyKind),
					Name: BuildahStrategyName,
				},
				Source: &buildv1beta1.Source{
					Type: buildv1beta1.LocalType,
				},
				Timeout: &metav1.Duration{Duration: timeout},
				Output: buildv1beta1.Image{
					Image:      image,
					PushSecret: imagePushSecret,
					Insecure:   ptr.To(true),
				},
			},
		}
		build, err = buildClientset.ShipwrightV1beta1().Builds(namespace).Create(ctx, build, metav1.CreateOptions{})
		Expect(err).ToNot(HaveOccurred(), "failed to create build")
	})

	It("should be able to upload source code from local and run build successfully", func() {
		os.Args = append([]string{
			"build",
			"upload",
			build.GetName(),
			filepath.Join(sourcePath, "buildah"),
		})

		configFlags := genericclioptions.NewConfigFlags(true)
		configFlags.Namespace = &namespace
		ioStreams := genericiooptions.IOStreams{In: os.Stdin, Out: io.Discard, ErrOut: os.Stderr}

		// TODO: Change this pattern of mocking in upstream.
		params := cliparams.NewParamsForTest(
			kubeClientset,
			buildClientset,
			configFlags,
			namespace,
			nil,
			nil,
		)

		c := clibuild.Command(params, &ioStreams)
		Expect(c.Execute()).ToNot(HaveOccurred(), "failed to upload and run build")
	})
})
