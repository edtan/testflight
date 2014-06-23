package testflight_test

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"syscall"
	"testing"
	"time"

	WardenRunner "github.com/cloudfoundry-incubator/warden-linux/integration/runner"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
	"github.com/tedsuo/ifrit"
	"github.com/tedsuo/ifrit/grouper"

	"github.com/concourse/testflight/runner"
)

var processes ifrit.Process
var fixturesDir = "./fixtures"

var builtComponents map[string]string

var wardenBinPath string

var _ = SynchronizedBeforeSuite(func() []byte {
	wardenBinPath = os.Getenv("WARDEN_BINPATH")
	Ω(wardenBinPath).ShouldNot(BeEmpty(), "must provide $WARDEN_BINPATH")

	Ω(os.Getenv("BASE_GOPATH")).ShouldNot(BeEmpty(), "must provide $BASE_GOPATH")

	turbineBin, err := buildWithGodeps("github.com/concourse/turbine", "-race")
	Ω(err).ShouldNot(HaveOccurred())

	atcBin, err := buildWithGodeps("github.com/concourse/atc", "-race")
	Ω(err).ShouldNot(HaveOccurred())

	gliderBin, err := buildWithGodeps("github.com/concourse/glider", "-race")
	Ω(err).ShouldNot(HaveOccurred())

	flyBin, err := buildWithGodeps("github.com/concourse/fly", "-race")
	Ω(err).ShouldNot(HaveOccurred())

	wardenLinuxBin, err := buildWithGodeps("github.com/cloudfoundry-incubator/warden-linux", "-race")
	Ω(err).ShouldNot(HaveOccurred())

	components, err := json.Marshal(map[string]string{
		"turbine":      turbineBin,
		"atc":          atcBin,
		"glider":       gliderBin,
		"fly":          flyBin,
		"warden-linux": wardenLinuxBin,
	})
	Ω(err).ShouldNot(HaveOccurred())

	return components
}, func(components []byte) {
	err := json.Unmarshal(components, &builtComponents)
	Ω(err).ShouldNot(HaveOccurred())
})

var _ = BeforeEach(func() {
	externalAddr := os.Getenv("EXTERNAL_ADDRESS")
	Ω(externalAddr).ShouldNot(BeEmpty(), "must specify $EXTERNAL_ADDRESS")

	wardenRunner := WardenRunner.New(
		builtComponents["warden-linux"],
		wardenBinPath,
		"bogus/rootfs",
	)

	turbineRunner := runner.NewRunner(
		builtComponents["turbine"],
		"-wardenNetwork", wardenRunner.Network(),
		"-wardenAddr", wardenRunner.Addr(),
		"-resourceTypes", `{"raw":"concourse/raw-resource#dev"}`,
	)

	gliderRunner := runner.NewRunner(
		builtComponents["glider"],
		"-peerAddr", externalAddr+":5637",
	)

	processes = grouper.EnvokeGroup(grouper.RunGroup{
		"turbine": turbineRunner,
		//"atc":      runner.NewRunner(builtComponents["atc"]),
		"glider":       gliderRunner,
		"warden-linux": wardenRunner,
	})

	Consistently(processes.Wait(), 5*time.Second).ShouldNot(Receive())

	os.Setenv("GLIDER_URL", "http://127.0.0.1:5637")
})

var _ = AfterEach(func() {
	processes.Signal(syscall.SIGINT)
	Eventually(processes.Wait(), 10).Should(Receive())
})

func TestFlightTest(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "FlightTest Suite")
}

func findSource(pkg string) string {
	for _, path := range filepath.SplitList(os.Getenv("GOPATH")) {
		srcPath := filepath.Join(path, "src", pkg)

		_, err := os.Stat(srcPath)
		if err != nil {
			continue
		}

		return srcPath
	}

	return ""
}

func buildWithGodeps(pkg string, args ...string) (string, error) {
	srcPath := findSource(pkg)
	Ω(srcPath).ShouldNot(BeEmpty(), "could not find source for "+pkg)

	gopath := fmt.Sprintf(
		"%s%c%s",
		filepath.Join(srcPath, "Godeps", "_workspace"),
		os.PathListSeparator,
		os.Getenv("BASE_GOPATH"),
	)

	return gexec.BuildIn(gopath, pkg, args...)
}