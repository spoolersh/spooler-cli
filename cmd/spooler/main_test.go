package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spoolersh/spooler-go"
)

// The commands are tested end to end in process, with the SDK stubbed by
// fakeClient.

var _ client = (*fakeClient)(nil)

// fakeClient implements client with a function field per method; a nil
// field fails with an error naming the method.
type fakeClient struct {
	DoSend        func(context.Context, spooler.SendRequest) (spooler.SendResult, error)
	DoRecv        func(context.Context, spooler.RecvRequest) (spooler.RecvResult, bool, error)
	DoAck         func(context.Context, spooler.AckRequest) error
	DoNack        func(context.Context, spooler.NackRequest) error
	DoRelease     func(context.Context, spooler.ReleaseRequest) error
	DoFail        func(context.Context, spooler.FailRequest) error
	DoRenew       func(context.Context, spooler.RenewRequest) (spooler.RenewResult, error)
	DoInspect     func(context.Context, spooler.InspectRequest) (spooler.InspectResult, error)
	DoRecover     func(context.Context, spooler.RecoverRequest) error
	DoDiscard     func(context.Context, spooler.DiscardRequest) error
	DoMessages    func(context.Context, spooler.MessagesRequest) (spooler.Page[spooler.Message], error)
	DoSpool       func(context.Context, spooler.SpoolRequest) (spooler.Spool, error)
	DoQueues      func(context.Context, spooler.QueuesRequest) (spooler.Page[spooler.Queue], error)
	DoQueue       func(context.Context, spooler.QueueRequest) (spooler.Queue, error)
	DoCreateQueue func(context.Context, spooler.CreateQueueRequest) (spooler.Queue, error)
	DoUpdateQueue func(context.Context, spooler.UpdateQueueRequest) error
	DoDeleteQueue func(context.Context, spooler.DeleteQueueRequest) error

	// sdk is the SDK client the root's flags built, for a test of them.
	sdk *spooler.Client
}

// notStubbed returns the error for a method no test stubbed.
func notStubbed(method string) error {
	return fmt.Errorf("fake client: %s is not stubbed", method)
}

func (f *fakeClient) Send(ctx context.Context, req spooler.SendRequest) (spooler.SendResult, error) {
	if f.DoSend == nil {
		return spooler.SendResult{}, notStubbed("Send")
	}
	return f.DoSend(ctx, req)
}

func (f *fakeClient) Recv(ctx context.Context, req spooler.RecvRequest) (spooler.RecvResult, bool, error) {
	if f.DoRecv == nil {
		return spooler.RecvResult{}, false, notStubbed("Recv")
	}
	return f.DoRecv(ctx, req)
}

func (f *fakeClient) Ack(ctx context.Context, req spooler.AckRequest) error {
	if f.DoAck == nil {
		return notStubbed("Ack")
	}
	return f.DoAck(ctx, req)
}

func (f *fakeClient) Nack(ctx context.Context, req spooler.NackRequest) error {
	if f.DoNack == nil {
		return notStubbed("Nack")
	}
	return f.DoNack(ctx, req)
}

func (f *fakeClient) Release(ctx context.Context, req spooler.ReleaseRequest) error {
	if f.DoRelease == nil {
		return notStubbed("Release")
	}
	return f.DoRelease(ctx, req)
}

func (f *fakeClient) Fail(ctx context.Context, req spooler.FailRequest) error {
	if f.DoFail == nil {
		return notStubbed("Fail")
	}
	return f.DoFail(ctx, req)
}

func (f *fakeClient) Renew(ctx context.Context, req spooler.RenewRequest) (spooler.RenewResult, error) {
	if f.DoRenew == nil {
		return spooler.RenewResult{}, notStubbed("Renew")
	}
	return f.DoRenew(ctx, req)
}

func (f *fakeClient) Inspect(ctx context.Context, req spooler.InspectRequest) (spooler.InspectResult, error) {
	if f.DoInspect == nil {
		return spooler.InspectResult{}, notStubbed("Inspect")
	}
	return f.DoInspect(ctx, req)
}

func (f *fakeClient) Recover(ctx context.Context, req spooler.RecoverRequest) error {
	if f.DoRecover == nil {
		return notStubbed("Recover")
	}
	return f.DoRecover(ctx, req)
}

func (f *fakeClient) Discard(ctx context.Context, req spooler.DiscardRequest) error {
	if f.DoDiscard == nil {
		return notStubbed("Discard")
	}
	return f.DoDiscard(ctx, req)
}

func (f *fakeClient) Messages(ctx context.Context, req spooler.MessagesRequest) (spooler.Page[spooler.Message], error) {
	if f.DoMessages == nil {
		return spooler.Page[spooler.Message]{}, notStubbed("Messages")
	}
	return f.DoMessages(ctx, req)
}

func (f *fakeClient) Spool(ctx context.Context, req spooler.SpoolRequest) (spooler.Spool, error) {
	if f.DoSpool == nil {
		return spooler.Spool{}, notStubbed("Spool")
	}
	return f.DoSpool(ctx, req)
}

func (f *fakeClient) Queues(ctx context.Context, req spooler.QueuesRequest) (spooler.Page[spooler.Queue], error) {
	if f.DoQueues == nil {
		return spooler.Page[spooler.Queue]{}, notStubbed("Queues")
	}
	return f.DoQueues(ctx, req)
}

