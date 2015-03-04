package main

import (
	"bytes"
	"fmt"
	"io"
	"net"
	"os/exec"
	"time"
)

type SSHUnixConn struct {
	UserAndHost string
	Socket      string
	cmd         *exec.Cmd
	stdin       io.WriteCloser
	stdout      io.ReadCloser
	stderr      bytes.Buffer
}

func (c *SSHUnixConn) Dial(_, _ string) (net.Conn, error) {
	var err error
	c.cmd = exec.Command("ssh", c.UserAndHost, "socat", "STDIO", "UNIX-CONNECT:"+c.Socket)

	c.stdin, err = c.cmd.StdinPipe()
	if err != nil {
		return nil, err
	}

	c.stdout, err = c.cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}

	c.cmd.Stderr = &c.stderr

	err = c.cmd.Start()
	if err != nil {
		return nil, err
	}
	return c, nil
}

func (c *SSHUnixConn) Read(b []byte) (n int, err error) {
	n, err = c.stdout.Read(b)
	if err != nil && c.stderr.Len() > 0 {
		err = fmt.Errorf("%s\n%s", err.Error(), c.stderr.String())
	}
	return
}

func (c *SSHUnixConn) Write(b []byte) (n int, err error) {
	n, err = c.stdin.Write(b)
	if err != nil && c.stderr.Len() > 0 {
		err = fmt.Errorf("%s\n%s", err.Error(), c.stderr.String())
	}
	return
}

func (c *SSHUnixConn) Close() error {
	// I think this is the right thing to do. It seems to work fine with or
	// without when working correctly or on error. I'm guessing the Transport
	// closes stdin and stdout which is detected by ssh which then terminates
	// itself. Releasing the underlying processes appears to be another option.
	c.cmd.Wait()
	return nil
}

func (c *SSHUnixConn) LocalAddr() net.Addr {
	return nil
}

func (c *SSHUnixConn) RemoteAddr() net.Addr {
	return c
}

func (c *SSHUnixConn) SetDeadline(t time.Time) error {
	return fmt.Errorf("SSHUnixConn.SetDeadline NOT IMPL")
}

func (c *SSHUnixConn) SetReadDeadline(t time.Time) error {
	return fmt.Errorf("SSHUnixConn.SetReadDeadline NOT IMPL")
}

func (c *SSHUnixConn) SetWriteDeadline(t time.Time) error {
	return fmt.Errorf("SSHUnixConn.SetWriteline NOT IMPL")
}

func (c *SSHUnixConn) Network() string {
	return "SSHUNIX"
}

func (c *SSHUnixConn) String() string {
	return fmt.Sprintf("sshunix://%s@%s:%s", c.UserAndHost, c.Socket)
}
