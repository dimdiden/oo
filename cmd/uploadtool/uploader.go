package main

import (
	"net/http"

	pb "gopkg.in/cheggaaa/pb.v1"
)

type bars struct {
	*pb.Pool
}

func newBars() *bars {
	return &bars{pb.NewPool()}
}

func (b *bars) filterFunc(r *http.Request) (*http.Request, error) {
	bar := pb.StartNew(int(r.ContentLength)).SetUnits(pb.U_BYTES)
	b.Add(bar)
	reader := bar.NewProxyReader(r.Body)
	r.Body = reader
	return r, nil
}

func (b *bars) deferFunc() {
	b.Stop()
}