func (f *fakeClient) Queue(ctx context.Context, req spooler.QueueRequest) (spooler.Queue, error) {
	if f.DoQueue == nil {
		return spooler.Queue{}, notStubbed("Queue")
	}
	return f.DoQueue(ctx, req)
}

func (f *fakeClient) CreateQueue(ctx context.Context, req spooler.CreateQueueRequest) (spooler.Queue, error) {
	if f.DoCreateQueue == nil {
		return spooler.Queue{}, notStubbed("CreateQueue")
	}
	return f.DoCreateQueue(ctx, req)
}

func (f *fakeClient) UpdateQueue(ctx context.Context, req spooler.UpdateQueueRequest) error {
	if f.DoUpdateQueue == nil {
		return notStubbed("UpdateQueue")
	}
	return f.DoUpdateQueue(ctx, req)
}

func (f *fakeClient) DeleteQueue(ctx context.Context, req spooler.DeleteQueueRequest) error {
	if f.DoDeleteQueue == nil {
		return notStubbed("DeleteQueue")
	}
	return f.DoDeleteQueue(ctx, req)
}

// result holds what one invocation produced.
type result struct {
	stdout string
	stderr string
	code   int
}

// run invokes the CLI in process with fake as the client and returns what
// it produced; stdin, when not empty, is its standard input.
func run(t *testing.T, fake *fakeClient, stdin string, args ...string) result {
	t.Helper()
	return runEnv(t, fake, nil, stdin, args...)
}

// runEnv is run with env in the environment, which otherwise holds no
// SPOOLER_* variable and a fresh HOME and working directory.
func runEnv(t *testing.T, fake *fakeClient, env map[string]string, stdin string, args ...string) result {
	t.Helper()
	return runCtx(t, t.Context(), fake, env, stdin, args...)
}

// runCtx is runEnv under ctx, for a test that cancels it as a signal does.
func runCtx(t *testing.T, ctx context.Context, fake *fakeClient, env map[string]string, stdin string, args ...string) result {
	t.Helper()
	for _, kv := range os.Environ() {
		if name, _, _ := strings.Cut(kv, "="); strings.HasPrefix(name, "SPOOLER_") {
			t.Setenv(name, "")
			os.Unsetenv(name)
		}
	}
	for name, value := range env {
		t.Setenv(name, value)
	}
	dir := t.TempDir()
	t.Setenv("HOME", dir)
	t.Chdir(dir)

	stdout := swapFile(t, &os.Stdout, filepath.Join(dir, ".stdout"), "")
	if stdin != "" {
		swapFile(t, &os.Stdin, filepath.Join(dir, ".stdin"), stdin)
	}
	var stderr bytes.Buffer

	r, cmd := app(func(c *spooler.Client) client {
		fake.sdk = c
		return fake
	})
	r.Stdout = stdout
	r.Stderr = &stderr
	code := r.Run(ctx, cmd, "spooler", args)

	if _, err := stdout.Seek(0, io.SeekStart); err != nil {
		t.Fatalf("seek stdout: %v", err)
	}
	out, err := io.ReadAll(stdout)
	if err != nil {
		t.Fatalf("read stdout: %v", err)
	}
	return result{
		stdout: string(out),
		stderr: stderr.String(),
		code:   code,
	}
}

// swapFile points *std at a file holding content until the test ends.
func swapFile(t *testing.T, std **os.File, name, content string) *os.File {
	t.Helper()
	if err := os.WriteFile(name, []byte(content), 0600); err != nil {
		t.Fatalf("write %s: %v", name, err)
	}
	f, err := os.OpenFile(name, os.O_RDWR, 0)
	if err != nil {
		t.Fatalf("open %s: %v", name, err)
	}
	orig := *std
	*std = f
	t.Cleanup(func() {
		*std = orig
		f.Close()
	})
	return f
}

// expect compares act with exp: stdout, stderr and the exit code, exactly.
func expect(t *testing.T, act, exp result) {
	t.Helper()
	if act.code != exp.code {
		t.Errorf("exit: %d; want %d", act.code, exp.code)
	}
	if act.stdout != exp.stdout {
		t.Errorf("stdout:\n%s\nwant:\n%s", act.stdout, exp.stdout)
	}
	if act.stderr != exp.stderr {
		t.Errorf("stderr:\n%s\nwant:\n%s", act.stderr, exp.stderr)
	}
}

// record returns json pretty-encoded obj.
func record(t *testing.T, obj any) string {
	t.Helper()
	bts, err := json.MarshalIndent(obj, "", "  ")
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	return string(bts) + "\n"
}

// frames returns json-encoded objects one per line.
func frames(t *testing.T, objects ...any) string {
	t.Helper()
	var sb strings.Builder
	for _, v := range objects {
		bts, err := json.Marshal(v)
		if err != nil {
			t.Fatalf("marshal: %v", err)
		}
		sb.Write(bts)
		sb.WriteByte('\n')
	}
	return sb.String()
}

// apiError returns an API error of kind with msg.
func apiError(kind spooler.ErrorKind, msg string) error {
	return &spooler.Error{
		Kind:    kind,
		Message: msg,
	}
}

// readFile returns the content and mode of the file name.
func readFile(t *testing.T, name string) (string, os.FileMode) {
	t.Helper()
	bts, err := os.ReadFile(name)
	if err != nil {
		t.Fatalf("read %s: %v", name, err)
	}
	info, err := os.Stat(name)
	if err != nil {
		t.Fatalf("stat %s: %v", name, err)
	}
	return string(bts), info.Mode().Perm()
}
