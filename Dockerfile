FROM golang:1.12-alpine AS merlin-build

ENV OUTDIR=/out \
    GO111MODULE=on

COPY . /go/src/github.com/kouzoh/merlin
WORKDIR /go/src/github.com/kouzoh/merlin

RUN apk add --no-cache git && \
    set -eux && \
	mkdir -p "${OUTDIR}" && \
	go mod tidy -v && \
	go mod vendor -v && \
	GOBIN=$OUTDIR CGO_ENABLED=0 go install -a -v -mod=vendor -tags='osusergo netgo static static_build' -ldflags="-d -s -w '-extldflags=-static'" -installsuffix='netgo' github.com/kouzoh/merlin

# target: nonroot
FROM gcr.io/distroless/static:nonroot AS nonroot
COPY --from=merlin-build /out/ /
USER nonroot:nonroot
CMD ["/merlin"]
