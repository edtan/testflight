package git_pipeline_test

import (
	"github.com/concourse/testflight/gitserver"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
	"github.com/onsi/gomega/gexec"
)

var _ = Describe("A job with a task with a timeout", func() {
	var originGitServer *gitserver.Server

	BeforeEach(func() {
		originGitServer = gitserver.Start(gitServerRootfs, gardenClient)

		configurePipeline(
			"-c", "fixtures/timeout.yml",
			"-v", "origin-git-server="+originGitServer.URI(),
		)

		originGitServer.Commit()
	})

	AfterEach(func() {
		originGitServer.Stop()
	})

	It("enforces the timeout", func() {
		By("not aborting if the step completes in time")
		watch := flyWatch("duration-successful-job")
		Ω(watch).Should(gbytes.Say("initializing"))
		Ω(watch).Should(gbytes.Say("passing-task succeeded"))
		Ω(watch).Should(gexec.Exit(0))

		By("aborting when the step takes too long")
		watch = flyWatch("duration-fail-job")
		Ω(watch).Should(gbytes.Say("initializing"))
		Ω(watch).Should(gbytes.Say("interrupted"))
		Ω(watch).Should(gexec.Exit(1))
	})
})
