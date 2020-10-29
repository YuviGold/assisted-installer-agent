package ntp_synchronizer

import (
	"encoding/json"
	"strconv"
	"strings"

	"github.com/openshift/assisted-installer-agent/src/util"
	"github.com/openshift/assisted-service/models"
	"github.com/pkg/errors"

	"github.com/sirupsen/logrus"
)

const ChronyTimeoutSeconds = 10

//go:generate mockery -name INTPDependencies -inpkg
type INTPDependencies interface {
	Execute(command string, args ...string) (stdout string, stderr string, exitCode int)
}

type ProcessExecuter struct{}

func (e *ProcessExecuter) Execute(command string, args ...string) (stdout string, stderr string, exitCode int) {
	return util.Execute(command, args...)
}

func convertSourceState(val string) models.SourceState {
	switch val {
	case "*":
		return models.SourceStateSynced
	case "+":
		return models.SourceStateCombined
	case "-":
		return models.SourceStateNotCombined
	case "?":
		return models.SourceStateUnreachable
	case "x":
		return models.SourceStateError
	case "~":
		return models.SourceStateVariable
	default:
		return models.SourceStateError
	}
}

// func StartDaemon(e INTPDependencies) error {
// 	stdout, stderr, exitCode := e.Execute("chronyd")

// 	if exitCode == 0 {
// 		return nil
// 	} else {
// 		return errors.Errorf("chronyd exited with non-zero exit code %d: %s\n%s", exitCode, stdout, stderr)
// 	}
// }

func AddServer(e INTPDependencies, ntpSource string) error {
	stdout, stderr, exitCode := e.Execute("chronyc", "add", "server", ntpSource)

	if exitCode == 0 {
		return nil
	} else {
		return errors.Errorf("chronyc exited with non-zero exit code %d: %s\n%s", exitCode, stdout, stderr)
	}
}

func GetNTPSources(e INTPDependencies) (*[]*models.NtpSource, error) {
	stdout, stderr, exitCode := e.Execute("timeout", strconv.FormatInt(ChronyTimeoutSeconds, 10), "chronyc", "-c", "sources")

	sources := make([]*models.NtpSource, 0)

	for _, line := range strings.Split(stdout, "\n") {
		if line != "" {
			cols := strings.Split(line, ",")

			if len(cols) < 3 {
				continue
			}

			sources = append(sources, &models.NtpSource{SourceName: cols[2], SourceState: convertSourceState(cols[1])})
		}
	}

	switch exitCode {
	case 0:
		return &sources, nil
	case util.TimeoutExitCode:
		return nil, errors.Errorf("chronyc was timed out after %d seconds", ChronyTimeoutSeconds)
	default:
		return nil, errors.Errorf("chronyc exited with non-zero exit code %d: %s\n%s", exitCode, stdout, stderr)
	}
}

func NtpSync(ntpSyncRequestStr string, executer INTPDependencies, log logrus.FieldLogger) (stdout string, stderr string, exitCode int) {
	var ntpSyncRequest models.NtpSynchronizationRequest

	err := json.Unmarshal([]byte(ntpSyncRequestStr), &ntpSyncRequest)
	if err != nil {
		log.WithError(err).Errorf("NTPSync: json.Unmarshal")
		return "", err.Error(), -1
	}

	// err = StartDaemon(executer)

	// if err != nil {
	// 	log.WithError(err).Errorf("NTPSync: StartDaemon")
	// 	return "", err.Error(), -1
	// }

	if ntpSyncRequest.NtpSource != nil && *ntpSyncRequest.NtpSource != "" {
		err = AddServer(executer, *ntpSyncRequest.NtpSource)

		if err != nil {
			log.WithError(err).Errorf("NTPSync: GetNTPSources")
			return "", err.Error(), -1
		}
	}

	sources, err := GetNTPSources(executer)

	if err != nil {
		log.WithError(err).Errorf("NTPSync: GetNTPSources")
		return "", err.Error(), -1
	}

	var ntpSyncResponse models.NtpSynchronizationResponse = models.NtpSynchronizationResponse{
		NtpSources: *sources,
	}

	b, err := json.Marshal(&ntpSyncResponse)
	if err != nil {
		log.WithError(err).Error("NTPSync: json.Marshal")
		return "", err.Error(), -1
	}
	return string(b), "", 0
}
