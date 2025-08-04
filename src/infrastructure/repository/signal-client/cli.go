package signal_client

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	logger "go-multi-chat-api/src/infrastructure/logger"
	utils2 "go-multi-chat-api/src/infrastructure/utils"
	"os/exec"
	"strings"
	"time"
)

type CliClient struct {
	signalCliMode      SignalCliMode
	signalCliApiConfig *utils2.SignalCliApiConfig
	Logger             *logger.Logger
}

func NewCliClient(signalCliMode SignalCliMode, signalCliApiConfig *utils2.SignalCliApiConfig, loggerInstance *logger.Logger) *CliClient {
	return &CliClient{
		signalCliMode:      signalCliMode,
		signalCliApiConfig: signalCliApiConfig,
		Logger:             loggerInstance,
	}
}

func stripInfoAndWarnMessages(input string) (string, string, string) {
	output := ""
	infoMessages := ""
	warnMessages := ""
	lines := strings.Split(input, "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "INFO") {
			if infoMessages != "" {
				infoMessages += "\n"
			}
			infoMessages += line
		} else if strings.HasPrefix(line, "WARN") {
			if warnMessages != "" {
				warnMessages += "\n"
			}
			warnMessages += line
		} else {
			if output != "" {
				output += "\n"
			}
			output += line
		}
	}
	return output, infoMessages, warnMessages
}

func (s *CliClient) Execute(wait bool, args []string, stdin string) (string, error) {
	containerId, err := getContainerId()
	s.Logger.Debug("If you want to run this command manually, run the following steps on your host system:")
	if err == nil {
		s.Logger.Debug(fmt.Sprintf("*) docker exec -it %s /bin/bash", containerId))
	} else {
		s.Logger.Debug("*) docker exec -it <container id> /bin/bash")
	}

	signalCliBinary := ""
	if s.signalCliMode == Normal {
		signalCliBinary = "signal-cli"
	} else if s.signalCliMode == Native {
		signalCliBinary = "signal-cli-native"
	} else {
		return "", errors.New("Invalid signal-cli mode")
	}

	//check if args contain number
	trustModeStr := ""
	for i, arg := range args {
		if (arg == "-a" || arg == "--account") && (((i + 1) < len(args)) && (utils2.IsPhoneNumber(args[i+1]))) {
			number := args[i+1]
			trustMode, err := s.signalCliApiConfig.GetTrustModeForNumber(number)
			if err == nil {
				trustModeStr, err = utils2.TrustModeToString(trustMode)
				if err != nil {
					trustModeStr = ""
					s.Logger.Error(fmt.Sprintf("Invalid trust mode: %s", trustModeStr))
				}
			}
			break
		}
	}

	if trustModeStr != "" {
		args = append([]string{"--trust-new-identities", trustModeStr}, args...)
	}

	fullCmd := ""
	if stdin != "" {
		fullCmd += "echo '" + stdin + "' | "
	}
	fullCmd += signalCliBinary + " " + strings.Join(args, " ")

	s.Logger.Debug("*) su signal-api")
	s.Logger.Debug(fmt.Sprintf("*) %s", fullCmd))

	cmdTimeout, err := utils2.GetIntEnv("SIGNAL_CLI_CMD_TIMEOUT", 120)
	if err != nil {
		s.Logger.Error("Env variable 'SIGNAL_CLI_CMD_TIMEOUT' contains an invalid timeout...falling back to default timeout (120 seconds)")
		cmdTimeout = 120
	}

	cmd := exec.Command(signalCliBinary, args...)
	if stdin != "" {
		cmd.Stdin = strings.NewReader(stdin)
	}
	if wait {
		var stdoutBuffer bytes.Buffer
		var stderrBuffer bytes.Buffer
		cmd.Stdout = &stdoutBuffer
		cmd.Stderr = &stderrBuffer

		err := cmd.Start()
		if err != nil {
			return "", err
		}

		done := make(chan error, 1)
		go func() {
			done <- cmd.Wait()
		}()
		select {
		case <-time.After(time.Duration(cmdTimeout) * time.Second):
			err := cmd.Process.Kill()
			if err != nil {
				return "", err
			}
			return "", errors.New("process killed as timeout reached")
		case err := <-done:
			if err != nil {
				combinedOutput := stdoutBuffer.String() + stderrBuffer.String()
				s.Logger.Debug(fmt.Sprintf("signal-cli output (stdout): %s", stdoutBuffer.String()))
				s.Logger.Debug(fmt.Sprintf("signal-cli output (stderr): %s", stderrBuffer.String()))
				return "", errors.New(combinedOutput)
			}
		}

		combinedOutput := stdoutBuffer.String() + stderrBuffer.String()
		s.Logger.Debug(fmt.Sprintf("signal-cli output (stdout): %s", stdoutBuffer.String()))
		s.Logger.Debug(fmt.Sprintf("signal-cli output (stderr): %s", stderrBuffer.String()))
		strippedOutput, infoMessages, warnMessages := stripInfoAndWarnMessages(combinedOutput)
		for _, line := range strings.Split(infoMessages, "\n") {
			if line != "" {
				s.Logger.Info(line)
			}
		}

		for _, line := range strings.Split(warnMessages, "\n") {
			if line != "" {
				s.Logger.Warn(line)
			}
		}

		return strippedOutput, nil
	} else {
		stdout, err := cmd.StdoutPipe()
		if err != nil {
			return "", err
		}
		cmd.Start()
		buf := bufio.NewReader(stdout) // Notice that this is not in a loop
		line, _, _ := buf.ReadLine()
		return string(line), nil
	}
}
