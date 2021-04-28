FROM golang:1.15-alpine as builder
RUN mkdir /build
ADD . /build/
WORKDIR /build
RUN CGO_ENABLED=0 go build -a -installsuffix cgo -ldflags '-extldflags "-static"' -o cole .

FROM scratch as production
LABEL maintainer "Daniel Ramirez <dxas90@gmail.com>"
LABEL source "https://github.com/dxas90/cole.git"
COPY --from=builder /build/cole /app/

ENTRYPOINT [ "/app/cole" ]
