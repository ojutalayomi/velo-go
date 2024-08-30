# Build stage
FROM golang:1.18-alpine AS builder

ENV GOPROXY http://proxy.golang.org

RUN mkdir -p /src/velo
WORKDIR /src/velo

# Install Node.js and npm
RUN apk add --no-cache nodejs npm

# Install Buffalo CLI
RUN go install github.com/gobuffalo/cli/cmd/buffalo@latest

# Copy the Go Modules manifests
COPY go.mod go.mod
COPY go.sum go.sum

# Clear npm cache
RUN npm cache clean --force

# Set npm registry
RUN npm config set registry https://registry.npmjs.org/

# cache deps before building and copying source so that we don't need to re-download as much
# and so that source changes don't invalidate our downloaded layer
RUN go mod download

# this will cache the npm install step, unless package.json changes
ADD package.json .
RUN npm install --no-progress

ADD . .
RUN buffalo build --static -o /bin/app

FROM alpine
RUN apk add --no-cache bash
RUN apk add --no-cache ca-certificates

WORKDIR /bin/

COPY --from=builder /bin/app .

# Uncomment to run the binary in "production" mode:
# ENV GO_ENV=production

# Bind the app to 0.0.0.0 so it can be seen from outside the container
ENV ADDR=0.0.0.0

EXPOSE 3000

# Uncomment to run the migrations before running the binary:
# CMD /bin/app migrate; /bin/app
CMD exec /bin/app
