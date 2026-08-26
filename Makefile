VERSION ?= $(shell git describe --tags --always 2>/dev/null | sed 's/^v//')
# Injected by CI from a secret; falls back to the local git identity for local builds.
GIT_EMAIL ?= $(shell git config user.email 2>/dev/null)

.PHONY: build deb rpm aur aur-pkgbuild aur-x11 aur-x11-pkgbuild

build:
	go build -o browserchooser .

deb: build
	mkdir -p dist
	VERSION=$(VERSION) GIT_EMAIL=$(GIT_EMAIL) nfpm package --packager deb --config packaging/browserchooser.yaml \
		--target dist/browserchooser_$(VERSION)_amd64.deb

rpm: build
	mkdir -p dist
	VERSION=$(VERSION) GIT_EMAIL=$(GIT_EMAIL) nfpm package --packager rpm --config packaging/browserchooser.yaml \
		--target dist/browserchooser-$(VERSION)-1.x86_64.rpm

# Build the Arch packages. Run makepkg directly (needs Arch; on CI it is
# wrapped in an archlinux container). Requires the v<version> tag to exist.
aur: aur-pkgbuild
	cd .build-aur && makepkg -f

# Generate only the PKGBUILD (no compile). Enough for an AUR push.
aur-pkgbuild:
	rm -rf .build-aur
	mkdir -p .build-aur
	sed "s/@PKGVER@/$(VERSION)/; s/@PKGEMAIL@/$(GIT_EMAIL)/" packaging/browserchooser/PKGBUILD > .build-aur/PKGBUILD

aur-x11: aur-x11-pkgbuild
	cd .build-aur-x11 && makepkg -f

aur-x11-pkgbuild:
	rm -rf .build-aur-x11
	mkdir -p .build-aur-x11
	sed "s/@PKGVER@/$(VERSION)/; s/@PKGEMAIL@/$(GIT_EMAIL)/" packaging/browserchooser-x11/PKGBUILD > .build-aur-x11/PKGBUILD
