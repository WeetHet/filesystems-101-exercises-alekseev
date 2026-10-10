package docker

import (
	"bufio"
	"context"
	"encoding/json"
	"io"
	"os"
	"path"
	"syscall"
	"time"

	"tests-runner/util/errors"
	"tests-runner/util/ulid"
	dockertypes "github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	dockercontainer "github.com/docker/docker/api/types/container"
	dockerimage "github.com/docker/docker/api/types/image"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/api/types/strslice"
	dockerclient "github.com/docker/docker/client"
	"github.com/docker/docker/pkg/archive"
	"github.com/docker/docker/pkg/stdcopy"
	"github.com/docker/go-units"
	containerspecs "github.com/opencontainers/image-spec/specs-go/v1"
)

func Build(ctx context.Context, dir, dockerfile string, tag string) (err error) {
	defer func() { err = errors.Wrapf(err, "docker.Build(%q)", path.Base(dockerfile)) }()

	d, err := dockerclient.NewClientWithOpts(dockerclient.WithAPIVersionNegotiation())
	if err != nil {
		return err
	}
	defer d.Close()

	tar, err := archive.TarWithOptions(dir, &archive.TarOptions{})
	if err != nil {
		return err
	}
	defer tar.Close()

	resp, err := d.ImageBuild(ctx, tar, dockertypes.ImageBuildOptions{
		Dockerfile: dockerfile,
		Tags:       []string{tag},
	})
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	var errLine string
	var dockerErr struct {
		Error  string `json:"error"`
		Detail struct {
			Message string `json:"message"`
		} `json:"errorDetail"`
	}

	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		errLine = scanner.Text()
	}
	if err = scanner.Err(); err != nil {
		return err
	}

	err = json.Unmarshal([]byte(errLine), &dockerErr)
	if err != nil {
		return errors.ErrInternal.WithMessagef("can't parse a response from docker build")
	}

	if dockerErr.Error != "" {
		return errors.ErrInternal.WithMessagef("docker build failed: %s", dockerErr.Error)
	}

	return nil
}

func Remove(ctx context.Context, tag string) (err error) {
	defer func() { err = errors.Wrap(err, "docker.Remove()") }()

	d, err := dockerclient.NewClientWithOpts(dockerclient.WithAPIVersionNegotiation())
	if err != nil {
		return err
	}
	defer d.Close()

	images, err := d.ImageList(ctx, dockerimage.ListOptions{All: true})
	if err != nil {
		return err
	}

	var id string
	for _, image := range images {
		hasTag := false
		for _, t := range image.RepoTags {
			if t == tag {
				hasTag = true
				break
			}
		}
		if hasTag {
			id = image.ID
			break
		}
	}

	if id == "" {
		return errors.ErrNotFound.WithMessagef("image not found: %q", tag)
	}

	deleteResp, err := d.ImageRemove(ctx, id, dockerimage.RemoveOptions{Force: true})
	if err != nil {
		return err
	}
	_ = deleteResp

	return nil
}

type RunConfig struct {
	Exe       string
	Args      []string
	InputDir  string
	OutputDir string
}

type RunResult struct {
	ExitCode int
	Stdout   string
	Stderr   string
}

func (r *RunResult) WasKilled() bool {
	s := syscall.WaitStatus(r.ExitCode)
	return s.Signaled() && (s.Signal() == syscall.SIGKILL)
}

