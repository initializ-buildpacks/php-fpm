package phpfpm_test

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/paketo-buildpacks/packit/v2"
	"github.com/paketo-buildpacks/packit/v2/scribe"
	phpfpm "github.com/initializ-buildpacks/php-fpm"
	"github.com/initializ-buildpacks/php-fpm/fakes"
	"github.com/sclevine/spec"

	. "github.com/onsi/gomega"
)

func testBuild(t *testing.T, context spec.G, it spec.S) {
	var (
		Expect = NewWithT(t).Expect

		layersDir  string
		workingDir string
		cnbDir     string

		buffer *bytes.Buffer
		config *fakes.ConfigWriter

		buildContext packit.BuildContext
		build        packit.BuildFunc
	)

	it.Before(func() {
		var err error
		layersDir, err = os.MkdirTemp("", "layers")
		Expect(err).NotTo(HaveOccurred())

		cnbDir, err = os.MkdirTemp("", "cnb")
		Expect(err).NotTo(HaveOccurred())

		workingDir, err = os.MkdirTemp("", "working-dir")
		Expect(err).NotTo(HaveOccurred())

		buffer = bytes.NewBuffer(nil)
		logEmitter := scribe.NewEmitter(buffer)

		config = &fakes.ConfigWriter{}
		config.WriteCall.Returns.String = "some-path"

		Expect(os.Setenv("PHPRC", "some-php-dist-path")).To(Succeed())

		buildContext = packit.BuildContext{
			WorkingDir: workingDir,
			CNBPath:    cnbDir,
			Stack:      "some-stack",
			BuildpackInfo: packit.BuildpackInfo{
				Name:    "Some Buildpack",
				Version: "some-version",
			},
			Plan: packit.BuildpackPlan{
				Entries: []packit.BuildpackPlanEntry{
					{Name: "some-entry"},
				},
			},
			Layers: packit.Layers{Path: layersDir},
		}

		build = phpfpm.Build(config, logEmitter)
	})

	it.After(func() {
		Expect(os.RemoveAll(layersDir)).To(Succeed())
		Expect(os.RemoveAll(cnbDir)).To(Succeed())
		Expect(os.RemoveAll(workingDir)).To(Succeed())
		Expect(os.Unsetenv("PHPRC")).To(Succeed())
	})

	it("writes a config file into its layer and stores the location in an env var", func() {
		result, err := build(buildContext)
		Expect(err).NotTo(HaveOccurred())

		Expect(config.WriteCall.Receives.Layer).To(Equal(filepath.Join(layersDir, phpfpm.PhpFpmConfigLayerName)))
		Expect(config.WriteCall.Receives.PhpDistPath).To(Equal("some-php-dist-path"))
		Expect(config.WriteCall.Receives.WorkingDir).To(Equal(workingDir))
		Expect(config.WriteCall.Receives.CnbPath).To(Equal(cnbDir))

		Expect(result.Layers).To(HaveLen(1))
		Expect(result.Layers[0].Name).To(Equal("php-fpm-config"))
		Expect(result.Layers[0].Path).To(Equal(filepath.Join(layersDir, "php-fpm-config")))
		Expect(result.Layers[0].SharedEnv).To(Equal(packit.Environment{
			"PHP_FPM_PATH.default": "some-path",
		}))

		Expect(result.Layers[0].Build).To(BeFalse())
		Expect(result.Layers[0].Cache).To(BeFalse())
		Expect(result.Layers[0].Launch).To(BeFalse())
	})

	context("when php-fpm is required at build/launch time", func() {
		it.Before(func() {
			buildContext.Plan.Entries = append(buildContext.Plan.Entries, packit.BuildpackPlanEntry{
				Name: phpfpm.PhpFpmDependency,
				Metadata: map[string]interface{}{
					"build":  true,
					"launch": true,
				},
			})
		})

		it("makes the layer available at build and launch time", func() {
			result, err := build(buildContext)
			Expect(err).NotTo(HaveOccurred())

			Expect(result.Layers).To(HaveLen(1))
			Expect(result.Layers[0].Name).To(Equal("php-fpm-config"))
			Expect(result.Layers[0].Build).To(BeTrue())
			Expect(result.Layers[0].Cache).To(BeFalse())
			Expect(result.Layers[0].Launch).To(BeTrue())
		})
	})

	context("failure cases", func() {
		context("when config layer cannot be gotten", func() {
			it.Before(func() {
				err := os.WriteFile(filepath.Join(layersDir, fmt.Sprintf("%s.toml", phpfpm.PhpFpmConfigLayerName)), nil, 0000)
				Expect(err).NotTo(HaveOccurred())
			})
			it("returns an error", func() {
				_, err := build(buildContext)
				Expect(err).To(MatchError(ContainSubstring("failed to parse layer content metadata")))
			})
		})

		context("when config file cannot be written", func() {
			it.Before(func() {
				config.WriteCall.Returns.Error = errors.New("config writing error")
			})
			it("returns an error", func() {
				_, err := build(buildContext)
				Expect(err).To(MatchError(ContainSubstring("config writing error")))
			})
		})

		context("when config file cannot be written", func() {
			it.Before(func() {
				config.WriteCall.Returns.Error = errors.New("config writing error")
			})
			it("returns an error", func() {
				_, err := build(packit.BuildContext{
					WorkingDir: workingDir,
					CNBPath:    cnbDir,
					Stack:      "some-stack",
					BuildpackInfo: packit.BuildpackInfo{
						Name:    "Some Buildpack",
						Version: "some-version",
					},
					Plan: packit.BuildpackPlan{
						Entries: []packit.BuildpackPlanEntry{
							{Name: phpfpm.PhpFpmDependency},
						},
					},
					Layers: packit.Layers{Path: layersDir},
				})
				Expect(err).To(MatchError(ContainSubstring("config writing error")))
			})
		})
	})
}
