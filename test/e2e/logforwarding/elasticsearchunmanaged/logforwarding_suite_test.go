package elasticsearchunmanaged

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestLogForwarding(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "LogForwarding Integration E2E Suite - Unmanaged Elasticsearch")
}