func Run(ctx context.Context, tag string, conf RunConfig) (r *RunResult, err error) {
	defer func() { err = errors.Wrapf(err, "docker.Run(%q)", tag) }()

	d, err := dockerclient.NewClientWithOpts(dockerclient.WithAPIVersionNegotiation())
	if err != nil {
		return nil, err
	}
	defer d.Close()

	cmd := make([]string, 1+len(conf.Args))
	cmd[0] = conf.Exe
	copy(cmd[1:], conf.Args)

	mounts := make([]mount.Mount, 0)
	if conf.InputDir != "" {
		mounts = append(mounts, mount.Mount{
			Target: "/in",
			Type:   mount.TypeBind,
			Source: conf.InputDir,
		})
	}
	if conf.OutputDir != "" {
		mounts = append(mounts, mount.Mount{
			Target: "/out",
			Type:   mount.TypeBind,
			Source: conf.OutputDir,
		})
	}

	rlimits := container.Resources{
		Memory: 1 * 1024 * 1024 * 1024,
		Ulimits: []*units.Ulimit{
			&units.Ulimit{Name: "nofile", Soft: 2560, Hard: 2560},
			&units.Ulimit{Name: "nproc", Soft: 80, Hard: 80},
			&units.Ulimit{Name: "cpu", Soft: 60, Hard: 60},
		},
	}

	containerCreated, err := d.ContainerCreate(ctx,
		&container.Config{
			AttachStdout:    true,
			AttachStderr:    true,
			Hostname:        "test",
			Domainname:      "vault13",
			Cmd:             strslice.StrSlice(cmd),
			Image:           tag,
			WorkingDir:      "/",
			NetworkDisabled: true,
		},
		&container.HostConfig{
			AutoRemove:     true,
			Mounts:         mounts,
			LogConfig:      container.LogConfig{Type: "journald"},
			NetworkMode:    container.NetworkMode("none"),
			IpcMode:        container.IpcMode("private"),
			PidMode:        container.PidMode(""),
			UTSMode:        container.UTSMode(""),
			UsernsMode:     container.UsernsMode(""),
			Privileged:     true,
			Resources:      rlimits,
			ReadonlyRootfs: true,
			Tmpfs: map[string]string{
				"/tmp": "rw,noexec,nosuid,size=16m",
			},
		},
		&network.NetworkingConfig{},
		&containerspecs.Platform{},
		ulid.New().String())
	if err != nil {
		return nil, err
	}
	ctid := containerCreated.ID

	defer d.ContainerRemove(ctx, ctid, dockercontainer.RemoveOptions{
		RemoveLinks: true,
		Force:       true,
	})
	defer d.ContainerKill(ctx, ctid, "")

	streams, err := d.ContainerAttach(ctx, ctid, dockercontainer.AttachOptions{
		Stream: true,
		Stdout: true,
		Stderr: true,
	})
	if err != nil {
		return nil, err
	}
	defer streams.Close()

	stdoutF, err := os.Create(path.Join(conf.OutputDir, "__stdout"))
	if err != nil {
		return nil, err
	}
	defer stdoutF.Close()

	stderrF, err := os.Create(path.Join(conf.OutputDir, "__stderr"))
	if err != nil {
		return nil, err
	}
	defer stderrF.Close()

	go stdcopy.StdCopy(stdoutF, stderrF, streams.Reader)

	err = d.ContainerStart(ctx, ctid, dockercontainer.StartOptions{})
	if err != nil {
		return nil, err
	}

	containerExited := make(chan struct{})
	go func() {
		select {
		case <-containerExited:
			return
		case <-time.After((60 + 10) * time.Second):
			d.ContainerKill(ctx, ctid, "")
		}
	}()

	resC, errC := d.ContainerWait(ctx, ctid, container.WaitConditionRemoved)
	select {
	case err = <-errC:
		close(containerExited)
		return nil, err

	case res := <-resC:
		close(containerExited)
		r = &RunResult{
			ExitCode: int(res.StatusCode),
			Stdout:   readAll(conf.OutputDir, "__stdout"),
			Stderr:   readAll(conf.OutputDir, "__stderr"),
		}
		return r, nil
	}
}

func readAll(dir, file string) (res string) {
	f, err := os.Open(path.Join(dir, file))
	if err != nil {
		return err.Error()
	}
	defer f.Close()

	r, err := io.ReadAll(io.LimitReader(f, 256*1024))
	if err != nil {
		return err.Error()
	}
	return string(r)
}
