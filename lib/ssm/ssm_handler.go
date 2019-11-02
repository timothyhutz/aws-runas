package ssm

import (
	"encoding/json"
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/client"
	"github.com/aws/aws-sdk-go/service/ssm"
	"github.com/aws/aws-sdk-go/service/ssm/ssmiface"
	"os"
	"os/exec"
)

type sessionHandler struct {
	client   ssmiface.SSMAPI
	cfg      aws.Config
	log      aws.Logger
	region   string
	endpoint string
	testing  bool
}

// NewSsmHandler creates the handler type needed to create shell and port-forwarding sessions
func NewSsmHandler(c client.ConfigProvider) *sessionHandler {
	s := ssm.New(c)
	return &sessionHandler{client: s, cfg: s.Config, region: s.SigningRegion, endpoint: s.Endpoint, log: aws.NewDefaultLogger()}
}

// WithLogger is a fluent method used with NewSsmHandler to configure a conforming logging type
func (h *sessionHandler) WithLogger(l aws.Logger) *sessionHandler {
	h.log = l
	return h
}

// StartSession will initiate an SSM shell session with the provided target EC2 instance
func (h *sessionHandler) StartSession(target string) error {
	in := ssm.StartSessionInput{Target: aws.String(target)}

	c, err := h.cmd(&in)
	if err != nil {
		return err
	}

	if h.testing {
		return nil
	}
	return c.Run()
}

// ForwardPort will open an SSM port-forwarding session with the provided target EC2 instance using the
// provided local and remote ports (lp and rp, respectively).  If lp is 0, a random, open local port is chosen.
func (h *sessionHandler) ForwardPort(target, lp, rp string) error {
	params := map[string][]*string{
		"localPortNumber": {aws.String(lp)},
		"portNumber":      {aws.String(rp)},
	}

	in := ssm.StartSessionInput{
		DocumentName: aws.String("AWS-StartPortForwardingSession"),
		Target:       aws.String(target),
		Parameters:   params,
	}

	c, err := h.cmd(&in)
	if err != nil {
		return err
	}

	if h.testing {
		return nil
	}
	return c.Run()
}

func (h *sessionHandler) cmd(input *ssm.StartSessionInput) (*exec.Cmd, error) {
	out, err := h.client.StartSession(input)
	if err != nil {
		return nil, err
	}

	outJ, err := json.Marshal(out)
	if err != nil {
		return nil, err
	}

	inJ, err := json.Marshal(input)
	if err != nil {
		return nil, err
	}

	c := exec.Command("session-manager-plugin", string(outJ), h.region, "StartSession", "", string(inJ), h.endpoint)
	c.Stdin = os.Stdin
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr

	h.debug("COMMAND: %s", c.String())
	return c, nil
}

func (h *sessionHandler) debug(f string, msg ...interface{}) {
	if h.cfg.LogLevel.AtLeast(aws.LogDebug) {
		h.log.Log(fmt.Sprintf(f, msg...))
	}
}
