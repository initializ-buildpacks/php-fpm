package phpfpm_test

import (
	"os"
	"testing"

	"github.com/paketo-buildpacks/packit/v2"
	phpfpm "github.com/initializ-buildpacks/php-fpm"
	"github.com/sclevine/spec"

	. "github.com/onsi/gomega"
)

func testDetect(t *testing.T, context spec.G, it spec.S) {
	var (
		Expect = NewWithT(t).Expect

		workingDir string
		detect     packit.DetectFunc
	)

	it.Before(func() {
		var err error
		workingDir, err = os.MkdirTemp("", "working-dir")
		Expect(err).NotTo(HaveOccurred())

		detect = phpfpm.Detect()
	})

	it.After(func() {
		Expect(os.RemoveAll(workingDir)).To(Succeed())
	})

	context("in all cases", func() {
		it("requires php and provides php-fpm", func() {
			result, err := detect(packit.DetectContext{
				WorkingDir: workingDir,
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(result.Plan).To(Equal(packit.BuildPlan{
				Requires: []packit.BuildPlanRequirement{
					{
						Name: phpfpm.PhpDist,
						Metadata: map[string]interface{}{
							"build":  true,
							"launch": true,
						},
					},
				},
				Provides: []packit.BuildPlanProvision{
					{
						Name: phpfpm.PhpFpmDependency,
					},
				},
			}))
		})
	})
}
