package aspnet

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/cloudfoundry/libcfbuildpack/buildpackplan"
	"github.com/cloudfoundry/libcfbuildpack/layers"
	"github.com/cloudfoundry/libcfbuildpack/test"
	. "github.com/onsi/gomega"
	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"
)

func TestUnitDotnet(t *testing.T) {
	spec.Run(t, "Detect", testDotnet, spec.Report(report.Terminal{}))
}

func testDotnet(t *testing.T, when spec.G, it spec.S) {
	var (
		factory                 *test.BuildFactory
		stubDotnetAspnetFixture = filepath.Join("testdata", "stub-dotnet-aspnet.tar.xz")
		symlinkPath             string
		symlinkLayer            layers.Layer
	)

	it.Before(func() {
		var err error

		RegisterTestingT(t)
		factory = test.NewBuildFactory(t)
		factory.AddDependencyWithVersion(DotnetAspNet, "2.2.5", stubDotnetAspnetFixture)
		symlinkLayer = factory.Build.Layers.Layer("aspnet-symlinks")

		symlinkPath, err = ioutil.TempDir(os.TempDir(), "runtime")
		Expect(err).ToNot(HaveOccurred())

		os.Setenv("DOTNET_ROOT", symlinkPath)
	})

	it.After(func() {
		os.RemoveAll(symlinkPath)
		os.Unsetenv("DOTNET_ROOT")
	})

	when("runtime.NewContributor", func() {
		when("when there is no buildpack.yml", func() {
			it("returns true if a build plan exists and matching version is found", func() {
				factory.AddPlan(buildpackplan.Plan{Name: DotnetAspNet, Version: "2.2.5"})
				factory.AddPlan(buildpackplan.Plan{Name: DotnetAspNet, Version: "2.2.5"})

				contributor, willContribute, err := NewContributor(factory.Build)
				Expect(err).NotTo(HaveOccurred())
				Expect(willContribute).To(BeTrue())
				Expect(contributor.aspnetLayer.Dependency.Version.String()).To(Equal("2.2.5"))
			})

			it("returns true if a build plan exists and no matching version is found", func() {
				factory.AddPlan(buildpackplan.Plan{Name: DotnetAspNet, Version: "1.0.0"})

				_, willContribute, err := NewContributor(factory.Build)
				Expect(err).To(HaveOccurred())
				Expect(willContribute).To(BeFalse())
			})
		})

		when("when there is a buildpack.yml", func() {
			it.Before(func() {
				factory.AddPlan(buildpackplan.Plan{Name: DotnetAspNet, Version: "2.1.0"})
				factory.AddDependencyWithVersion(DotnetAspNet, "2.1.5", stubDotnetAspnetFixture)
				factory.AddDependencyWithVersion(DotnetAspNet, "2.2.2", stubDotnetAspnetFixture)
			})
			it("that has a version range it returns the highest patch for that range", func() {
				test.WriteFile(t, filepath.Join(factory.Build.Application.Root, "buildpack.yml"), fmt.Sprintf("dotnet-framework:\n  version: %s", "2.2.*"))
				defer os.RemoveAll(filepath.Join(factory.Build.Application.Root, "buildpack.yml"))

				contributor, willContribute, err := NewContributor(factory.Build)
				Expect(err).NotTo(HaveOccurred())
				Expect(willContribute).To(BeTrue())
				Expect(contributor.aspnetLayer.Dependency.Version.String()).To(Equal("2.2.5"))
			})
			it("that has an exact version it only uses that exact version", func() {
				test.WriteFile(t, filepath.Join(factory.Build.Application.Root, "buildpack.yml"), fmt.Sprintf("dotnet-framework:\n  version: %s", "2.2.2"))
				defer os.RemoveAll(filepath.Join(factory.Build.Application.Root, "buildpack.yml"))

				contributor, willContribute, err := NewContributor(factory.Build)
				Expect(err).NotTo(HaveOccurred())
				Expect(willContribute).To(BeTrue())
				Expect(contributor.aspnetLayer.Dependency.Version.String()).To(Equal("2.2.2"))
			})
		})

		it("returns false if a build plan does not exist", func() {
			contributor, willContribute, err := NewContributor(factory.Build)
			Expect(err).NotTo(HaveOccurred())
			Expect(willContribute).To(BeFalse())
			Expect(contributor).To(Equal(Contributor{}))
		})
	})

	when("Contribute", func() {
		it("installs the aspnet dependency, writes dotnet root", func() {
			factory.AddPlan(buildpackplan.Plan{Name: DotnetAspNet})

			dotnetASPNetContributor, _, err := NewContributor(factory.Build)
			Expect(err).NotTo(HaveOccurred())

			Expect(dotnetASPNetContributor.Contribute()).To(Succeed())

			layer := factory.Build.Layers.Layer(DotnetAspNet)
			Expect(filepath.Join(layer.Root, "stub-dir", "stub.txt")).To(BeARegularFile())
			Expect(symlinkLayer).To(test.HaveOverrideSharedEnvironment("DOTNET_ROOT", symlinkLayer.Root))
		})

		it("contributes dotnet runtime to the build layer when included in the build plan", func() {
			factory.AddPlan(buildpackplan.Plan{
				Name: DotnetAspNet,
				Metadata: buildpackplan.Metadata{
					"build": true,
				},
			})

			dotnetASPNetContributor, _, err := NewContributor(factory.Build)
			Expect(err).NotTo(HaveOccurred())

			Expect(dotnetASPNetContributor.Contribute()).To(Succeed())

			layer := factory.Build.Layers.Layer(DotnetAspNet)
			Expect(layer).To(test.HaveLayerMetadata(true, false, false))
		})

		it("contributes dotnet runtime to the launch layer when included in the build plan", func() {
			factory.AddPlan(buildpackplan.Plan{
				Name: DotnetAspNet,
				Metadata: buildpackplan.Metadata{
					"launch": true,
				},
			})

			dotnetASPNetContributor, _, err := NewContributor(factory.Build)
			Expect(err).NotTo(HaveOccurred())

			Expect(dotnetASPNetContributor.Contribute()).To(Succeed())

			layer := factory.Build.Layers.Layer(DotnetAspNet)
			Expect(layer).To(test.HaveLayerMetadata(false, false, true))
		})
	})
}
