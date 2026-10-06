# Sweepy — сборка, тесты, упаковка
VERSION  ?= 1.0.0
LDFLAGS   = -s -w -X sweepy/internal/build.Version=$(VERSION)

.PHONY: all build test vet clean dist-linux dist-darwin package-windows

all: vet test build

build:
	go build -ldflags "$(LDFLAGS)" -o sweepy .

test:
	go test ./...

vet:
	go vet ./...

# --- релизные сборки ---

dist-linux:
	CGO_ENABLED=1 go build -ldflags "$(LDFLAGS)" -o dist/sweepy-linux-amd64 .

dist-darwin:
	CGO_ENABLED=1 go build -ldflags "$(LDFLAGS)" -o dist/sweepy-darwin-arm64 .

# Windows: нужен mingw-w64 (x86_64-w64-mingw32-gcc).
# Под Linux удобнее через fyne-cross (docker): см. README.
dist-windows:
	CGO_ENABLED=1 GOOS=windows GOARCH=amd64 \
	  go build -ldflags "$(LDFLAGS) -H=windowsgui" -o dist/sweepy-windows-amd64.exe .

# Упаковка для сторов (требует go install fyne.io/fyne/v2/cmd/fyne@latest):
#   fyne package -os windows -name Sweepy -appID com.sweepy.app
package-windows: dist-windows
	fyne package -os windows -name Sweepy -appID com.sweepy.app -appVersion $(VERSION) \
	  -icon Icon.png -executable dist/sweepy-windows-amd64.exe

clean:
	rm -rf dist sweepy
