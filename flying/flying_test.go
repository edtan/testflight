package flying_test

import (
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
	"github.com/onsi/gomega/gexec"
)

var _ = Describe("Flying", func() {
	var tmpdir string
	var fixture string

	BeforeEach(func() {
		var err error

		tmpdir, err = ioutil.TempDir("", "fly-test")
		Ω(err).ShouldNot(HaveOccurred())

		fixture = filepath.Join(tmpdir, "fixture")

		err = os.MkdirAll(fixture, 0755)
		Ω(err).ShouldNot(HaveOccurred())

		err = ioutil.WriteFile(
			filepath.Join(fixture, "run"),
			[]byte(`#!/bin/bash
echo some output
echo FOO is $FOO
echo ARGS are "$@"
exit 0
`),
			0755,
		)
		Ω(err).ShouldNot(HaveOccurred())

		err = ioutil.WriteFile(
			filepath.Join(fixture, "build.yml"),
			[]byte(`---
image: docker:///concourse/testflight-helper

params:
  FOO: 1

run:
  path: fixture/run
`),
			0644,
		)
		Ω(err).ShouldNot(HaveOccurred())
	})

	AfterEach(func() {
		os.RemoveAll(tmpdir)
	})

	It("works", func() {
		fly := exec.Command(flyBin, "--", "SOME", "ARGS")
		fly.Dir = fixture

		session, err := gexec.Start(fly, GinkgoWriter, GinkgoWriter)
		Ω(err).ShouldNot(HaveOccurred())

		Eventually(session, 1*time.Minute).Should(gexec.Exit(0))

		Ω(session).Should(gbytes.Say("some output"))
		Ω(session).Should(gbytes.Say("FOO is 1"))
		Ω(session).Should(gbytes.Say("ARGS are SOME ARGS"))
	})

	Describe("hijacking", func() {
		It("executes an interactive command in a running build's container", func() {
			err := ioutil.WriteFile(
				filepath.Join(fixture, "run"),
				[]byte(`#!/bin/bash
mkfifo /tmp/fifo
echo waiting
cat < /tmp/fifo
`),
				0755,
			)
			Ω(err).ShouldNot(HaveOccurred())

			fly := exec.Command(flyBin)
			fly.Dir = fixture

			flyS, err := gexec.Start(fly, GinkgoWriter, GinkgoWriter)
			Ω(err).ShouldNot(HaveOccurred())

			Eventually(flyS, 10*time.Second).Should(gbytes.Say("waiting"))

			hijack := exec.Command(flyBin, "hijack", "--", "bash", "-c", "echo marco > /tmp/fifo")

			hijackS, err := gexec.Start(hijack, GinkgoWriter, GinkgoWriter)
			Ω(err).ShouldNot(HaveOccurred())

			Eventually(flyS, 10*time.Second).Should(gbytes.Say("marco"))

			Eventually(hijackS, 5*time.Second).Should(gexec.Exit())

			Eventually(flyS, 5*time.Second).Should(gexec.Exit(0))
		})
	})

	Describe("aborting", func() {
		It("terminates the running build", func() {
			err := ioutil.WriteFile(
				filepath.Join(fixture, "run"),
				[]byte(`#!/bin/bash
trap "echo build got sigterm; exit 1" SIGTERM
sleep 1000 &
echo waiting
wait
`),
				0755,
			)
			Ω(err).ShouldNot(HaveOccurred())

			fly := exec.Command(flyBin)
			fly.Dir = fixture

			flyS, err := gexec.Start(fly, GinkgoWriter, GinkgoWriter)
			Ω(err).ShouldNot(HaveOccurred())

			Eventually(flyS, 10*time.Second).Should(gbytes.Say("waiting"))

			flyS.Signal(syscall.SIGTERM)

			Eventually(flyS, 10*time.Second).Should(gbytes.Say("build got sigterm"))

			// build should have been aborted
			Eventually(flyS, 5*time.Second).Should(gexec.Exit(3))
		})
	})
})