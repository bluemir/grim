FROM fedora:44 AS build-env

RUN echo "fastestmirror=1" >> /etc/dnf/dnf.conf
RUN dnf install -y \
    make findutils which git-core \
    golang nodejs \
    && dnf clean all

ENV GOPATH=/root/go
ENV PATH=$PATH:/root/go/bin
ENV GOTOOLCHAIN=go1.27.1+auto

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
## VERSION can be passed with --build-arg; when it is unset or empty it is
## derived from `git describe`, so keep .git in the build context.
ARG VERSION

## copy source
COPY . /src

RUN VERSION="${VERSION:-$(git describe --tags --dirty --always)}" make prod

################################################################################
# running image
FROM fedora:44

WORKDIR /
COPY --from=build-env /src/build/grim-* /bin/grim

CMD grim
