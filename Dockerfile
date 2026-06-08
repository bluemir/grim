ARG VERSION=dev
FROM fedora:43 AS build-env

RUN echo "fastestmirror=1" >> /etc/dnf/dnf.conf
RUN dnf install -y \
    make findutils which \
    golang nodejs \
    && dnf clean all

ENV GOPATH=/root/go
ENV PATH=$PATH:/root/go/bin
ENV GOTOOLCHAIN=go1.26.3+auto

# pre build
WORKDIR /src

COPY Makefile ./
COPY scripts/ scripts/

## install build tools
RUN make build-tools 2>/dev/null

## download dependency
COPY go.mod go.sum package.json package-lock.json ./

### go
RUN go mod download
### nodejs
RUN npm install

# build
ARG VERSION

## copy source
COPY . /src

RUN make prod

################################################################################
# running image
FROM fedora:43

WORKDIR /
COPY --from=build-env /src/build/grim-* /bin/grim

CMD grim
