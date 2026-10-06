// Package build хранит сведения о версии, подставляемые при сборке:
//
//	go build -ldflags "-X sweepy/internal/build.Version=1.0.0"
package build

// Version — версия релиза. Переопределяется через -ldflags -X.
var Version = "1.0.0"
