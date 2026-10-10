package main

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"fmt"
	"os"
	"sync"

	"google.golang.org/grpc"

	parhashpb "fs101ex/pkg/gen/parhashsvc"
	"fs101ex/pkg/hash"
	"fs101ex/pkg/parhash"
	"fs101ex/pkg/workgroup"
)

const (
	nrBackends       = 16
	nrBuffersPerTest = 13
)

func main() {
	ctx := context.Background()

	backends, addrs := startBackends(ctx, nrBackends)
	defer func() {
		for _, b := range backends {
			b.Stop()
		}
	}()

	ph := parhash.New(parhash.Config{
		ListenAddr:   "127.0.0.1:0",
		BackendAddrs: addrs,
		Concurrency:  2 * nrBackends,
	})
	defer ph.Stop()

	err := ph.Start(ctx)
	fatalIf(err)

	phConn, err := grpc.Dial(ph.ListenAddr(), grpc.WithInsecure())
	fatalIf(err)
	defer phConn.Close()

	phClient := parhashpb.NewParallelHashSvcClient(phConn)

	wg := sync.WaitGroup{}
	for i := 0; i < nrBackends; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			bufs := make([][]byte, nrBuffersPerTest)
			hashes := make([][]byte, nrBuffersPerTest)
			for i := range bufs {
				bufs[i] = make([]byte, 4096)
				rand.Read(bufs[i])

				h := sha256.Sum256(bufs[i])
				hashes[i] = h[:]
			}

			resp, err := phClient.ParallelHash(ctx, &parhashpb.ParHashReq{
				Data: bufs,
			})
			failIf(err != nil,
				"parhash.Server failed to respond: %v", err)
			failIf(len(resp.Hashes) != nrBuffersPerTest,
				"the number of hashes differs from the number of buffers")

			for i := range hashes {
				failIf(!bytes.Equal(hashes[i], resp.Hashes[i]),
					"hashes do not match")
			}
		}()
	}
	wg.Wait()

	for _, b := range backends {
		failIf(b.NrInvocations != nrBuffersPerTest,
			"sub-requests were not fanned out evenly")
	}

	failIf(workgroup.WasUsed.Load() != true, "workgroup.Go() was not used")
}

func startBackends(ctx context.Context, n int) (backends []*hash.Server, addrs []string) {
	backends = make([]*hash.Server, n)
	addrs = make([]string, n)
	for i := 0; i < n; i++ {
		backends[i] = hash.New(hash.Config{ListenAddr: "127.0.0.1:0"})
		err := backends[i].Start(ctx)
		fatalIf(err)
		addrs[i] = backends[i].ListenAddr()
	}

	return backends, addrs
}

func fatalIf(err error) {
	if err != nil {
		fmt.Fprintf(os.Stderr, "an error should not have occurred: %v\n", err)
		os.Exit(0)
	}
}

func failIf(cond bool, format string, args ...any) {
	if cond {
		fmt.Fprintf(os.Stderr, format, args...)
		fmt.Fprintf(os.Stderr, "\n")
		os.Exit(1)
	}
}
