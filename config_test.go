package phpfpm_test

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	phpfpm "github.com/initializ-buildpacks/php-fpm"
	"github.com/sclevine/spec"

	. "github.com/onsi/gomega"
)

func testConfig(t *testing.T, context spec.G, it spec.S) {
	var (
		Expect = NewWithT(t).Expect

		layerDir    string
		phpDistPath string
		workingDir  string
		cnbDir      string
		config      phpfpm.Config
	)

	it.Before(func() {
		var err error
		layerDir, err = os.MkdirTemp("", "php-fpm-layer")
		Expect(err).NotTo(HaveOccurred())
		Expect(os.Chmod(layerDir, os.ModePerm)).To(Succeed())

		phpDistPath = "some-dist-path"

		workingDir, err = os.MkdirTemp("", "workingDir")
		Expect(err).NotTo(HaveOccurred())

		cnbDir, err = os.MkdirTemp("", "cnb")
		Expect(err).NotTo(HaveOccurred())
		Expect(os.Chmod(layerDir, os.ModePerm)).To(Succeed())

		Expect(os.MkdirAll(filepath.Join(cnbDir, "config"), os.ModePerm)).To(Succeed())
		Expect(os.WriteFile(filepath.Join(cnbDir, "config", "php-fpm-base.conf"), []byte(`
include = {{ .PhpDistribution }}
include = {{ .PhpFpmBuildpack }}
include = {{ .OtherBuildpacks }}
include = {{ .UserInclude }}
`), os.ModePerm)).To(Succeed())
		Expect(os.WriteFile(filepath.Join(cnbDir, "config", "php-fpm-buildpack.conf"), nil, os.ModePerm)).To(Succeed())

		config = phpfpm.NewConfig()
	})

	it.After(func() {
		Expect(os.RemoveAll(layerDir)).To(Succeed())
		Expect(os.RemoveAll(cnbDir)).To(Succeed())
		Expect(os.RemoveAll(workingDir)).To(Succeed())
	})

	it("writes a php-fpm.conf file into layerDir based on the template", func() {
		path, err := config.Write(layerDir, phpDistPath, workingDir, cnbDir)
		Expect(err).NotTo(HaveOccurred())
		Expect(path).To(Equal(filepath.Join(layerDir, "base.conf")))

		Expect(filepath.Join(layerDir, "base.conf")).To(BeARegularFile())

		contents, err := os.ReadFile(filepath.Join(layerDir, "base.conf"))
		Expect(err).NotTo(HaveOccurred())
		Expect(string(contents)).To(ContainSubstring(fmt.Sprintf("include = %s/php-fpm.d/www.conf.default", phpDistPath)))
		Expect(string(contents)).To(ContainSubstring(fmt.Sprintf("include = %s/buildpack.conf", layerDir)))
		Expect(string(contents)).To(ContainSubstring(fmt.Sprintf("include = %s/.php.fpm.bp/*.conf", workingDir)))
		Expect(string(contents)).NotTo(ContainSubstring(fmt.Sprintf("include = %s/.php.fpm.d/*.conf", workingDir)))
	})

	context("there's no PHP distribution config", func() {
		it("the base conf does not contain the associated include", func() {
			path, err := config.Write(layerDir, "", workingDir, cnbDir)
			Expect(err).NotTo(HaveOccurred())
			Expect(path).To(Equal(filepath.Join(layerDir, "base.conf")))

			contents, err := os.ReadFile(filepath.Join(layerDir, "base.conf"))
			Expect(err).NotTo(HaveOccurred())
			Expect(string(contents)).NotTo(ContainSubstring("php-fpm.d/www.conf.default"))
		})
	})

	context("there's a user-provided config", func() {
		it.Before(func() {
			Expect(os.MkdirAll(filepath.Join(workingDir, ".php.fpm.d"), os.ModePerm)).To(Succeed())
			Expect(os.WriteFile(filepath.Join(workingDir, ".php.fpm.d", "fpm.conf"), nil, os.ModePerm)).To(Succeed())
		})
		it.After(func() {
			Expect(os.RemoveAll(filepath.Join(workingDir, ".php.fpm.d"))).To(Succeed())
		})

		it("the base conf contains the path to it", func() {
			path, err := config.Write(layerDir, phpDistPath, workingDir, cnbDir)
			Expect(err).NotTo(HaveOccurred())
			Expect(path).To(Equal(filepath.Join(layerDir, "base.conf")))

			contents, err := os.ReadFile(filepath.Join(layerDir, "base.conf"))
			Expect(err).NotTo(HaveOccurred())
			Expect(string(contents)).To(ContainSubstring(fmt.Sprintf("include = %s/.php.fpm.d/*.conf", workingDir)))
		})
	})

	context("failure cases", func() {
		context("when copying buildpack config fails", func() {
			it("returns an error", func() {
				_, err := config.Write(layerDir, phpDistPath, workingDir, "some-fake-path")
				Expect(err).To(MatchError(ContainSubstring("no such file or directory")))
			})
		})
		context("when template is not parseable", func() {
			it.Before(func() {
				Expect(os.WriteFile(filepath.Join(cnbDir, "config", "php-fpm-base.conf"), []byte(`
		include = {{ .PhpDistribution
		`), os.ModePerm)).To(Succeed())
			})
			it("returns an error", func() {
				_, err := config.Write(layerDir, phpDistPath, workingDir, cnbDir)
				Expect(err).To(MatchError(ContainSubstring("unclosed action")))
			})
		})

		context("the workspace/php.fpm.bp directory cannot be created", func() {
			it.Before(func() {
				Expect(os.Chmod(workingDir, 0000)).To(Succeed())
			})
			it.After(func() {
				Expect(os.Chmod(workingDir, 0644)).To(Succeed())
			})
			it("returns an error", func() {
				_, err := config.Write(layerDir, phpDistPath, workingDir, cnbDir)
				Expect(err).To(MatchError(ContainSubstring(fmt.Sprintf("failed to create %s/.php.fpm.bp:", workingDir))))
			})
		})

		context("when conf file can't be opened for writing", func() {
			it.Before(func() {
				Expect(os.WriteFile(filepath.Join(layerDir, "base.conf"), nil, 0400)).To(Succeed())
			})
			it("returns an error", func() {
				_, err := config.Write(layerDir, phpDistPath, workingDir, cnbDir)
				Expect(err).To(MatchError(ContainSubstring("permission denied")))
			})
		})
	})
}
