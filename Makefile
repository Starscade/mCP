.POSIX:

all:

	@                         \
	go clean -testcache    && \
	go mod tidy            && \
	go fmt                 && \
	go test -v -json ./...


