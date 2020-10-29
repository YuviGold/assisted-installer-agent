package subsystem

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/openshift/assisted-service/models"
)

const (
	stepNTPID        = "ntp-synchronizer-step"
	singleRunTimeout = 10 * time.Second
)

var _ = Describe("NTP tests", func() {
	var (
		hostID string
	)

	BeforeEach(func() {
		resetAll()
		hostID = nextHostID()
	})

	It("no_sources", func() {
		setNTPSyncRequestStub(hostID, models.NtpSynchronizationRequest{})

		ntpResponse := getNTPResponse(hostID)
		Expect(ntpResponse).ShouldNot(BeNil())
	})

	It("add_server", func() {
		sourceName := "0.asia.pool.ntp.org"

		setNTPSyncRequestStub(hostID, models.NtpSynchronizationRequest{
			NtpSource: &sourceName,
		})

		ntpResponse := getNTPResponse(hostID)
		Expect(ntpResponse).ShouldNot(BeNil())
	})
})

func setNTPSyncRequestStub(hostID string, request models.NtpSynchronizationRequest) {
	_, err := addRegisterStub(hostID, http.StatusCreated)
	Expect(err).NotTo(HaveOccurred())

	b, err := json.Marshal(&request)
	Expect(err).ShouldNot(HaveOccurred())

	_, err = addNextStepStub(hostID, 100,
		&models.Step{
			StepType: models.StepTypeNtpSynchronizer,
			StepID:   stepNTPID,
			Command:  "docker",
			Args: []string{
				"run", "--privileged", "--net=host", "--rm",
				"-v", "/var/log:/var/log",
				"-v", "/run/systemd/journal/socket:/run/systemd/journal/socket",
				"quay.io/ocpmetal/assisted-installer-agent:latest",
				"ntp_synchronizer",
				string(b),
			},
		},
	)
	Expect(err).NotTo(HaveOccurred())
}

func getNTPResponse(hostID string) *models.NtpSynchronizationResponse {
	setReplyStartAgent(hostID)
	Eventually(func() bool {
		return isReplyFound(hostID, &NTPSynchronizerVerifier{})
	}, singleRunTimeout, 5*time.Second).Should(BeTrue())

	stepReply := getSpecificStep(hostID, &NTPSynchronizerVerifier{})
	return getNTPResponseFromStepReply(stepReply)
}

type NTPSynchronizerVerifier struct{}

func (i *NTPSynchronizerVerifier) verify(actualReply *models.StepReply) bool {
	if actualReply.ExitCode != 0 {
		log.Errorf("NTPSynchronizerVerifier returned with exit code %d. error: %s", actualReply.ExitCode, actualReply.Error)
		return false
	}
	if actualReply.StepType != models.StepTypeNtpSynchronizer {
		log.Errorf("NTPSynchronizerVerifier invalid step replay %s", actualReply.StepType)
		return false
	}
	var response models.NtpSynchronizationResponse
	err := json.Unmarshal([]byte(actualReply.Output), &response)
	if err != nil {
		log.Errorf("NTPSynchronizerVerifier failed to unmarshal")
		return false
	}

	for _, source := range response.NtpSources {
		fmt.Println(*source)
		if source.SourceState == models.SourceStateSynced {
			return true
		}
	}

	return false
}

func getNTPResponseFromStepReply(actualReply *models.StepReply) *models.NtpSynchronizationResponse {
	var response models.NtpSynchronizationResponse
	err := json.Unmarshal([]byte(actualReply.Output), &response)
	Expect(err).NotTo(HaveOccurred())
	return &response
}
