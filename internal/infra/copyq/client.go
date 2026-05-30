package copyq

import (
	"fmt"
	"os/exec"
)

type Client struct{}

func NewClient() *Client {
	return &Client{}
}

func (c *Client) Copy(value string) error {
	if out, err := exec.Command("copyq", "write", "0", "text/plain", value).CombinedOutput(); err != nil {
		return fmt.Errorf("copyq write: %s: %w", string(out), err)
	}
	if out, err := exec.Command("copyq", "select", "0").CombinedOutput(); err != nil {
		return fmt.Errorf("copyq select: %s: %w", string(out), err)
	}
	return nil
}

func (c *Client) CopySensitive(value string) error {
	if out, err := exec.Command("copyq", "write", "0",
		"application/x-copyq-hidden", "1",
		"text/plain", value,
	).CombinedOutput(); err != nil {
		return fmt.Errorf("copyq write hidden: %s: %w", string(out), err)
	}
	if out, err := exec.Command("copyq", "select", "0").CombinedOutput(); err != nil {
		return fmt.Errorf("copyq select: %s: %w", string(out), err)
	}
	return nil
}

func (c *Client) SupportsHiddenCopy() bool {
	return true
}
