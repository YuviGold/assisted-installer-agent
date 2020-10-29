package commands

import (
	"fmt"
	"io/ioutil"
	"regexp"

	"github.com/openshift/assisted-installer-agent/src/util"
	"github.com/pkg/errors"
)

const defaultChronyConf = "/etc/chrony.conf"

func allowChronyConf(content string) string {
	/*
		allow - client access
		cmdallow - monitoring access
		bindcmdaddress - listens only to loopback interface by default.
	*/

	// Delete previous configuration
	c := regexp.MustCompile(`(\s+)(allow|cmdallow|bindcmdaddress)\s+.*`)
	conf := c.ReplaceAllString(content, "")

	conf += `
cmdallow all
allow all
bindcmdaddress 0.0.0.0
bindcmdaddress ::
`

	return conf
}

func ChronyAllowAll(confFile string) error {
	bytes, err := ioutil.ReadFile(confFile)

	if err != nil {
		return errors.Wrapf(err, fmt.Sprintf("Could not read conf file %s", confFile))
	}

	conf := allowChronyConf(string(bytes))

	err = ioutil.WriteFile(confFile, []byte(conf), 0o644)

	if err != nil {
		return errors.Wrapf(err, fmt.Sprintf("Could not write conf file %s", confFile))
	}

	return nil
}

func StartNTPDaemon() (stdout string, stderr string, exitCode int) {
	err := ChronyAllowAll(defaultChronyConf)

	if err != nil {
		return "", err.Error(), -1
	}

	return util.Execute("chronyd")
}
