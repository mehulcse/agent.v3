package cmdvalidateconfig

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"time"

	"github.com/pinpt/agent/pkg/gitclone"

	"github.com/pinpt/agent/cmd/cmdintegration"
)

type Opts struct {
	cmdintegration.Opts
	Output io.Writer
}

type AgentConfig = cmdintegration.AgentConfig

func Run(opts Opts) error {
	exp, err := newValidator(opts)
	if err != nil {
		return err
	}
	return exp.Destroy()
}

type validator struct {
	*cmdintegration.Command

	Opts Opts

	integration cmdintegration.Integration
}

func newValidator(opts Opts) (_ *validator, rerr error) {
	s := &validator{}
	if len(opts.Integrations) != 1 {
		panic("pass exactly 1 integration")
	}

	var err error
	s.Command, err = cmdintegration.NewCommand(opts.Opts)
	if err != nil {
		rerr = err
		return
	}
	s.Opts = opts

	err = s.SetupIntegrations(nil)
	if err != nil {
		err := s.outputErr(err)
		if err != nil {
			rerr = err
			return
		}
		return
	}

	s.integration = s.OnlyIntegration()

	err = s.runValidateAndPrint()
	if err != nil {
		rerr = err
		return
	}

	return s, nil
}

type Result struct {
	Errors []string `json:"errors"`
	// Success is true if there are no errors. Useful when returning result as json to ensure that marshalling worked.
	Success       bool   `json:"success"`
	ServerVersion string `json:"server_version"`
}

func (s *validator) runValidateAndPrint() error {
	serverVersion, errs := s.runValidate()
	return s.output(serverVersion, errs)
}

func (s *validator) outputErr(err error) error {
	return s.output("", []string{err.Error()})
}

func (s *validator) output(serverVersion string, errs []string) error {
	res := Result{}
	res.Errors = errs
	res.ServerVersion = serverVersion

	if len(res.Errors) == 0 {
		res.Success = true
	}

	b, err := json.Marshal(res)
	if err != nil {
		return err
	}

	s.Logger.Info("validate-config completed", "errors", res.Errors)

	_, err = s.Opts.Output.Write(b)
	if err != nil {
		return err
	}

	// BUG: last log message is missing without this
	time.Sleep(100 * time.Millisecond)
	return nil
}

func (s *validator) runValidate() (serverVersion string, errs []string) {
	ctx := context.Background()
	client := s.integration.ILoader.RPCClient()

	rerr := func(err error) {
		errs = append(errs, err.Error())
	}

	res0, err := client.ValidateConfig(ctx, s.integration.ExportConfig)
	if err != nil {
		_ = s.CloseOnlyIntegrationAndHandlePanic(s.integration.ILoader)
		rerr(err)
		return
	}

	if res0.RepoURL != "" { // repo url is only set for git integrations
		err = s.cloneRepo(res0.RepoURL)
		if err != nil {
			rerr(err)
			return
		}
	}

	err = s.CloseOnlyIntegrationAndHandlePanic(s.integration.ILoader)
	if err != nil {
		rerr(err)
		return
	}

	return res0.ServerVersion, res0.Errors
}

func (s *validator) cloneRepo(url string) error {
	urlWithoutCreds, err := urlWithoutCreds(url)
	if err != nil {
		return fmt.Errorf("url passed to git clone validation is not valid, err: %v", err)
	}

	s.Logger.Info("git clone validation start", "url", urlWithoutCreds)

	err = gitclone.TestClone(s.Logger, url, s.Locs.Temp)
	if err != nil {
		return fmt.Errorf("git clone validation failed. url: %v err: %v", urlWithoutCreds, err)
	}

	s.Logger.Info("git clone validation success", "url", urlWithoutCreds)
	return nil
}

func urlWithoutCreds(urlStr string) (string, error) {
	u, err := url.Parse(urlStr)
	if err != nil {
		return "", err
	}
	u.User = nil
	return u.String(), nil
}

func (s *validator) Destroy() error {
	return nil
}
