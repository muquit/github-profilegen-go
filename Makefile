#====================================================================
# Mar-29-2025 muquit@muquit.com 
#====================================================================
README_ORIG	  :=./docs/README.md
README		  :=./README.md
BINARY		  :=./github-profilegen-go
GEN_TOC_PROG  :=markdown-toc-go
VERSION       := $(shell cat VERSION)
LDFLAGS 	  := -ldflags "-w -s -X main.version=$(VERSION)"
BUILD_OPTIONS := -trimpath

all: build build_all doc

build:
	echo "*** Compiling ..."
	go build $(BUILD_OPTIONS) $(LDFLAGS) -o $(BINARY)

build_all: clean
	echo "*** Cross Compiling ...."
	go-xbuild-go -build-args '$(BUILD_OPTIONS) $(LDFLAGS)'

# make sure:
#  - to run: make clean
#  - to run: make doc
#  - to check VERSION file
#  - run 'make build_all' before release
#  - release_notes.md exists in cwd
release: check_github_token
	@echo "*** Releasing on github ..."
	go-xbuild-go -release

# check if GITHUB_TOKEN is set and valid, fail the build otherwise
check_github_token:
	@if [ -z "$(GITHUB_TOKEN)" ]; then \
        echo "*** ERROR: GITHUB_TOKEN is not set"; \
        exit 1; \
    fi
	@status=$$(curl -s -o /tmp/check_github_token.$$$$.json -w '%{http_code}' \
        -H "Authorization: token $(GITHUB_TOKEN)" https://api.github.com/user); \
    if [ "$$status" != "200" ]; then \
        echo "*** ERROR: GITHUB_TOKEN is not valid (HTTP $$status)"; \
        cat /tmp/check_github_token.$$$$.json; \
        rm -f /tmp/check_github_token.$$$$.json; \
        exit 1; \
    fi; \
    jq '{login, name, type}' < /tmp/check_github_token.$$$$.json; \
    rm -f /tmp/check_github_token.$$$$.json
	@curl -sI -H "Authorization: token $(GITHUB_TOKEN)" \
        https://api.github.com/user | grep -i x-oauth-scopes

doc:
	echo "*** Generating README.md with TOC ..."
	chmod 600 $(README)
	$(GEN_TOC_PROG) -i $(README_ORIG) -o $(README) -f
	chmod 444 $(README)

clean:
	/bin/rm -f $(BINARY)
	/bin/rm -rf ./bin
